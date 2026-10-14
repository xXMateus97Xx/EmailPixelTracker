package sqlrepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

type TenantRepository struct {
	db *gorm.DB
}

func (r *TenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	m := newTenantModel(tenant)
	return translateError(r.db.WithContext(ctx).Create(&m).Error)
}

func (r *TenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	// A map is used so zero values (e.g. Active=false) are also written.
	err := r.db.WithContext(ctx).
		Model(&tenantModel{}).
		Where("id = ?", tenant.Id).
		Updates(map[string]any{
			"name":   tenant.Name,
			"active": tenant.Active,
		}).Error
	return translateError(err)
}

func (r *TenantRepository) GetById(ctx context.Context, id string) (*domain.Tenant, error) {
	var m tenantModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&m).Error; err != nil {
		return nil, translateError(err)
	}
	t := m.toDomain()
	return &t, nil
}

func (r *TenantRepository) ExistsByName(ctx context.Context, name, excludeId string) (bool, error) {
	q := r.db.WithContext(ctx).Model(&tenantModel{}).Where("name = ?", name)
	if excludeId != "" {
		q = q.Where("id <> ?", excludeId)
	}
	return exists(q)
}

func (r *TenantRepository) List(ctx context.Context, page repository.Pagination) ([]domain.Tenant, int64, error) {
	q := r.db.WithContext(ctx).Model(&tenantModel{}).Session(&gorm.Session{})

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var models []tenantModel
	if err := paginate(q, page).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	tenants := make([]domain.Tenant, len(models))
	for i, m := range models {
		tenants[i] = m.toDomain()
	}
	return tenants, total, nil
}
