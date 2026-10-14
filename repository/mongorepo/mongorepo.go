// Package mongorepo implements the repositories on top of MongoDB.
package mongorepo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

const (
	tenantsCollection       = "tenants"
	usersCollection         = "users"
	campaignsCollection     = "campaigns"
	campaignViewsCollection = "campaignViews"
)

// Open connects to MongoDB. monitor, when not nil, observes every command.
func Open(ctx context.Context, uri string, monitor *event.CommandMonitor) (*mongo.Client, error) {
	opts := options.Client().ApplyURI(uri)
	if monitor != nil {
		opts.SetMonitor(monitor)
	}
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	return client, nil
}

// EnsureIndexes migrates documents written by older versions and creates the
// indexes that enforce uniqueness and back the tenant filters. It is idempotent.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	if err := migrateUsers(ctx, db.Collection(usersCollection)); err != nil {
		return err
	}

	indexes := map[string][]mongo.IndexModel{
		tenantsCollection: {
			{Keys: bson.D{{Key: "name", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		usersCollection: {
			{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "createdAtUtc", Value: -1}}},
		},
		campaignsCollection: {
			{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "name", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "createdAtUtc", Value: -1}}},
		},
	}
	for collection, models := range indexes {
		if _, err := db.Collection(collection).Indexes().CreateMany(ctx, models); err != nil {
			return err
		}
	}
	return nil
}

// migrateUsers upgrades users from the previous layout, where the unique
// "name" was the login and "roles" was a list, to "username" + single "role".
// It must run before the unique username index is created.
func migrateUsers(ctx context.Context, col *mongo.Collection) error {
	_, err := col.UpdateMany(ctx,
		bson.D{{Key: "username", Value: bson.D{{Key: "$exists", Value: false}}}},
		mongo.Pipeline{
			{{Key: "$set", Value: bson.D{
				{Key: "username", Value: bson.D{{Key: "$toLower", Value: "$name"}}},
				{Key: "role", Value: bson.D{{Key: "$arrayElemAt", Value: bson.A{"$roles", 0}}}},
			}}},
			{{Key: "$unset", Value: "roles"}},
		},
	)
	if err != nil {
		return err
	}

	// The display name is no longer unique.
	specs, err := col.Indexes().ListSpecifications(ctx)
	if err != nil {
		var cmdErr mongo.CommandError
		if errors.As(err, &cmdErr) && cmdErr.Code == namespaceNotFoundCode {
			return nil // collection not created yet
		}
		return err
	}
	for _, spec := range specs {
		if spec.Name == "name_1" {
			return col.Indexes().DropOne(ctx, spec.Name)
		}
	}
	return nil
}

const namespaceNotFoundCode = 26

func New(client *mongo.Client, db *mongo.Database) *repository.Repositories {
	return repository.NewRepositories(repository.Constructors{
		Tenants: func() repository.TenantRepository {
			return &TenantRepository{col: db.Collection(tenantsCollection)}
		},
		Users: func() repository.UserRepository {
			return &UserRepository{col: db.Collection(usersCollection)}
		},
		Campaigns: func() repository.CampaignRepository {
			return &CampaignRepository{col: db.Collection(campaignsCollection)}
		},
		CampaignViews: func() repository.CampaignViewRepository {
			return &CampaignViewRepository{col: db.Collection(campaignViewsCollection)}
		},
		Ping: func(ctx context.Context) error {
			return client.Ping(ctx, nil)
		},
		Close: client.Disconnect,
	})
}

func translateError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, mongo.ErrNoDocuments):
		return repository.ErrNotFound
	case mongo.IsDuplicateKeyError(err):
		return repository.ErrDuplicate
	default:
		return err
	}
}

func updateResultError(res *mongo.UpdateResult, err error) error {
	if err != nil {
		return translateError(err)
	}
	if res.MatchedCount == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func exists(ctx context.Context, col *mongo.Collection, filter bson.D) (bool, error) {
	count, err := col.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func withExclude(filter bson.D, excludeId string) bson.D {
	if excludeId != "" {
		filter = append(filter, bson.E{Key: "_id", Value: bson.D{{Key: "$ne", Value: excludeId}}})
	}
	return filter
}

// list runs a paginated query and decodes each document with toDomain.
func list[D any, T any](ctx context.Context, col *mongo.Collection, filter bson.D, page repository.Pagination, toDomain func(D) T) ([]T, int64, error) {
	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAtUtc", Value: -1}, {Key: "_id", Value: 1}}).
		SetSkip(int64(page.Offset)).
		SetLimit(int64(page.Limit))

	cursor, err := col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}

	var docs []D
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, 0, err
	}

	items := make([]T, len(docs))
	for i, d := range docs {
		items[i] = toDomain(d)
	}
	return items, total, nil
}
