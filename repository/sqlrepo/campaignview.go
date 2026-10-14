package sqlrepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

// visitorExpr identifies the visitor of a view: the external user id when
// informed, otherwise the IP address. The prefixes keep an external id from
// ever matching an IP. CONCAT exists in PostgreSQL, MySQL and SQL Server.
const visitorExpr = "CONCAT(CASE WHEN external_user_id IS NULL THEN 'i:' ELSE 'e:' END, COALESCE(external_user_id, ip_address))"

const summarySelect = "COUNT(*) AS total_views, COUNT(DISTINCT " + visitorExpr + ") AS unique_views"

type CampaignViewRepository struct {
	db *gorm.DB
	// dayExpr truncates viewed_on_utc to its UTC date.
	dayExpr string
}

func newCampaignViewRepository(db *gorm.DB) *CampaignViewRepository {
	// PostgreSQL stores time.Time as timestamptz, whose cast to date uses the
	// session time zone; MySQL (datetime) and SQL Server (datetimeoffset) keep
	// the UTC value that was written.
	dayExpr := "CAST(viewed_on_utc AS DATE)"
	if db.Dialector.Name() == DriverPostgres {
		dayExpr = "CAST(viewed_on_utc AT TIME ZONE 'UTC' AS DATE)"
	}
	return &CampaignViewRepository{db: db, dayExpr: dayExpr}
}

func (r *CampaignViewRepository) Create(ctx context.Context, view *domain.CampaignView) error {
	m := newCampaignViewModel(view)
	return translateError(r.db.WithContext(ctx).Create(&m).Error)
}

func (r *CampaignViewRepository) query(ctx context.Context, campaignId string, period domain.Period) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&campaignViewModel{}).Where("campaign_id = ?", campaignId)
	if !period.FromUtc.IsZero() {
		q = q.Where("viewed_on_utc >= ?", period.FromUtc)
	}
	if !period.ToUtc.IsZero() {
		q = q.Where("viewed_on_utc < ?", period.ToUtc)
	}
	return q.Session(&gorm.Session{})
}

type summaryRow struct {
	TotalViews  int64
	UniqueViews int64
}

func (r *CampaignViewRepository) Summary(ctx context.Context, campaignId string, period domain.Period) (domain.CampaignViewSummary, error) {
	var row summaryRow
	if err := r.query(ctx, campaignId, period).Select(summarySelect).Scan(&row).Error; err != nil {
		return domain.CampaignViewSummary{}, err
	}
	return domain.CampaignViewSummary{TotalViews: row.TotalViews, UniqueViews: row.UniqueViews}, nil
}

type dailyRow struct {
	Day         time.Time
	TotalViews  int64
	UniqueViews int64
}

func (r *CampaignViewRepository) DailySummary(ctx context.Context, campaignId string, period domain.Period) ([]domain.DailyCampaignViews, error) {
	var rows []dailyRow
	err := r.query(ctx, campaignId, period).
		Select(r.dayExpr + " AS day, " + summarySelect).
		// SQL Server does not accept the alias in GROUP BY.
		Group(r.dayExpr).
		Order("day").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	days := make([]domain.DailyCampaignViews, len(rows))
	for i, row := range rows {
		y, m, d := row.Day.Date()
		days[i] = domain.DailyCampaignViews{
			DateUtc:             time.Date(y, m, d, 0, 0, 0, 0, time.UTC),
			CampaignViewSummary: domain.CampaignViewSummary{TotalViews: row.TotalViews, UniqueViews: row.UniqueViews},
		}
	}
	return days, nil
}

type userAgentVisitorRow struct {
	UserAgent string
	Visitor   string
	Views     int64
}

func (r *CampaignViewRepository) UserAgentVisitors(ctx context.Context, campaignId string, period domain.Period) ([]domain.VisitorUserAgentViews, error) {
	var rows []userAgentVisitorRow
	err := r.query(ctx, campaignId, period).
		Select("user_agent, " + visitorExpr + " AS visitor, COUNT(*) AS views").
		// SQL Server does not accept the alias in GROUP BY.
		Group("user_agent, " + visitorExpr).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]domain.VisitorUserAgentViews, len(rows))
	for i, row := range rows {
		result[i] = domain.VisitorUserAgentViews{UserAgent: row.UserAgent, Visitor: row.Visitor, Views: row.Views}
	}
	return result, nil
}

func (r *CampaignViewRepository) List(ctx context.Context, campaignId string, period domain.Period, page repository.Pagination) ([]domain.CampaignView, int64, error) {
	q := r.query(ctx, campaignId, period)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var models []campaignViewModel
	err := q.Order("viewed_on_utc DESC").Order("id").Offset(page.Offset).Limit(page.Limit).Find(&models).Error
	if err != nil {
		return nil, 0, err
	}

	views := make([]domain.CampaignView, len(models))
	for i, m := range models {
		views[i] = m.toDomain()
	}
	return views, total, nil
}
