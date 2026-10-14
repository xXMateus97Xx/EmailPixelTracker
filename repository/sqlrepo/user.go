package sqlrepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

type UserRepository struct {
	db *gorm.DB
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	m := newUserModel(user)
	return translateError(r.db.WithContext(ctx).Create(&m).Error)
}

func (r *UserRepository) Update(ctx context.Context, tenantId string, user *domain.User) error {
	m := newUserModel(user)
	err := r.db.WithContext(ctx).
		Model(&userModel{}).
		Where("id = ? AND tenant_id = ?", user.Id, tenantId).
		Updates(map[string]any{
			"name":     m.Name,
			"username": m.Username,
			"active":   m.Active,
			"password": m.Password,
			"role":     m.Role,
		}).Error
	return translateError(err)
}

func (r *UserRepository) GetById(ctx context.Context, tenantId, id string) (*domain.User, error) {
	var m userModel
	err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantId).Take(&m).Error
	if err != nil {
		return nil, translateError(err)
	}
	u := m.toDomain()
	return &u, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var m userModel
	if err := r.db.WithContext(ctx).Where("username = ?", username).Take(&m).Error; err != nil {
		return nil, translateError(err)
	}
	u := m.toDomain()
	return &u, nil
}

func (r *UserRepository) ExistsByUsername(ctx context.Context, username, excludeId string) (bool, error) {
	q := r.db.WithContext(ctx).Model(&userModel{}).Where("username = ?", username)
	if excludeId != "" {
		q = q.Where("id <> ?", excludeId)
	}
	return exists(q)
}

func (r *UserRepository) List(ctx context.Context, tenantId string, page repository.Pagination) ([]domain.User, int64, error) {
	q := r.db.WithContext(ctx).Model(&userModel{}).Where("tenant_id = ?", tenantId).Session(&gorm.Session{})

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var models []userModel
	if err := paginate(q, page).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	users := make([]domain.User, len(models))
	for i, m := range models {
		users[i] = m.toDomain()
	}
	return users, total, nil
}
