package main

import (
	"log"
	"net/http"
	"os"

	"stockoverflow/internal/database"
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

	repo := product.NewRepository(db)
	productService := product.NewService(repo)
	productHandler := product.NewHandler(productService)

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

	port := getenv("APP_PORT", "8080")

	log.Printf("listening on: %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
