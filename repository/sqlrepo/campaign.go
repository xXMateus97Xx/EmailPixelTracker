package sqlrepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

type CampaignRepository struct {
	db *gorm.DB
}

func (r *CampaignRepository) Create(ctx context.Context, campaign *domain.Campaign) error {
	m := newCampaignModel(campaign)
	return translateError(r.db.WithContext(ctx).Create(&m).Error)
}

func (r *CampaignRepository) Update(ctx context.Context, tenantId string, campaign *domain.Campaign) error {
	err := r.db.WithContext(ctx).
		Model(&campaignModel{}).
		Where("id = ? AND tenant_id = ?", campaign.Id, tenantId).
		Updates(map[string]any{
			"name":          campaign.Name,
			"active":        campaign.Active,
			"expire_in_utc": campaign.ExpireInUtc,
			"pixel_color_r": campaign.PixelColorR,
			"pixel_color_g": campaign.PixelColorG,
			"pixel_color_b": campaign.PixelColorB,
			"pixel_color_a": campaign.PixelColorA,
		}).Error
	return translateError(err)
}

func (r *CampaignRepository) GetById(ctx context.Context, tenantId, id string) (*domain.Campaign, error) {
	var m campaignModel
	err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantId).Take(&m).Error
	if err != nil {
		return nil, translateError(err)
	}
	c := m.toDomain()
	return &c, nil
}

func (r *CampaignRepository) GetByIdAnyTenant(ctx context.Context, id string) (*domain.Campaign, error) {
	var m campaignModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&m).Error; err != nil {
		return nil, translateError(err)
	}
	c := m.toDomain()
	return &c, nil
}

func (r *CampaignRepository) ExistsByName(ctx context.Context, tenantId, name, excludeId string) (bool, error) {
	q := r.db.WithContext(ctx).Model(&campaignModel{}).Where("tenant_id = ? AND name = ?", tenantId, name)
	if excludeId != "" {
		q = q.Where("id <> ?", excludeId)
	}
	return exists(q)
}

func (r *CampaignRepository) List(ctx context.Context, tenantId string, page repository.Pagination) ([]domain.Campaign, int64, error) {
	q := r.db.WithContext(ctx).Model(&campaignModel{}).Where("tenant_id = ?", tenantId).Session(&gorm.Session{})

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var models []campaignModel
	if err := paginate(q, page).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	campaigns := make([]domain.Campaign, len(models))
	for i, m := range models {
		campaigns[i] = m.toDomain()
	}
	return campaigns, total, nil
}
