package postgres

import (
	"context"
	"errors"

	"erp/services/stock-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CatalogRepo struct {
	pool *pgxpool.Pool
}

func NewCatalogRepo(pool *pgxpool.Pool) *CatalogRepo {
	return &CatalogRepo{pool: pool}
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (r *CatalogRepo) InsertSaleHistory(ctx context.Context, h domain.PriceHistory) (domain.PriceHistory, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO sale_price_history (product_id, sku, previous_price, new_price)
		VALUES ($1,$2,$3,$4)
		RETURNING id, created_at
	`, h.ProductID, h.SKU, h.PreviousPrice, h.NewPrice).Scan(&h.ID, &h.CreatedAt)
	return h, err
}

func (r *CatalogRepo) InsertPurchaseHistory(ctx context.Context, h domain.PriceHistory) (domain.PriceHistory, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO purchase_price_history (product_id, sku, previous_price, new_price, reference_doc_id)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at
	`, h.ProductID, h.SKU, h.PreviousPrice, h.NewPrice, nilIfEmpty(h.ReferenceDocID)).Scan(&h.ID, &h.CreatedAt)
	return h, err
}

func (r *CatalogRepo) ListSaleHistory(ctx context.Context, productID string) ([]domain.PriceHistory, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, product_id, sku, previous_price, new_price, created_at
		FROM sale_price_history
		WHERE ($1 = '' OR product_id = $1)
		ORDER BY created_at DESC
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PriceHistory
	for rows.Next() {
		var h domain.PriceHistory
		if err := rows.Scan(&h.ID, &h.ProductID, &h.SKU, &h.PreviousPrice, &h.NewPrice, &h.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if out == nil {
		out = []domain.PriceHistory{}
	}
	return out, rows.Err()
}

func (r *CatalogRepo) ListPurchaseHistory(ctx context.Context, productID string) ([]domain.PriceHistory, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, product_id, sku, previous_price, new_price, COALESCE(reference_doc_id, ''), created_at
		FROM purchase_price_history
		WHERE ($1 = '' OR product_id = $1)
		ORDER BY created_at DESC
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PriceHistory
	for rows.Next() {
		var h domain.PriceHistory
		if err := rows.Scan(&h.ID, &h.ProductID, &h.SKU, &h.PreviousPrice, &h.NewPrice, &h.ReferenceDocID, &h.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if out == nil {
		out = []domain.PriceHistory{}
	}
	return out, rows.Err()
}

func (r *CatalogRepo) CreateAssembly(ctx context.Context, a domain.Assembly) (domain.Assembly, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Assembly{}, err
	}
	defer tx.Rollback(ctx)
	if err := tx.QueryRow(ctx, `
		INSERT INTO assemblies (code, name, product_id, margin_percent, cost, suggested_price)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at
	`, a.Code, a.Name, nilIfEmpty(a.ProductID), a.MarginPercent, a.Cost, a.SuggestedPrice).Scan(&a.ID, &a.CreatedAt); err != nil {
		if isUnique(err) {
			return domain.Assembly{}, domain.ErrConflict
		}
		return domain.Assembly{}, err
	}
	items, err := insertItems(ctx, tx, a.ID, a.Items)
	if err != nil {
		return domain.Assembly{}, err
	}
	a.Items = items
	if err := tx.Commit(ctx); err != nil {
		return domain.Assembly{}, err
	}
	return a, nil
}

func (r *CatalogRepo) UpdateAssembly(ctx context.Context, a domain.Assembly) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE assemblies
		SET code=$2, name=$3, product_id=$4, margin_percent=$5, cost=$6, suggested_price=$7
		WHERE id=$1
	`, a.ID, a.Code, a.Name, nilIfEmpty(a.ProductID), a.MarginPercent, a.Cost, a.SuggestedPrice)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM assembly_items WHERE assembly_id=$1`, a.ID); err != nil {
		return err
	}
	if _, err := insertItems(ctx, tx, a.ID, a.Items); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *CatalogRepo) GetAssembly(ctx context.Context, id string) (domain.Assembly, error) {
	a, err := scanAssembly(r.pool.QueryRow(ctx, `
		SELECT id, code, name, product_id, margin_percent, cost, suggested_price, created_at
		FROM assemblies WHERE id=$1
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Assembly{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Assembly{}, err
	}
	items, err := r.loadItems(ctx, []string{a.ID})
	if err != nil {
		return domain.Assembly{}, err
	}
	a.Items = items[a.ID]
	if a.Items == nil {
		a.Items = []domain.AssemblyItem{}
	}
	return a, nil
}

func (r *CatalogRepo) ListAssemblies(ctx context.Context) ([]domain.Assembly, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, name, product_id, margin_percent, cost, suggested_price, created_at
		FROM assemblies ORDER BY code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Assembly
	var ids []string
	for rows.Next() {
		a, err := scanAssembly(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
		ids = append(ids, a.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		return []domain.Assembly{}, nil
	}
	grouped, err := r.loadItems(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Items = grouped[out[i].ID]
		if out[i].Items == nil {
			out[i].Items = []domain.AssemblyItem{}
		}
	}
	return out, nil
}

func (r *CatalogRepo) ProductInUse(ctx context.Context, productID string) (bool, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(1) FROM assemblies WHERE product_id=$1
	`, productID).Scan(&n)
	if err != nil || n > 0 {
		return n > 0, err
	}
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(1) FROM assembly_items WHERE product_id=$1
	`, productID).Scan(&n)
	return n > 0, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAssembly(row rowScanner) (domain.Assembly, error) {
	var a domain.Assembly
	var productID *string
	err := row.Scan(&a.ID, &a.Code, &a.Name, &productID, &a.MarginPercent, &a.Cost, &a.SuggestedPrice, &a.CreatedAt)
	if productID != nil {
		a.ProductID = *productID
	}
	return a, err
}

func insertItems(ctx context.Context, tx pgx.Tx, assemblyID string, items []domain.AssemblyItem) ([]domain.AssemblyItem, error) {
	out := make([]domain.AssemblyItem, 0, len(items))
	for _, it := range items {
		if err := tx.QueryRow(ctx, `
			INSERT INTO assembly_items (assembly_id, product_id, quantity, role, unit_price)
			VALUES ($1,$2,$3,$4,$5)
			RETURNING id
		`, assemblyID, it.ProductID, it.Quantity, it.Role, it.UnitPrice).Scan(&it.ID); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, nil
}

func (r *CatalogRepo) loadItems(ctx context.Context, assemblyIDs []string) (map[string][]domain.AssemblyItem, error) {
	out := map[string][]domain.AssemblyItem{}
	if len(assemblyIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, assembly_id, product_id, quantity, role, unit_price
		FROM assembly_items WHERE assembly_id = ANY($1::uuid[])
	`, assemblyIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it domain.AssemblyItem
		var assemblyID string
		if err := rows.Scan(&it.ID, &assemblyID, &it.ProductID, &it.Quantity, &it.Role, &it.UnitPrice); err != nil {
			return nil, err
		}
		out[assemblyID] = append(out[assemblyID], it)
	}
	return out, rows.Err()
}
