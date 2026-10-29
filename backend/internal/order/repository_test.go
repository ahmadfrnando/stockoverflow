package order

import (
	"context"
	"errors"
	"testing"
	"sync"

	"stockoverflow/internal/testutil"
)

func TestCreate_InsufficientStock_WritesNothing(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO products (sku, name, price, stock) VALUES
		('SKU-A', 'Produk A', 10000, 10),
		('SKU-B', 'Produk B', 20000, 2)`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := repo.Create(ctx, CreateOrderInput{
		Customer: "Nando",
		Items: []Item{
			{SKU: "SKU-A", Qty: 2},
			{SKU: "SKU-B", Qty: 5},
		},
	})

	ise, ok := errors.AsType[*InsufficientStockError](err)
	if !ok {
		t.Fatalf("want InsufficientStockError, got %v", err)
	}
	if ise.SKU != "SKU-B" || ise.Available != 2 {
		t.Errorf("got sku=%s available=%d, want sku=SKU-B available=2", ise.SKU, ise.Available)
	}

	want := map[string]int{"SKU-A": 10, "SKU-B": 2}

	for sku, wantStock := range want {
		var stock int
		if err := db.QueryRow(`SELECT stock FROM products WHERE sku = $1`, sku).Scan(&stock); err != nil {
			t.Fatalf("read stock %s: %v", sku, err)
		}
		if stock != wantStock {
			t.Errorf("%s stock = %d, want %d", sku, stock, wantStock)
		}
	}

	for _, table := range []string{"sales_orders", "sales_order_items", "stock_movements"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}

		if count != 0 {
			t.Errorf("%s count = %d, want 0", table, count)
		}
	}
}

func TestCreate_HappyPath_UpdatesStockAndWritesMovements(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO products (sku, name, price, stock) VALUES
		('SKU-A', 'Produk A', 10000, 10),
		('SKU-B', 'Produk B', 20000, 2)`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// act: SKU-A qty 2, SKU-B qty 1
	o, err := repo.Create(ctx, CreateOrderInput{
		Customer: "nando",
		Items: []Item{
			{SKU: "SKU-A", Qty: 2},
			{SKU: "SKU-B", Qty: 1},
		},
	})

	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	
	if o.Status != "CREATED" {
		t.Errorf("status = %s, want CREATED", o.Status)
	}
	if o.Total != 40000 {
		t.Errorf("total = %d, want 40000", o.Total)
	}
	if len(o.Items) != 2 {
		t.Errorf("items = %d, want 2", len(o.Items))
	}

	wants := map[string]int{"SKU-A": 8, "SKU-B": 1}
	for sku, wantStock := range wants {
		var stock int
		if err := db.QueryRow(`SELECT stock FROM products WHERE sku = $1`, sku).Scan(&stock); err != nil {
			t.Fatalf("read stock %s: %v", sku, err)
		}
		if stock != wantStock {
			t.Errorf("%s stock = %d, want %d", sku, stock, wantStock)
		}
	}

	var movementCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM stock_movements WHERE type = 'out'`).Scan(&movementCount); err != nil {
		t.Fatalf("count movements: %v", err)
	}

	if movementCount != 2 {
		t.Errorf("movements = %d, want 2", movementCount)
	}

	var itemCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sales_order_items`).Scan(&itemCount); err != nil {
		t.Fatalf("count items: %v", err)
	}

	if itemCount != 2 {
		t.Errorf("order items = %d, want 2", itemCount)
	}
}

func TestCreate_Concurrent_NoOversell(t *testing.T) {
	db := testutil.OpenTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO products (sku, name, price, stock) VALUES ('SKU-A', 'product a', 10000, 5)`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	const workers = 10
	errs := make(chan error, workers)

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			_, err := repo.Create(ctx, CreateOrderInput{
				Customer: "nando",
				Items: []Item{{SKU: "SKU-A", Qty: 1}},
			})
			errs <- err
		})
	}
	wg.Wait()
	close(errs)

	success, insufficient := 0, 0
	for err := range errs {
		if err == nil {
			success++
			continue
		}
		if _, ok := errors.AsType[*InsufficientStockError](err); ok {
			insufficient++
			continue
		}
		t.Errorf("unexpected error: %v", err)
	}

	if success != 5 || insufficient != 5 {
		t.Errorf("success=%d insufficient=%d, want 5 and 5", success, insufficient)
	}

	var stock int
	if err := db.QueryRow(`SELECT stock FROM products WHERE sku = 'SKU-A'`).Scan(&stock); err != nil{
		t.Fatalf("read stock: %v", err)
	}
	if stock != 0 {
		t.Errorf("stock %d, want 0", stock)
	}

	var orderCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sales_orders`).Scan(&orderCount); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if orderCount != 5 {
		t.Errorf("orders = %d, want 5", orderCount)
	}
}
