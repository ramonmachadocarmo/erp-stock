package postgres

import (
	"context"
	"errors"
	"strings"

	"erp/services/stock-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const reservedSQL = `COALESCE((
	SELECT SUM(r.quantity) FROM stock_reservations r
	WHERE r.product_id = stock_balances.product_id AND r.status = 'OPEN'
	  AND (r.warehouse_id IS NULL OR r.warehouse_id = stock_balances.warehouse_id)
), 0)`

type StockRepo struct {
	pool *pgxpool.Pool
}

func NewStockRepo(pool *pgxpool.Pool) *StockRepo {
	return &StockRepo{pool: pool}
}

func (r *StockRepo) CreateWarehouse(ctx context.Context, w domain.Warehouse) (domain.Warehouse, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO warehouses (code, name, active) VALUES ($1,$2,TRUE)
		RETURNING id, active, created_at
	`, w.Code, w.Name).Scan(&w.ID, &w.Active, &w.CreatedAt)
	if isUnique(err) {
		return domain.Warehouse{}, domain.ErrConflict
	}
	return w, err
}

func (r *StockRepo) UpdateWarehouse(ctx context.Context, w domain.Warehouse) error {
	tag, err := r.pool.Exec(ctx, `UPDATE warehouses SET code=$2, name=$3 WHERE id=$1`, w.ID, w.Code, w.Name)
	if err != nil {
		if isUnique(err) {
			return domain.ErrConflict
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *StockRepo) ListWarehouses(ctx context.Context) ([]domain.Warehouse, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, code, name, active, created_at FROM warehouses ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Warehouse
	for rows.Next() {
		var w domain.Warehouse
		if err := rows.Scan(&w.ID, &w.Code, &w.Name, &w.Active, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	if out == nil {
		out = []domain.Warehouse{}
	}
	return out, rows.Err()
}

func (r *StockRepo) GetWarehouse(ctx context.Context, id string) (domain.Warehouse, error) {
	var w domain.Warehouse
	err := r.pool.QueryRow(ctx, `SELECT id, code, name, active, created_at FROM warehouses WHERE id=$1`, id).
		Scan(&w.ID, &w.Code, &w.Name, &w.Active, &w.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Warehouse{}, domain.ErrNotFound
	}
	return w, err
}

func (r *StockRepo) UpsertBalance(ctx context.Context, productID, warehouseID string, available float64) (domain.Balance, error) {
	var b domain.Balance
	err := r.pool.QueryRow(ctx, `
		INSERT INTO stock_balances (product_id, warehouse_id, quantity_available)
		VALUES ($1,$2,$3)
		ON CONFLICT (product_id, warehouse_id)
		DO UPDATE SET quantity_available=EXCLUDED.quantity_available, updated_at=NOW()
		RETURNING id, product_id, warehouse_id, quantity_available, `+reservedSQL+`, updated_at
	`, productID, warehouseID, available).Scan(&b.ID, &b.ProductID, &b.WarehouseID, &b.QuantityAvailable, &b.QuantityReserved, &b.UpdatedAt)
	return b, err
}

func (r *StockRepo) GetBalance(ctx context.Context, productID, warehouseID string) (domain.Balance, error) {
	var b domain.Balance
	err := r.pool.QueryRow(ctx, `
		SELECT id, product_id, warehouse_id, quantity_available, `+reservedSQL+`, updated_at
		FROM stock_balances WHERE product_id=$1 AND warehouse_id=$2
	`, productID, warehouseID).Scan(&b.ID, &b.ProductID, &b.WarehouseID, &b.QuantityAvailable, &b.QuantityReserved, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Balance{}, domain.ErrNotFound
	}
	return b, err
}

func (r *StockRepo) ListBalances(ctx context.Context) ([]domain.Balance, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, product_id, warehouse_id, quantity_available, `+reservedSQL+`, updated_at
		FROM stock_balances ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Balance
	for rows.Next() {
		var b domain.Balance
		if err := rows.Scan(&b.ID, &b.ProductID, &b.WarehouseID, &b.QuantityAvailable, &b.QuantityReserved, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if out == nil {
		out = []domain.Balance{}
	}
	return out, rows.Err()
}

func (r *StockRepo) ReplaceReservations(ctx context.Context, orderID, warehouseID string, items []domain.OrderItem) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	products := map[string]struct{}{}
	rows, err := tx.Query(ctx, `SELECT product_id FROM stock_reservations WHERE sales_order_id=$1`, orderID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			rows.Close()
			return err
		}
		products[pid] = struct{}{}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	var wh any
	if warehouseID != "" {
		wh = warehouseID
	}
	keep := make([]string, 0, len(items))
	for _, it := range items {
		if it.ProductID == "" || it.Quantity <= 0 {
			continue
		}
		keep = append(keep, it.ProductID)
		products[it.ProductID] = struct{}{}
		if _, err := tx.Exec(ctx, `
			INSERT INTO stock_reservations (sales_order_id, product_id, warehouse_id, quantity, status)
			VALUES ($1,$2,$3,$4,'OPEN')
			ON CONFLICT (sales_order_id, product_id)
			DO UPDATE SET quantity = EXCLUDED.quantity,
				warehouse_id = COALESCE(EXCLUDED.warehouse_id, stock_reservations.warehouse_id),
				status = 'OPEN', updated_at = NOW()
		`, orderID, it.ProductID, wh, it.Quantity); err != nil {
			return err
		}
	}
	if len(keep) == 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE stock_reservations SET status='RELEASED', updated_at=NOW()
			WHERE sales_order_id=$1 AND status='OPEN'
		`, orderID); err != nil {
			return err
		}
	} else if _, err := tx.Exec(ctx, `
		UPDATE stock_reservations SET status='RELEASED', updated_at=NOW()
		WHERE sales_order_id=$1 AND status='OPEN' AND NOT (product_id = ANY($2))
	`, orderID, keep); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return r.recalcReserved(ctx, keys(products))
}

func (r *StockRepo) ReleaseReservations(ctx context.Context, orderID string) error {
	return r.ReplaceReservations(ctx, orderID, "", nil)
}

func (r *StockRepo) ConfirmReservations(ctx context.Context, orderID string) ([]domain.Reservation, error) {
	rows, err := r.pool.Query(ctx, `
		UPDATE stock_reservations SET status='CONFIRMED', updated_at=NOW()
		WHERE sales_order_id=$1 AND status='OPEN'
		RETURNING id, sales_order_id::text, product_id, COALESCE(warehouse_id::text, ''), quantity, status, created_at, updated_at
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Reservation
	products := map[string]struct{}{}
	for rows.Next() {
		var rv domain.Reservation
		if err := rows.Scan(&rv.ID, &rv.SalesOrderID, &rv.ProductID, &rv.WarehouseID, &rv.Quantity, &rv.Status, &rv.CreatedAt, &rv.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, rv)
		products[rv.ProductID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) > 0 {
		if err := r.recalcReserved(ctx, keys(products)); err != nil {
			return nil, err
		}
		return out, nil
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(1) FROM stock_reservations WHERE sales_order_id=$1`, orderID).Scan(&n); err != nil {
		return nil, err
	}
	if n > 0 {
		return []domain.Reservation{}, nil
	}
	return nil, nil
}

func (r *StockRepo) ListOpenReservations(ctx context.Context, productID string) ([]domain.Reservation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, sales_order_id::text, product_id, COALESCE(warehouse_id::text, ''), quantity, status, created_at, updated_at
		FROM stock_reservations
		WHERE product_id=$1 AND status='OPEN'
		ORDER BY created_at
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Reservation{}
	for rows.Next() {
		var rv domain.Reservation
		if err := rows.Scan(&rv.ID, &rv.SalesOrderID, &rv.ProductID, &rv.WarehouseID, &rv.Quantity, &rv.Status, &rv.CreatedAt, &rv.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

func (r *StockRepo) recalcReserved(ctx context.Context, productIDs []string) error {
	if len(productIDs) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE stock_balances b
		SET quantity_reserved = COALESCE((
			SELECT SUM(r.quantity) FROM stock_reservations r
			WHERE r.product_id = b.product_id AND r.status = 'OPEN'
			  AND (r.warehouse_id IS NULL OR r.warehouse_id = b.warehouse_id)
		), 0), updated_at = NOW()
		WHERE b.product_id = ANY($1)
	`, productIDs)
	return err
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func (r *StockRepo) Reserve(ctx context.Context, productID, warehouseID string, qty float64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE stock_balances
		SET quantity_reserved = quantity_reserved + $3, updated_at=NOW()
		WHERE product_id=$1 AND warehouse_id=$2 AND quantity_available - quantity_reserved >= $3
	`, productID, warehouseID, qty)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInsufficient
	}
	return nil
}

func (r *StockRepo) ConfirmSale(ctx context.Context, productID, warehouseID string, qty float64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE stock_balances
		SET quantity_available = quantity_available - $3,
		    quantity_reserved = quantity_reserved - $3,
		    updated_at=NOW()
		WHERE product_id=$1 AND warehouse_id=$2 AND quantity_reserved >= $3 AND quantity_available >= $3
	`, productID, warehouseID, qty)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInsufficient
	}
	return nil
}

func (r *StockRepo) AddAvailable(ctx context.Context, productID, warehouseID string, qty float64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO stock_balances (product_id, warehouse_id, quantity_available)
		VALUES ($1,$2,$3)
		ON CONFLICT (product_id, warehouse_id)
		DO UPDATE SET quantity_available = stock_balances.quantity_available + $3, updated_at=NOW()
	`, productID, warehouseID, qty)
	return err
}

func (r *StockRepo) RemoveAvailable(ctx context.Context, productID, warehouseID string, qty float64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE stock_balances
		SET quantity_available = quantity_available - $3, updated_at=NOW()
		WHERE product_id=$1 AND warehouse_id=$2 AND quantity_available - quantity_reserved >= $3
	`, productID, warehouseID, qty)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInsufficient
	}
	return nil
}

func (r *StockRepo) HasMovement(ctx context.Context, docType, docID, movementType string) (bool, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(1) FROM stock_movements
		WHERE reference_doc_type=$1 AND reference_doc_id=$2 AND movement_type=$3
	`, docType, docID, movementType).Scan(&n)
	return n > 0, err
}

func (r *StockRepo) ProductInUse(ctx context.Context, productID string) (bool, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(1) FROM stock_movements WHERE product_id=$1
	`, productID).Scan(&n)
	if err != nil || n > 0 {
		return n > 0, err
	}
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(1) FROM stock_balances
		WHERE product_id=$1 AND (quantity_available <> 0 OR quantity_reserved <> 0)
	`, productID).Scan(&n)
	if err != nil || n > 0 {
		return n > 0, err
	}
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(1) FROM stock_reservations WHERE product_id=$1 AND status='OPEN'
	`, productID).Scan(&n)
	return n > 0, err
}

func (r *StockRepo) InsertMovement(ctx context.Context, m domain.Movement) error {
	var refID any
	if m.ReferenceDocID != "" {
		refID = m.ReferenceDocID
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO stock_movements (product_id, warehouse_id, movement_type, subtype, quantity, reference_doc_type, reference_doc_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, m.ProductID, m.WarehouseID, m.MovementType, m.Subtype, m.Quantity, m.ReferenceDocType, refID)
	return err
}

func (r *StockRepo) ListMovements(ctx context.Context) ([]domain.Movement, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, product_id, warehouse_id, movement_type, subtype, quantity, reference_doc_type, COALESCE(reference_doc_id::text, ''), created_at
		FROM stock_movements ORDER BY created_at DESC LIMIT 200
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Movement
	for rows.Next() {
		var m domain.Movement
		if err := rows.Scan(&m.ID, &m.ProductID, &m.WarehouseID, &m.MovementType, &m.Subtype, &m.Quantity, &m.ReferenceDocType, &m.ReferenceDocID, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if out == nil {
		out = []domain.Movement{}
	}
	return out, rows.Err()
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && (pgErr.Code == "23505" || strings.Contains(err.Error(), "duplicate"))
}
