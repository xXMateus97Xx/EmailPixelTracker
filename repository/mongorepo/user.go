package mongorepo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

type UserRepository struct {
	col *mongo.Collection
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	_, err := r.col.InsertOne(ctx, newUserDocument(user))
	return translateError(err)
}

func (r *UserRepository) Update(ctx context.Context, tenantId string, user *domain.User) error {
	filter := bson.D{{Key: "_id", Value: user.Id}, {Key: "tenantId", Value: tenantId}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "name", Value: user.Name},
		{Key: "username", Value: user.Username},
		{Key: "active", Value: user.Active},
		{Key: "password", Value: user.Password},
		{Key: "role", Value: user.Role},
	}}}
	return updateResultError(r.col.UpdateOne(ctx, filter, update))
}

func (r *UserRepository) GetById(ctx context.Context, tenantId, id string) (*domain.User, error) {
	var doc userDocument
	filter := bson.D{{Key: "_id", Value: id}, {Key: "tenantId", Value: tenantId}}
	if err := r.col.FindOne(ctx, filter).Decode(&doc); err != nil {
		return nil, translateError(err)
	}
	u := doc.toDomain()
	return &u, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var doc userDocument
	if err := r.col.FindOne(ctx, bson.D{{Key: "username", Value: username}}).Decode(&doc); err != nil {
		return nil, translateError(err)
	}
	u := doc.toDomain()
	return &u, nil
}

func (r *UserRepository) ExistsByUsername(ctx context.Context, username, excludeId string) (bool, error) {
	return exists(ctx, r.col, withExclude(bson.D{{Key: "username", Value: username}}, excludeId))
}

func (r *UserRepository) List(ctx context.Context, tenantId string, page repository.Pagination) ([]domain.User, int64, error) {
	return list(ctx, r.col, bson.D{{Key: "tenantId", Value: tenantId}}, page, userDocument.toDomain)
}
