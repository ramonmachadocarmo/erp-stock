ALTER TABLE stock_movements
    ADD COLUMN subtype VARCHAR(30) NOT NULL DEFAULT '';

UPDATE stock_movements SET subtype = 'PURCHASE' WHERE movement_type = 'PURCHASE_IN';
UPDATE stock_movements SET subtype = 'SALE' WHERE movement_type = 'SALE_OUT';
