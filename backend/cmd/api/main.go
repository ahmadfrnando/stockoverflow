package main

import (
	"log"
	"net/http"
	"os"

	"stockoverflow/internal/database"
	"stockoverflow/internal/order"
	"stockoverflow/internal/product"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

func main() {
	db, err := database.Open()
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	defer db.Close()

	productRepo := product.NewRepository(db)
	productService := product.NewService(productRepo)
	productHandler := product.NewHandler(productService)
	orderRepo := order.NewRepository(db)
	orderService := order.NewService(orderRepo)
	orderHandler := order.NewHandler(orderService)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /api/products", productHandler.List)
	mux.HandleFunc("GET /api/products/{id}", productHandler.Show)
	mux.HandleFunc("POST /api/products", productHandler.Create)
	mux.HandleFunc("PUT /api/products/{id}", productHandler.Update)
	mux.HandleFunc("DELETE /api/products/{id}", productHandler.Delete)
	mux.HandleFunc("POST /api/products/{id}/stock", productHandler.AdjustStock)
	mux.HandleFunc("GET /api/products/{id}/movements", productHandler.Movements)
	// mux.HandleFunc("GET /api/orders", orderHandler.List)
	mux.HandleFunc("POST /api/orders", orderHandler.Create)
	mux.HandleFunc("GET /api/orders/{id}", orderHandler.Show)
	mux.HandleFunc("POST /api/orders/{id}/cancel", orderHandler.Cancel)

	port := getenv("APP_PORT", "8080")

	log.Printf("listening on: %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
