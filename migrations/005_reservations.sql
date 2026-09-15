CREATE TABLE stock_reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sales_order_id UUID NOT NULL,
    product_id VARCHAR(50) NOT NULL,
    warehouse_id UUID REFERENCES warehouses(id) ON DELETE RESTRICT,
    quantity NUMERIC(15,4) NOT NULL CHECK (quantity > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'RELEASED', 'CONFIRMED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_stock_reservation_order_product UNIQUE (sales_order_id, product_id)
);

CREATE INDEX idx_stock_reservations_open ON stock_reservations (product_id, status) WHERE status = 'OPEN';
CREATE INDEX idx_stock_reservations_order ON stock_reservations (sales_order_id, status);

UPDATE stock_balances SET quantity_reserved = 0;
