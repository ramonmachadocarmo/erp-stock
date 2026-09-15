CREATE TABLE sale_price_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id VARCHAR(50) NOT NULL,
    sku VARCHAR(50),
    previous_price NUMERIC(15,4) NOT NULL DEFAULT 0,
    new_price NUMERIC(15,4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE purchase_price_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id VARCHAR(50) NOT NULL,
    sku VARCHAR(50),
    previous_price NUMERIC(15,4) NOT NULL DEFAULT 0,
    new_price NUMERIC(15,4) NOT NULL,
    reference_doc_id VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE assemblies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(30) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    product_id VARCHAR(50),
    margin_percent NUMERIC(8,4) NOT NULL DEFAULT 0,
    cost NUMERIC(15,4) NOT NULL DEFAULT 0,
    suggested_price NUMERIC(15,4) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE assembly_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assembly_id UUID NOT NULL REFERENCES assemblies(id) ON DELETE CASCADE,
    product_id VARCHAR(50) NOT NULL,
    quantity NUMERIC(15,4) NOT NULL CHECK (quantity > 0),
    role VARCHAR(20) NOT NULL
);

CREATE INDEX idx_sale_price_history_product ON sale_price_history(product_id, created_at DESC);
CREATE INDEX idx_purchase_price_history_product ON purchase_price_history(product_id, created_at DESC);
CREATE INDEX idx_assembly_items_assembly ON assembly_items(assembly_id);
