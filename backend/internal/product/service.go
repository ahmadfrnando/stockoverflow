package product

import (
	"context"
	"strings"
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "validation failed"
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, search string, page, limit int) (ListResult, error) {
	search = strings.TrimSpace(search)

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

	products, total, err := s.repo.List(ctx, search, limit, offset)

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
	if in.Stock < 0 {
		fields["stock"] = "stock can not negative"
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
		fields["price"] = "price must be more than 0"
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
