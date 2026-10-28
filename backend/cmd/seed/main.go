package main

import (
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"stockoverflow/internal/database"
)

type productSeed struct {
	SKU      string
	Name     string
	Price    int
	Stock    int
	IsActive bool
}

var products = []productSeed{
	{SKU: "SKU-001", Name: "Kopi Arabica Gayo 250g", Price: 85000, Stock: 50, IsActive: true},
	{SKU: "SKU-002", Name: "Teh Hijau Celup 25 Pcs", Price: 18500, Stock: 120, IsActive: true},
	{SKU: "SKU-003", Name: "Gula Aren Bubuk 500g", Price: 32000, Stock: 40, IsActive: true},
	{SKU: "SKU-004", Name: "Beras Pandan Wangi 5kg", Price: 78000, Stock: 35, IsActive: true},
	{SKU: "SKU-005", Name: "Minyak Goreng 2L", Price: 38500, Stock: 60, IsActive: true},
	{SKU: "SKU-006", Name: "Gula Pasir 1kg", Price: 17500, Stock: 90, IsActive: true},
	{SKU: "SKU-007", Name: "Telur Ayam Negeri 1kg", Price: 29000, Stock: 25, IsActive: true},
	{SKU: "SKU-008", Name: "Tepung Terigu Serbaguna 1kg", Price: 13500, Stock: 70, IsActive: true},
	{SKU: "SKU-009", Name: "Susu UHT Full Cream 1L", Price: 19500, Stock: 48, IsActive: true},
	{SKU: "SKU-010", Name: "Mie Instan Goreng 5 Pcs", Price: 16000, Stock: 0, IsActive: true},
	{SKU: "SKU-011", Name: "Kecap Manis 520ml", Price: 24000, Stock: 0, IsActive: true},
	{SKU: "SKU-012", Name: "Garam Beryodium 250g", Price: 4500, Stock: 150, IsActive: true},
	{SKU: "SKU-013", Name: "Sirup Cocopandan 460ml", Price: 22000, Stock: 0, IsActive: false},
}

func main() {
	db, err := database.Open()

	if err != nil {
		log.Fatalf("connection database: %v", err)
	}

	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}

	defer tx.Rollback()

	const q = `INSERT INTO products (sku, name, price, stock, is_active) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (sku) DO NOTHING`
	const queryStock = `
			INSERT INTO stock_movements
			(product_id, type, qty, note)
			VALUES
			((SELECT id FROM products where SKU = $1), $2, $3, $4)`

	inserted := 0
	movement := 0

	for _, p := range products {
		res, err := tx.Exec(q, p.SKU, p.Name, p.Price, p.Stock, p.IsActive)

		if err != nil {
			log.Fatalf("insert products %s: %v", p.SKU, err)
		}
		n, _ := res.RowsAffected()

		if n == 1 && p.Stock > 0 {
			_, err := tx.Exec(queryStock, p.SKU, "in", p.Stock, "Opening Balance")

			if err != nil {
				log.Fatalf("insert stock movements: %v", err)
			}

			movement += int(n)
		} 
		inserted += int(n)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("commit: %v", err)
	}

	log.Printf("seed done: %d new products, %d skip products, %d movement stock", inserted, len(products)-inserted, movement)
}
