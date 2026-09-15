package mongoadapter

import (
	"context"
	"errors"
	"time"

	"erp/services/stock-service/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProductRepo struct {
	col *mongo.Collection
}

func NewProductRepo(db *mongo.Database) *ProductRepo {
	r := &ProductRepo{col: db.Collection("products")}
	_, _ = r.col.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "sku", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return r
}

func (r *ProductRepo) CountByCategory(ctx context.Context, categoryID string) (int64, error) {
	return r.col.CountDocuments(ctx, bson.M{"category_id": categoryID})
}

func (r *ProductRepo) Create(ctx context.Context, p domain.Product) (domain.Product, error) {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	if p.Attributes == nil {
		p.Attributes = map[string]any{}
	}
	_, err := r.col.InsertOne(ctx, p)
	if mongo.IsDuplicateKeyError(err) {
		return domain.Product{}, domain.ErrConflict
	}
	return p, err
}

func (r *ProductRepo) Update(ctx context.Context, p domain.Product) error {
	res, err := r.col.UpdateByID(ctx, p.ID, bson.M{"$set": bson.M{
		"sku": p.SKU, "barcode": p.Barcode, "name": p.Name, "category_id": p.CategoryID,
		"ncm": p.NCM, "unit_of_measure": p.UnitOfMeasure, "purchase_uom": p.PurchaseUoM,
		"sale_uom": p.SaleUoM, "stock_uom": p.StockUoM, "uom_conversions": p.Conversions,
		"kind": p.Kind, "weight_kg": p.WeightKg, "volume_m3": p.VolumeM3, "attributes": p.Attributes,
	}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ProductRepo) Get(ctx context.Context, id string) (domain.Product, error) {
	var p domain.Product
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.Product{}, domain.ErrNotFound
	}
	return p, err
}

func (r *ProductRepo) List(ctx context.Context) ([]domain.Product, error) {
	cur, err := r.col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.Product
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []domain.Product{}
	}
	return out, nil
}

func (r *ProductRepo) SetSalePrice(ctx context.Context, id string, price float64) error {
	return r.setPrice(ctx, id, "sale_price", price)
}

func (r *ProductRepo) SetPurchasePrice(ctx context.Context, id string, price float64) error {
	return r.setPrice(ctx, id, "purchase_price", price)
}

func (r *ProductRepo) setPrice(ctx context.Context, id, field string, price float64) error {
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{field: price}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ProductRepo) Delete(ctx context.Context, id string) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}
