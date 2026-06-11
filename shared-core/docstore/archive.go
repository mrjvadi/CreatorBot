package docstore

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mrjvadi/creatorbot/shared-core/documents"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// ArchiveFileStore مدیریت فایل‌های آرشیو با جستجوی متنی.
type ArchiveFileStore struct {
	Base
}

func NewArchiveFileStore(ds ports.DocumentStore, instanceID string) *ArchiveFileStore {
	s := &ArchiveFileStore{Base: NewBase(ds, instanceID)}
	return s
}

// EnsureIndexes ایندکس‌های لازم را ایجاد می‌کند.
// باید یک‌بار هنگام راه‌اندازی ربات صدا زده شود.
func (s *ArchiveFileStore) EnsureIndexes(ctx context.Context) error {
	col := s.col("archive_files")
	// text index برای جستجوی فارسی
	if err := col.CreateIndex(ctx,
		bson.D{{Key: "search_text", Value: "text"}}, false); err != nil {
		return err
	}
	// compound index برای فیلتر instance + category
	return col.CreateIndex(ctx,
		bson.D{
			{Key: "instance_id", Value: 1},
			{Key: "category_id", Value: 1},
		}, false)
}

func (s *ArchiveFileStore) Create(ctx context.Context, f *documents.ArchiveFile) error {
	f.DocBase = s.newDocBase()
	// ساخت search_text از title + tags + description
	f.SearchText = f.Title
	for _, tag := range f.Tags {
		f.SearchText += " " + tag
	}
	f.SearchText += " " + f.Description
	_, err := s.col("archive_files").InsertOne(ctx, f)
	return err
}

// Search جستجوی متنی با MongoDB text index.
func (s *ArchiveFileStore) Search(ctx context.Context, query string, limit int) ([]documents.ArchiveFile, error) {
	var files []documents.ArchiveFile
	filter := bson.D{
		{Key: "instance_id", Value: s.instanceID},
		{Key: "$text", Value: bson.D{{Key: "$search", Value: query}}},
	}
	err := s.col("archive_files").Find(ctx, filter, &files,
		ports.WithLimit(int64(limit)),
		ports.WithSort(bson.D{{Key: "score", Value: bson.D{{Key: "$meta", Value: "textScore"}}}}),
	)
	return files, err
}

// FindByCategory فایل‌های یک دسته‌بندی.
func (s *ArchiveFileStore) FindByCategory(ctx context.Context, categoryID string, limit int) ([]documents.ArchiveFile, error) {
	var files []documents.ArchiveFile
	filter := s.baseFilter(bson.E{Key: "category_id", Value: categoryID})
	err := s.col("archive_files").Find(ctx, filter, &files,
		ports.WithLimit(int64(limit)),
		ports.WithSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	return files, err
}

// ArchiveCategoryStore مدیریت دسته‌بندی‌ها.
type ArchiveCategoryStore struct {
	Base
}

func NewArchiveCategoryStore(ds ports.DocumentStore, instanceID string) *ArchiveCategoryStore {
	return &ArchiveCategoryStore{Base: NewBase(ds, instanceID)}
}

func (s *ArchiveCategoryStore) Create(ctx context.Context, cat *documents.ArchiveCategory) error {
	cat.DocBase = s.newDocBase()
	_, err := s.col("archive_categories").InsertOne(ctx, cat)
	return err
}

func (s *ArchiveCategoryStore) List(ctx context.Context) ([]documents.ArchiveCategory, error) {
	var cats []documents.ArchiveCategory
	err := s.col("archive_categories").Find(ctx, s.baseFilter(), &cats,
		ports.WithSort(bson.D{{Key: "name", Value: 1}}))
	return cats, err
}

func (s *ArchiveCategoryStore) FindByName(ctx context.Context, name string) (*documents.ArchiveCategory, error) {
	var cat documents.ArchiveCategory
	err := s.col("archive_categories").FindOne(ctx,
		s.baseFilter(bson.E{Key: "name", Value: name}), &cat)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &cat, err
}
