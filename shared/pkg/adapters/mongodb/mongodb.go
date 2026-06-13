// Package mongodb implements ports.DocumentStore using the official Go MongoDB driver.
// To swap to a different document DB: implement ports.DocumentStore in a new package.
package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Config holds MongoDB connection options.
type Config struct {
	URI      string // mongodb://user:pass@host:27017
	Database string // نام دیتابیس — یک DB برای همه ربات‌ها
}

// Store wraps *mongo.Database و implements ports.DocumentStore.
type Store struct {
	client *mongo.Client
	db     *mongo.Database
}

var _ ports.DocumentStore = (*Store)(nil)

// New connects to MongoDB and returns a ports.DocumentStore.
func New(cfg Config) (*Store, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, fmt.Errorf("mongodb: connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongodb: ping: %w", err)
	}

	return &Store{
		client: client,
		db:     client.Database(cfg.Database),
	}, nil
}

// Database raw *mongo.Database را برمی‌گرداند.
// برای configstore و سایر کدهایی که مستقیماً به driver نیاز دارند.
func (s *Store) Database() *mongo.Database { return s.db }

func (s *Store) Ping(ctx context.Context) error {
	return s.client.Ping(ctx, nil)
}

func (s *Store) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

func (s *Store) Collection(name string) ports.Collection {
	return &collection{coll: s.db.Collection(name)}
}

// ---- collection ----

type collection struct {
	coll *mongo.Collection
}

var _ ports.Collection = (*collection)(nil)

func (c *collection) InsertOne(ctx context.Context, doc any) (string, error) {
	res, err := c.coll.InsertOne(ctx, doc)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", res.InsertedID), nil
}

func (c *collection) FindOne(ctx context.Context, filter any, result any) error {
	return c.coll.FindOne(ctx, filter).Decode(result)
}

func (c *collection) Find(ctx context.Context, filter any, results any, opts ...ports.FindOption) error {
	cfg := &ports.FindConfig{}
	for _, o := range opts {
		o(cfg)
	}

	findOpts := options.Find()
	if cfg.Limit > 0 {
		findOpts.SetLimit(cfg.Limit)
	}
	if cfg.Skip > 0 {
		findOpts.SetSkip(cfg.Skip)
	}
	if cfg.Sort != nil {
		findOpts.SetSort(cfg.Sort)
	}

	cur, err := c.coll.Find(ctx, filter, findOpts)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)
	return cur.All(ctx, results)
}

func (c *collection) UpdateOne(ctx context.Context, filter any, update any) error {
	_, err := c.coll.UpdateOne(ctx, filter, update)
	return err
}

func (c *collection) DeleteOne(ctx context.Context, filter any) error {
	_, err := c.coll.DeleteOne(ctx, filter)
	return err
}

func (c *collection) CountDocuments(ctx context.Context, filter any) (int64, error) {
	return c.coll.CountDocuments(ctx, filter)
}

func (c *collection) CreateIndex(ctx context.Context, keys any, unique bool) error {
	model := mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetUnique(unique),
	}
	_, err := c.coll.Indexes().CreateOne(ctx, model)
	return err
}

// ---- Filter Builder ----

// Filter یک helper برای ساخت bson.D filter است.
// استفاده: Filter("instance_id", id, "status", "active")
func Filter(kv ...any) bson.D {
	d := bson.D{}
	for i := 0; i+1 < len(kv); i += 2 {
		key, _ := kv[i].(string)
		d = append(d, bson.E{Key: key, Value: kv[i+1]})
	}
	return d
}

// Set یک bson update document می‌سازد.
// استفاده: Set("status", "active", "updated_at", time.Now())
func Set(kv ...any) bson.D {
	d := bson.D{}
	for i := 0; i+1 < len(kv); i += 2 {
		key, _ := kv[i].(string)
		d = append(d, bson.E{Key: key, Value: kv[i+1]})
	}
	return bson.D{{Key: "$set", Value: d}}
}
