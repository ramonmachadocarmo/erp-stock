package mongoadapter

import (
	"context"
	"errors"

	"erp/services/stock-service/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type CategoryRepo struct {
	col *mongo.Collection
}

func NewCategoryRepo(db *mongo.Database) *CategoryRepo {
	col := db.Collection("categories")
	_, _ = col.Indexes().CreateOne(context.Background(), mongo.IndexModel{Keys: bson.D{{Key: "parent_id", Value: 1}}})
	return &CategoryRepo{col: col}
}

func (r *CategoryRepo) Create(ctx context.Context, c domain.Category) (domain.Category, error) {
	c.Children = nil
	_, err := r.col.InsertOne(ctx, c)
	return c, err
}

func (r *CategoryRepo) Update(ctx context.Context, c domain.Category) error {
	res, err := r.col.UpdateByID(ctx, c.ID, bson.M{"$set": bson.M{
		"parent_id": c.ParentID,
		"name":      c.Name,
	}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CategoryRepo) Get(ctx context.Context, id string) (domain.Category, error) {
	var c domain.Category
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.Category{}, domain.ErrNotFound
	}
	return c, err
}

func (r *CategoryRepo) List(ctx context.Context) ([]domain.Category, error) {
	cur, err := r.col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.Category
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []domain.Category{}
	}
	return out, nil
}

func (r *CategoryRepo) Delete(ctx context.Context, id string) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CategoryRepo) HasChildren(ctx context.Context, id string) (bool, error) {
	n, err := r.col.CountDocuments(ctx, bson.M{"parent_id": id})
	return n > 0, err
}
