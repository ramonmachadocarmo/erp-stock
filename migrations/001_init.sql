CREATE TABLE warehouses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE stock_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id VARCHAR(50) NOT NULL,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    quantity_available NUMERIC(15,4) NOT NULL DEFAULT 0 CHECK (quantity_available >= 0),
    quantity_reserved NUMERIC(15,4) NOT NULL DEFAULT 0 CHECK (quantity_reserved >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_stock_balance_product_warehouse UNIQUE (product_id, warehouse_id)
);

CREATE TABLE stock_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id VARCHAR(50) NOT NULL,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    movement_type VARCHAR(30) NOT NULL,
    quantity NUMERIC(15,4) NOT NULL,
    reference_doc_type VARCHAR(30) NOT NULL,
    reference_doc_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_stock_balances_product ON stock_balances(product_id);
CREATE INDEX idx_stock_balances_warehouse ON stock_balances(warehouse_id);
CREATE INDEX idx_stock_movements_product_wh ON stock_movements(product_id, warehouse_id, created_at DESC);
CREATE INDEX idx_stock_movements_ref_doc ON stock_movements(reference_doc_type, reference_doc_id);
