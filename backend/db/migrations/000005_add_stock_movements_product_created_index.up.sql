CREATE INDEX idx_stock_movements_product_created
    ON stock_movements (product_id, created_at DESC);