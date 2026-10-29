package order

import (
	"time"
)

type ListResult struct {
	Data  []Order `json:"data"`
	Page  int     `json:"page"`
	Limit int     `json:"limit"`
	Total int     `json:"total"`
}

type Order struct {
	ID           int64     `json:"id"`
	Customer     string    `json:"customer"`
	Status       string    `json:"status"`
	Total        int       `json:"total"`
	CreatedAt    time.Time `json:"created_at"`
	CanclelledAt time.Time `json:"cancelled_at"`
	Items        []Item    `json:"items"`
}

type CreateOrderInput struct {
	Customer string `json:"customer"`
	Items    []Item `json:"items"`
}

type Item struct {
	ID        int64  `json:"id"`
	SKU       string `json:"sku"`
	Qty       int    `json:"qty"`
	UnitPrice int    `json:"unit_price"`
}

// type UpdateProductInput struct {
// 	Name     string `json:"name"`
// 	Price    int    `json:"price"`
// 	IsActive *bool  `json:"is_active"`
// }
