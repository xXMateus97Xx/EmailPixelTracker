package mongorepo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

type CampaignRepository struct {
	col *mongo.Collection
}

func (r *CampaignRepository) Create(ctx context.Context, campaign *domain.Campaign) error {
	_, err := r.col.InsertOne(ctx, newCampaignDocument(campaign))
	return translateError(err)
}

func (r *CampaignRepository) Update(ctx context.Context, tenantId string, campaign *domain.Campaign) error {
	filter := bson.D{{Key: "_id", Value: campaign.Id}, {Key: "tenantId", Value: tenantId}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "name", Value: campaign.Name},
		{Key: "active", Value: campaign.Active},
		{Key: "expireInUtc", Value: campaign.ExpireInUtc},
		{Key: "pixelColorR", Value: campaign.PixelColorR},
		{Key: "pixelColorG", Value: campaign.PixelColorG},
		{Key: "pixelColorB", Value: campaign.PixelColorB},
		{Key: "pixelColorA", Value: campaign.PixelColorA},
	}}}
	return updateResultError(r.col.UpdateOne(ctx, filter, update))
}

func (r *CampaignRepository) GetById(ctx context.Context, tenantId, id string) (*domain.Campaign, error) {
	var doc campaignDocument
	filter := bson.D{{Key: "_id", Value: id}, {Key: "tenantId", Value: tenantId}}
	if err := r.col.FindOne(ctx, filter).Decode(&doc); err != nil {
		return nil, translateError(err)
	}
	c := doc.toDomain()
	return &c, nil
}

func (r *CampaignRepository) GetByIdAnyTenant(ctx context.Context, id string) (*domain.Campaign, error) {
	var doc campaignDocument
	if err := r.col.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&doc); err != nil {
		return nil, translateError(err)
	}
	c := doc.toDomain()
	return &c, nil
}

func (r *CampaignRepository) ExistsByName(ctx context.Context, tenantId, name, excludeId string) (bool, error) {
	filter := bson.D{{Key: "tenantId", Value: tenantId}, {Key: "name", Value: name}}
	return exists(ctx, r.col, withExclude(filter, excludeId))
}

func (r *CampaignRepository) List(ctx context.Context, tenantId string, page repository.Pagination) ([]domain.Campaign, int64, error) {
	return list(ctx, r.col, bson.D{{Key: "tenantId", Value: tenantId}}, page, campaignDocument.toDomain)
}
