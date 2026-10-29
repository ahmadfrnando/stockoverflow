package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
)

var (
	ErrNotFound         = errors.New("order not found")
	ErrAlreadyCancelled = errors.New("order already cancelled")
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
	sorted := slices.Clone(in.Items)
	slices.SortFunc(sorted, func(a, b Item) int {
		return strings.Compare(a.SKU, b.SKU)
	})

	for _, item := range sorted {
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

func (r *Repository) GetByID(ctx context.Context, id int64) (Order, error) {
	const queryOrder = `
	SELECT 
	id, customer, status, total, created_at, cancelled_at 
	FROM sales_orders 
	WHERE id = $1`

	var o Order

	err := r.db.QueryRowContext(ctx, queryOrder, id).Scan(&o.ID, &o.Customer, &o.Status, &o.Total, &o.CreatedAt, &o.CancelledAt)

	if errors.Is(err, sql.ErrNoRows) {
		return Order{}, ErrNotFound
	}

	if err != nil {
		return Order{}, fmt.Errorf("get orders %d: %w", id, err)
	}

	const queryOrderItems = `
	SELECT id, sku, qty, unit_price
	FROM sales_order_items
	WHERE order_id = $1
	ORDER BY id`

	rows, err := r.db.QueryContext(ctx, queryOrderItems, id)

	if err != nil {
		return Order{}, fmt.Errorf("list items: %w", err)
	}

	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var i Item
		if err := rows.Scan(&i.ID, &i.SKU, &i.Qty, &i.UnitPrice); err != nil {
			return Order{}, fmt.Errorf("scan items: %w", err)
		}

		items = append(items, i)
	}

	if err := rows.Err(); err != nil {
		return Order{}, fmt.Errorf("iterate order: %w", err)
	}

	o.Items = items

	return o, nil
}

func (r *Repository) Cancel(ctx context.Context, id int64) (Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Order{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	const queryUpdate = `
	UPDATE sales_orders 
	SET status = 'CANCELLED', cancelled_at = now()
	WHERE id = $1
	AND status != 'CANCELLED'
	RETURNING id`

	var cancelledID int64
	err = tx.QueryRowContext(ctx, queryUpdate, id).Scan(&cancelledID)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM sales_orders WHERE id = $1)`, id).Scan(&exists); err != nil {
			return Order{}, fmt.Errorf("check order %d: %w", id, err)
		}
		if !exists {
			return Order{}, ErrNotFound
		}

		return Order{}, ErrAlreadyCancelled
	}

	if err != nil {
		return Order{}, fmt.Errorf("cancel order %d: %w", id, err)
	}

	const queryItems = `
	SELECT sku, qty
	FROM sales_order_items
	WHERE order_id = $1
	ORDER BY sku`

	rows, err := tx.QueryContext(ctx, queryItems, id)

	if err != nil {
		return Order{}, fmt.Errorf("list items %d: %w", id, err)
	}

	defer rows.Close()

	items := []Item{}

	for rows.Next() {
		var i Item
		if err := rows.Scan(&i.SKU, &i.Qty); err != nil {
			return Order{}, fmt.Errorf("scan item: %w", err)
		}

		items = append(items, i)
	}

	if err := rows.Err(); err != nil {
		return Order{}, fmt.Errorf("iterate items: %w", err)
	}

	const queryRestore = `UPDATE products SET stock = stock + $1 WHERE sku = $2`

	const queryMovement = `
	INSERT INTO stock_movements (product_id, type, qty, note)
	VALUES ((SELECT id FROM products WHERE sku = $1), 'in', $2, $3)`

	note := fmt.Sprintf("cancel order #%d", id)

	for _, item := range items {
		if _, err := tx.ExecContext(ctx, queryRestore, item.Qty, item.SKU); err != nil {
			return Order{}, fmt.Errorf("restore stock %s: %w", item.SKU, err)
		}

		if _, err := tx.ExecContext(ctx, queryMovement, item.SKU, item.Qty, note); err != nil {
			return Order{}, fmt.Errorf("insert movements %s: %w", item.SKU, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Order{}, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, id)
}
