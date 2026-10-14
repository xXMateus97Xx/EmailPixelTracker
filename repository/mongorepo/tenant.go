package mongorepo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

type TenantRepository struct {
	col *mongo.Collection
}

func (r *TenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	_, err := r.col.InsertOne(ctx, newTenantDocument(tenant))
	return translateError(err)
}

func (r *TenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	filter := bson.D{{Key: "_id", Value: tenant.Id}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "name", Value: tenant.Name},
		{Key: "active", Value: tenant.Active},
	}}}
	return updateResultError(r.col.UpdateOne(ctx, filter, update))
}

func (r *TenantRepository) GetById(ctx context.Context, id string) (*domain.Tenant, error) {
	var doc tenantDocument
	if err := r.col.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&doc); err != nil {
		return nil, translateError(err)
	}
	t := doc.toDomain()
	return &t, nil
}

func (r *TenantRepository) ExistsByName(ctx context.Context, name, excludeId string) (bool, error) {
	return exists(ctx, r.col, withExclude(bson.D{{Key: "name", Value: name}}, excludeId))
}

func (r *TenantRepository) List(ctx context.Context, page repository.Pagination) ([]domain.Tenant, int64, error) {
	return list(ctx, r.col, bson.D{}, page, tenantDocument.toDomain)
}
