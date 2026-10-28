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

func (r *Repository) List(ctx context.Context, search string, limit int, offset int) ([]Product, int, error) {
	const listWhere = `WHERE ($1 = '' OR name ILIKE '%' || $1 || '%' OR sku ILIKE '%' || $1 || '%')`

	const countQuery = `SELECT COUNT(*) FROM products ` + listWhere

	var total int

	if err := r.db.QueryRowContext(ctx, countQuery, search).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	const query = `
	SELECT 
	id, sku, name, price, stock, is_active, created_at, ` + lastMovementAtCol + `
	 FROM products ` + listWhere + ` ORDER BY id DESC
	LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, search, limit, offset)

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
