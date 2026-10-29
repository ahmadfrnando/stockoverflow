package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrNotFound = errors.New("order not found")
)

type InsufficientStockError struct {
	SKU       string
	Available int
}

func (e *InsufficientStockError) Error() string {
	return fmt.Sprintf("insufficient stock for %s: available %d", e.SKU, e.Available)
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, search string, limit int, offset int) ([]Order, int, error) {
	const listWhere = `WHERE ($1 = '' OR customer ILIKE '%' || $1 || '%')`

	const countQuery = `SELECT COUNT(*) FROM sales_orders ` + listWhere

	var total int

	if err := r.db.QueryRowContext(ctx, countQuery, search).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	const query = `
	SELECT 
	id, customer, status, created_at, cancelled_at
	FROM sales_orders ` + listWhere + ` ORDER BY id DESC
	LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, search, limit, offset)

	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}

	defer rows.Close()

	var o Order

	orders := []Order{}
	for rows.Next() {
		var p Order
		if err := rows.Scan(&o.ID, &o.Customer, &o.Total, &o.CreatedAt, &o.CanclelledAt); err != nil {
			return nil, 0, fmt.Errorf("scan orders: %w", err)
		}

		orders = append(orders, p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate orders: %w", err)
	}

	return orders, total, nil
}

func (r *Repository) Create(ctx context.Context, in CreateOrderInput) (Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Order{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	skus := make([]string, 0, len(in.Items))

	for _, item := range in.Items {
		skus = append(skus, item.SKU)
	}

	const querySkuCheck = `SELECT sku, is_active FROM products WHERE sku = ANY($1)`

	active := map[string]bool{}
	rows, err := tx.QueryContext(ctx, querySkuCheck, skus)

	if err != nil {
		return Order{}, fmt.Errorf("check skus: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var sku string
		var isActive bool

		if err := rows.Scan(&sku, &isActive); err != nil {
			return Order{}, fmt.Errorf("scan product: %w", err)
		}

		active[sku] = isActive
	}

	if err := rows.Err(); err != nil {
		return Order{}, fmt.Errorf("iterate product: %w", err)
	}

	fields := map[string]string{}

	for i, item := range in.Items {
		key := fmt.Sprintf("items[%d].sku", i)

		isActive, ok := active[item.SKU]
		if !ok {
			fields[key] = "unknown sku"
		} else if !isActive {
			fields[key] = "product is inactive"
		}
	}

	if len(fields) > 0 {
		return Order{}, &ValidationError{Fields: fields}
	}

	const queryUpdateProduct = `UPDATE products SET stock = stock - $1 WHERE sku = $2 AND stock >= $1 RETURNING price`
	total := 0
	items := make([]Item, 0, len(in.Items))
	for _, item := range in.Items {
		var unitPrice int
		err := tx.QueryRowContext(ctx, queryUpdateProduct, item.Qty, item.SKU).Scan(&unitPrice)

		if errors.Is(err, sql.ErrNoRows) {
			var available int
			if err := tx.QueryRowContext(ctx, `SELECT stock FROM products WHERE sku = $1`, item.SKU).Scan(&available); err != nil {
				return Order{}, fmt.Errorf("read stock %s: %w", item.SKU, err)
			}
			return Order{}, &InsufficientStockError{SKU: item.SKU, Available: available}
		}
		if err != nil {
			return Order{}, fmt.Errorf("decrement %s: %w", item.SKU, err)
		}

		total += item.Qty * unitPrice
		items = append(items, Item{SKU: item.SKU, Qty: item.Qty, UnitPrice: unitPrice})
	}

	const queryInsertOrder = `INSERT INTO sales_orders (customer, total) VALUES ($1, $2) RETURNING id, status, created_at`
	o := Order{Customer: in.Customer, Total: total}

	err = tx.QueryRowContext(ctx, queryInsertOrder, in.Customer, total).Scan(&o.ID, &o.Status, &o.CreatedAt)

	if err != nil {
		return Order{}, fmt.Errorf("created order: %w", err)
	}

	const queryInsertItem = `INSERT INTO sales_order_items (order_id, sku, qty, unit_price) VALUES ($1, $2, $3, $4) RETURNING id`

	const queryMovement = `
	INSERT INTO stock_movements 
	(product_id, type, qty, note) 
	VALUES ((SELECT id FROM products WHERE sku = $1), $2, $3, $4)`

	note := fmt.Sprintf("order #%d", o.ID)

	for i := range items {
		err := tx.QueryRowContext(ctx, queryInsertItem, o.ID, items[i].SKU, items[i].Qty, items[i].UnitPrice).Scan(&items[i].ID)
		if err != nil {
			return Order{}, fmt.Errorf("create item order of sku %s: %w", items[i].SKU, err)
		}

		_, err = tx.ExecContext(ctx, queryMovement, items[i].SKU, "out", items[i].Qty, note)

		if err != nil {
			return Order{}, fmt.Errorf("insert movement %s: %w", items[i].SKU, err)
		}
	}

	o.Items = items

	if err := tx.Commit(); err != nil {
		return Order{}, fmt.Errorf("commit: %w", err)
	}
	return o, nil
}
