CREATE TABLE code_counters (
    name TEXT PRIMARY KEY,
    n BIGINT NOT NULL DEFAULT 0
);

INSERT INTO code_counters (name, n)
SELECT 'warehouse', COALESCE(MAX(code::BIGINT), 0)
FROM warehouses
WHERE code ~ '^[0-9]+$';

INSERT INTO code_counters (name, n)
SELECT 'assembly', COALESCE(MAX(code::BIGINT), 0)
FROM assemblies
WHERE code ~ '^[0-9]+$';
