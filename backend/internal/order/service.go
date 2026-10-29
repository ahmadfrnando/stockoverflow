package order

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

func (s *Service) Create(ctx context.Context, in CreateOrderInput) (Order, error) {
	in.Customer = strings.TrimSpace(in.Customer)

	fields := map[string]string{}
	if in.Customer == "" {
		fields["customer"] = "customer is required"
	}
	if len(in.Items) == 0 {
		fields["items"] = "items must not be empty"
	}

	seen := map[string]bool{}

	for i := range in.Items {
		in.Items[i].SKU = strings.ToUpper(strings.TrimSpace(in.Items[i].SKU))
		sku := in.Items[i].SKU
		skuKey := fmt.Sprintf("items[%d].sku", i)
		if in.Items[i].SKU == "" {
			fields[skuKey] = "sku is required"
		} else if seen[sku] {
			fields[skuKey] = "duplicate sku"
		}
		seen[sku] = true
		if in.Items[i].Qty <= 0 {
			fields[fmt.Sprintf("items[%d].qty", i)] = "qty must be greater than 0"
		}
	}

	if len(fields) > 0 {
		return Order{}, &ValidationError{Fields: fields}
	}

	return s.repo.Create(ctx, in)
}
