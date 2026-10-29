package product

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound          = errors.New("product not found")
	ErrDuplicateSKU      = errors.New("sku already exists")
	ErrProductHasHistory = errors.New("product has history")
)

const lastMovementAtCol = `(SELECT MAX(sm.created_at) FROM stock_movements sm WHERE sm.product_id = products.id)`

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func scanProduct(row interface{ Scan(...any) error }, p *Product) error {
	return row.Scan(&p.ID, &p.SKU, &p.Name, &p.Price, &p.Stock, &p.IsActive, &p.CreatedAt, &p.LastMovementAt)
}

func (r *Repository) List(ctx context.Context, search, status string, limit int, offset int) ([]Product, int, error) {
	const listWhere = `WHERE ($1 = '' OR name ILIKE '%' || $1 || '%' OR sku ILIKE '%' || $1 || '%')
	AND ($2 = ''
		OR ($2 = 'active' AND is_active)
		OR ($2 = 'inactive' AND NOT is_active)
		OR ($2 = 'low_stock' AND stock <= 10))`

	const countQuery = `SELECT COUNT(*) FROM products ` + listWhere

	var total int

	if err := r.db.QueryRowContext(ctx, countQuery, search, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	const query = `
	SELECT 
	id, sku, name, price, stock, is_active, created_at, ` + lastMovementAtCol + `
	 FROM products ` + listWhere + ` ORDER BY id DESC
	LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(ctx, query, search, status, limit, offset)

	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}

	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		var p Product
		if err := scanProduct(rows, &p); err != nil {
			return nil, 0, fmt.Errorf("scan products: %w", err)
		}

		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate products: %w", err)
	}

	return products, total, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Product, error) {
	const query = `SELECT id, sku, name, price, stock, is_active, created_at, ` + lastMovementAtCol + ` FROM products WHERE id = $1`

	var p Product

	err := scanProduct(r.db.QueryRowContext(ctx, query, id), &p)

	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrNotFound
	}

	if err != nil {
		return Product{}, fmt.Errorf("get product %d: %w", id, err)
	}

	return p, nil
}

func (r *Repository) Create(ctx context.Context, in CreateProductInput) (Product, error) {
	const query = `
	INSERT INTO products (sku, name, price) 
	VALUES ($1, $2, $3) 
	RETURNING id, sku, name, price, stock, is_active, created_at, ` + lastMovementAtCol

	var p Product

	err := scanProduct(r.db.QueryRowContext(ctx, query, in.SKU, in.Name, in.Price), &p)

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Product{}, ErrDuplicateSKU
	}

	if err != nil {
		return Product{}, fmt.Errorf("create product: %w", err)
	}

	return p, nil
}

func (r *Repository) Update(ctx context.Context, id int64, in UpdateProductInput) (Product, error) {
	const query = `
	UPDATE products SET 
	name = $1,
	price = $2,
	is_active = $3
	WHERE id = $4
	RETURNING id, sku, name, price, stock, is_active, created_at, ` + lastMovementAtCol

	var p Product

	err := scanProduct(r.db.QueryRowContext(ctx, query, in.Name, in.Price, *in.IsActive, id), &p)

	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrNotFound
	}

	if err != nil {
		return Product{}, fmt.Errorf("update product %d: %w", id, err)
	}

	return p, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM products WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23503" {
		return ErrProductHasHistory
	}

	if err != nil {
		return fmt.Errorf("delete product %d: %w", id, err)
	}

	n, err := res.RowsAffected()

	if err != nil {
		return fmt.Errorf("delete product %d: %w", id, err)
	}

	if n == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) AdjustStock(ctx context.Context, productID int64, in AdjustStockInput) (AdjustStockResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)

	if err != nil {
		return AdjustStockResult{}, fmt.Errorf("begin: %w", err)
	}

	defer tx.Rollback()

	delta := in.Qty

	if in.Type == "out" {
		delta = -in.Qty
	}

	const queryUpdate = `UPDATE products SET stock = stock + $1 WHERE id = $2 AND stock + $1 >= 0 RETURNING stock`

	var newStock int

	err = tx.QueryRowContext(ctx, queryUpdate, delta, productID).Scan(&newStock)

	if errors.Is(err, sql.ErrNoRows) {
		var current int
		err := tx.QueryRowContext(ctx, `SELECT stock FROM products WHERE id = $1`, productID).Scan(&current)

		if errors.Is(err, sql.ErrNoRows) {
			return AdjustStockResult{}, ErrNotFound
		}

		if err != nil {
			return AdjustStockResult{}, fmt.Errorf("read stock %d: %w", productID, err)
		}

		return AdjustStockResult{}, &InsufficientStockError{Available: current}
	}

	if err != nil {
		return AdjustStockResult{}, fmt.Errorf("adjust stock %d: %w", productID, err)
	}

	const insertQuery = `
	INSERT INTO stock_movements 
	(product_id, type, qty, note)
	VALUES
	($1, $2, $3, NULLIF($4, ''))
	RETURNING id, product_id, type, qty, note, created_at`

	var m StockMovement
	err = tx.QueryRowContext(ctx, insertQuery, productID, in.Type, in.Qty, in.Note).Scan(&m.ID, &m.ProductID, &m.Type, &m.Qty, &m.Note, &m.CreatedAt)

	if err != nil {
		return AdjustStockResult{}, fmt.Errorf("insert movement %d: %w", productID, err)
	}

	if err := tx.Commit(); err != nil {
		return AdjustStockResult{}, fmt.Errorf("commit: %w", err)
	}

	return AdjustStockResult{Movement: m, Stock: newStock}, nil
}

func (r *Repository) ListMovements(ctx context.Context, productID int64) ([]StockMovement, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM products WHERE id = $1)`, productID).Scan(&exists)

	if err != nil {
		return nil, fmt.Errorf("exists product %d: %w", productID, err)
	}

	if !exists {
		return nil, ErrNotFound
	}

	const q = `
	SELECT id, product_id, type, qty, note, created_at
	FROM stock_movements
	WHERE product_id = $1
	ORDER BY created_at DESC, id DESC`

	rows, err := r.db.QueryContext(ctx, q, productID)

	if err != nil {
		return nil, fmt.Errorf("list movements %d: %w", productID, err)
	}

	defer rows.Close()

	movements := []StockMovement{}

	for rows.Next() {
		var m StockMovement
		if err := rows.Scan(&m.ID, &m.ProductID, &m.Type, &m.Qty, &m.Note, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan movements: %w", err)
		}

		movements = append(movements, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate movements: %w", err)
	}

	return movements, nil
}
