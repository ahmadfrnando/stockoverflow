package product

import (
	"context"
	"fmt"
	"strings"
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "validation failed"
}

type InsufficientStockError struct {
	SKU       string
	Available int
}

func (e *InsufficientStockError) Error() string {
	return fmt.Sprintf("insufficient stock: available %d", e.Available)
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, search, status string, page, limit int) (ListResult, error) {
	search = strings.TrimSpace(search)

	switch status {
	case "", "active", "inactive", "low_stock":
	default:
		return ListResult{}, &ValidationError{Fields: map[string]string{
			"status": "must be one of active, inactive, low_stock",
		}}
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	products, total, err := s.repo.List(ctx, search, status, limit, offset)

	if err != nil {
		return ListResult{}, err
	}

	return ListResult{
		Data:  products,
		Page:  page,
		Limit: limit,
		Total: total,
	}, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (Product, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, in CreateProductInput) (Product, error) {
	in.SKU = strings.ToUpper(strings.TrimSpace(in.SKU))
	in.Name = strings.TrimSpace(in.Name)

	fields := map[string]string{}
	if in.SKU == "" {
		fields["sku"] = "sku is required"
	}
	if in.Name == "" {
		fields["name"] = "name is required"
	}
	if in.Price < 0 {
		fields["price"] = "price must be 0 or greater"
	}

	if len(fields) > 0 {
		return Product{}, &ValidationError{Fields: fields}
	}

	return s.repo.Create(ctx, in)
}

func (s *Service) Update(ctx context.Context, id int64, in UpdateProductInput) (Product, error) {
	in.Name = strings.TrimSpace(in.Name)

	fields := map[string]string{}

	if in.Name == "" {
		fields["name"] = "name is required"
	}
	if in.Price < 0 {
		fields["price"] = "price must be 0 or greater"
	}
	if in.IsActive == nil {
		fields["is_active"] = "status active is required"
	}

	if len(fields) > 0 {
		return Product{}, &ValidationError{Fields: fields}
	}

	return s.repo.Update(ctx, id, in)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) AdjustStock(ctx context.Context, productID int64, in AdjustStockInput) (AdjustStockResult, error) {
	in.Type = strings.ToLower(strings.TrimSpace(in.Type))
	in.Note = strings.TrimSpace(in.Note)

	fields := map[string]string{}
	if in.Type != "in" && in.Type != "out" {
		fields["type"] = "type must be 'in' or 'out'"
	}
	if in.Type == "" {
		fields["type"] = "type is required"
	}
	if in.Qty <= 0 {
		fields["qty"] = "qty must be greater than 0"
	}

	if len(fields) > 0 {
		return AdjustStockResult{}, &ValidationError{Fields: fields}
	}

	return s.repo.AdjustStock(ctx, productID, in)
}

func (s *Service) ListMovements(ctx context.Context, id int64) ([]StockMovement, error) {
	return s.repo.ListMovements(ctx, id)
}

func (s *Service) Summary(ctx context.Context) (Summary, error) {
	return s.repo.Summary(ctx)
}

