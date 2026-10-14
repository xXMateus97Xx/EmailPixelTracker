package mongorepo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

// CampaignViewRepository keeps one document per campaign, with every view in
// a list:
//
//	{ _id: <campaignId>, views: [ { id, viewedOnUtc, externalUserId?, ipAddress, userAgent? } ] }
//
// MongoDB limits a document to 16MB, which caps the number of views a single
// campaign can hold (roughly 100-150 thousand, depending on the field sizes).
type CampaignViewRepository struct {
	col *mongo.Collection
}

type campaignViewItem struct {
	Id             string    `bson:"id"`
	ViewedOnUtc    time.Time `bson:"viewedOnUtc"`
	ExternalUserId string    `bson:"externalUserId,omitempty"`
	IpAddress      string    `bson:"ipAddress"`
	UserAgent      string    `bson:"userAgent,omitempty"`
}

func (i campaignViewItem) toDomain(campaignId string) domain.CampaignView {
	return domain.CampaignView{
		Id:             i.Id,
		CampaignId:     campaignId,
		ViewedOnUtc:    i.ViewedOnUtc.UTC(),
		ExternalUserId: i.ExternalUserId,
		IpAddress:      i.IpAddress,
		UserAgent:      i.UserAgent,
	}
}

func (r *CampaignViewRepository) Create(ctx context.Context, view *domain.CampaignView) error {
	item := campaignViewItem{
		Id:             view.Id,
		ViewedOnUtc:    view.ViewedOnUtc,
		ExternalUserId: view.ExternalUserId,
		IpAddress:      view.IpAddress,
		UserAgent:      view.UserAgent,
	}
	_, err := r.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: view.CampaignId}},
		bson.D{{Key: "$push", Value: bson.D{{Key: "views", Value: item}}}},
		options.UpdateOne().SetUpsert(true),
	)
	return translateError(err)
}

// viewsPipeline turns the campaign document into one document per view in the period.
func viewsPipeline(campaignId string, period domain.Period) mongo.Pipeline {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "_id", Value: campaignId}}}},
		{{Key: "$unwind", Value: "$views"}},
		{{Key: "$replaceRoot", Value: bson.D{{Key: "newRoot", Value: "$views"}}}},
	}
	dateFilter := bson.D{}
	if !period.FromUtc.IsZero() {
		dateFilter = append(dateFilter, bson.E{Key: "$gte", Value: period.FromUtc})
	}
	if !period.ToUtc.IsZero() {
		dateFilter = append(dateFilter, bson.E{Key: "$lt", Value: period.ToUtc})
	}
	if len(dateFilter) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{{Key: "viewedOnUtc", Value: dateFilter}}}})
	}
	return pipeline
}

// visitorExpr identifies the visitor of a view: the external user id when
// informed, otherwise the IP address. The prefixes keep an external id from
// ever matching an IP.
var visitorExpr = bson.D{{Key: "$cond", Value: bson.D{
	{Key: "if", Value: bson.D{{Key: "$eq", Value: bson.A{bson.D{{Key: "$type", Value: "$externalUserId"}}, "string"}}}},
	{Key: "then", Value: bson.D{{Key: "$concat", Value: bson.A{"e:", "$externalUserId"}}}},
	{Key: "else", Value: bson.D{{Key: "$concat", Value: bson.A{"i:", "$ipAddress"}}}},
}}}

func (r *CampaignViewRepository) aggregate(ctx context.Context, pipeline mongo.Pipeline, out any) error {
	cursor, err := r.col.Aggregate(ctx, pipeline, options.Aggregate().SetAllowDiskUse(true))
	if err != nil {
		return err
	}
	return cursor.All(ctx, out)
}

type summaryResult struct {
	TotalViews  int64 `bson:"totalViews"`
	UniqueViews int64 `bson:"uniqueViews"`
}

func (r *CampaignViewRepository) Summary(ctx context.Context, campaignId string, period domain.Period) (domain.CampaignViewSummary, error) {
	// Grouping by visitor first counts unique views without holding every
	// visitor of the campaign in a single array.
	pipeline := append(viewsPipeline(campaignId, period),
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: visitorExpr},
			{Key: "views", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "totalViews", Value: bson.D{{Key: "$sum", Value: "$views"}}},
			{Key: "uniqueViews", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	)

	var results []summaryResult
	if err := r.aggregate(ctx, pipeline, &results); err != nil {
		return domain.CampaignViewSummary{}, err
	}
	if len(results) == 0 {
		return domain.CampaignViewSummary{}, nil
	}
	return domain.CampaignViewSummary{TotalViews: results[0].TotalViews, UniqueViews: results[0].UniqueViews}, nil
}

type dailyResult struct {
	Day         string `bson:"_id"`
	TotalViews  int64  `bson:"totalViews"`
	UniqueViews int64  `bson:"uniqueViews"`
}

func (r *CampaignViewRepository) DailySummary(ctx context.Context, campaignId string, period domain.Period) ([]domain.DailyCampaignViews, error) {
	pipeline := append(viewsPipeline(campaignId, period),
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				// $dateToString uses UTC unless a timezone is given.
				{Key: "day", Value: bson.D{{Key: "$dateToString", Value: bson.D{
					{Key: "format", Value: "%Y-%m-%d"},
					{Key: "date", Value: "$viewedOnUtc"},
				}}}},
				{Key: "visitor", Value: visitorExpr},
			}},
			{Key: "views", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$_id.day"},
			{Key: "totalViews", Value: bson.D{{Key: "$sum", Value: "$views"}}},
			{Key: "uniqueViews", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	)

	var results []dailyResult
	if err := r.aggregate(ctx, pipeline, &results); err != nil {
		return nil, err
	}

	days := make([]domain.DailyCampaignViews, len(results))
	for i, res := range results {
		date, err := time.Parse(time.DateOnly, res.Day)
		if err != nil {
			return nil, err
		}
		days[i] = domain.DailyCampaignViews{
			DateUtc:             date,
			CampaignViewSummary: domain.CampaignViewSummary{TotalViews: res.TotalViews, UniqueViews: res.UniqueViews},
		}
	}
	return days, nil
}

type userAgentVisitorResult struct {
	Id struct {
		UserAgent string `bson:"userAgent"`
		Visitor   string `bson:"visitor"`
	} `bson:"_id"`
	Views int64 `bson:"views"`
}

func (r *CampaignViewRepository) UserAgentVisitors(ctx context.Context, campaignId string, period domain.Period) ([]domain.VisitorUserAgentViews, error) {
	pipeline := append(viewsPipeline(campaignId, period),
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				// The field is omitted when empty.
				{Key: "userAgent", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$userAgent", ""}}}},
				{Key: "visitor", Value: visitorExpr},
			}},
			{Key: "views", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	)

	var results []userAgentVisitorResult
	if err := r.aggregate(ctx, pipeline, &results); err != nil {
		return nil, err
	}
	views := make([]domain.VisitorUserAgentViews, len(results))
	for i, res := range results {
		views[i] = domain.VisitorUserAgentViews{UserAgent: res.Id.UserAgent, Visitor: res.Id.Visitor, Views: res.Views}
	}
	return views, nil
}

type listResult struct {
	Items []campaignViewItem `bson:"items"`
	Total []struct {
		Count int64 `bson:"count"`
	} `bson:"total"`
}

func (r *CampaignViewRepository) List(ctx context.Context, campaignId string, period domain.Period, page repository.Pagination) ([]domain.CampaignView, int64, error) {
	pipeline := append(viewsPipeline(campaignId, period),
		bson.D{{Key: "$sort", Value: bson.D{{Key: "viewedOnUtc", Value: -1}, {Key: "id", Value: 1}}}},
		bson.D{{Key: "$facet", Value: bson.D{
			{Key: "items", Value: bson.A{
				bson.D{{Key: "$skip", Value: page.Offset}},
				bson.D{{Key: "$limit", Value: page.Limit}},
			}},
			{Key: "total", Value: bson.A{bson.D{{Key: "$count", Value: "count"}}}},
		}}},
	)

	var results []listResult
	if err := r.aggregate(ctx, pipeline, &results); err != nil {
		return nil, 0, err
	}
	if len(results) == 0 || len(results[0].Total) == 0 {
		return []domain.CampaignView{}, 0, nil
	}

	views := make([]domain.CampaignView, len(results[0].Items))
	for i, item := range results[0].Items {
		views[i] = item.toDomain(campaignId)
	}
	return views, results[0].Total[0].Count, nil
}
