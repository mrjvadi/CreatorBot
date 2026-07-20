// Package store لایه‌ی دسترسی داده‌ی archive-bot روی MongoDB (بدون Postgres).
//
// این بازنویسی از نسخه‌ی قبلیِ GORM/Postgres است؛ نام و امضای متدها عمداً
// یکی نگه داشته شده تا کد لایه‌ی tgbot بدون تغییرِ زیاد کار کند. برخلاف
// vpn-bot/member-bot، اینجا هیچ deleted_at/soft-delete ای وجود ندارد — مدل‌های
// قبلی هم گرچه از gorm.DeletedAt استفاده می‌کردند، هیچ مسیر «بازیابی» ای در
// کل کدبیس وجود نداشت؛ پس حذف واقعی (hard delete) دقیقاً همان رفتارِ
// قابل‌مشاهده‌ی قبلی است، بدون پیچیدگیِ اضافه.
//
// چندمستأجری: همه‌ی instanceهای archive-bot یک دیتابیسِ Mongo مشترک دارند
// (MONGO_DB=archive_bot، برای بهترین عملکرد به‌جای دیتابیسِ جدا به‌ازای هر
// instance) — جداسازیِ دیتای هر instance با فیلدِ instance_id روی هر سند
// انجام می‌شود، دقیقاً همان الگویی که uploader-bot با shared-core/docstore
// پیاده می‌کند. **مهم‌ترین نکته‌ی این فایل**: Search باید همیشه instance_id
// را در فیلترش داشته باشد — وگرنه جستجوی یک instance فایل‌های instanceِ
// دیگری از archive-bot را هم برمی‌گرداند (نشتِ دیتای بینِ‌مشتری‌ها).
//
// جستجوی فازی: به‌جای pg_trgm.similarity() (که در MongoDB self-hosted معادل
// ندارد)، هر File هنگام نوشتن یک فیلد ngrams (trigramهای کاراکتریِ متنِ
// نرمال‌شده‌ی title+tags+description، رجوع internal/search) ذخیره می‌کند.
// Search ایندکس‌شده روی همین فیلد یک candidate set ارزان می‌گیرد، سپس در Go
// امتیازِ Jaccard را دقیقاً با همان فرمولِ pg_trgm محاسبه و رتبه‌بندی می‌کند.
package store

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mrjvadi/creatorbot/archive-bot/internal/models"
	"github.com/mrjvadi/creatorbot/archive-bot/internal/search"
)

// similarityThreshold آستانه‌ی پذیرش نتیجه — معادلِ همان `similarity(...) > 0.1`
// که در نسخه‌ی Postgres همه‌جا استفاده می‌شد.
const similarityThreshold = 0.1

type Store struct {
	db         *mongo.Database
	instanceID string
}

func New(db *mongo.Database, instanceID string) *Store {
	return &Store{db: db, instanceID: instanceID}
}

// scoped فیلترِ instance_id را به filter اضافه می‌کند — هر خواندن/نوشتن در
// این فایل باید از این عبور کند، وگرنه instanceهای دیگرِ archive-bot که همین
// دیتابیس را شریک‌اند دیتای هم را می‌بینند (شاملِ نتایجِ Search).
func (s *Store) scoped(extra bson.M) bson.M {
	if extra == nil {
		extra = bson.M{}
	}
	extra["instance_id"] = s.instanceID
	return extra
}

// EnsureIndexes ایندکس‌های لازم را idempotent می‌سازد — باید یک‌بار در startup
// صدا زده شود (معادل AutoMigrate + CREATE EXTENSION pg_trgm/GIN index قبلی).
// instance_id کلیدِ پیشروی هر ایندکس است تا instanceهای مختلف دیتای هم را
// رد نکنند و ایندکسِ ngrams هم به‌ازای هر instance جدا فیلتر شود.
func (s *Store) EnsureIndexes(ctx context.Context) error {
	models := []struct {
		coll  string
		index mongo.IndexModel
	}{
		{"users", mongo.IndexModel{
			Keys:    bson.D{{Key: "instance_id", Value: 1}, {Key: "telegram_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"categories", mongo.IndexModel{
			Keys:    bson.D{{Key: "instance_id", Value: 1}, {Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"files", mongo.IndexModel{
			Keys: bson.D{{Key: "instance_id", Value: 1}, {Key: "category_id", Value: 1}},
		}},
		// ایندکس چندکلیدی روی (instance_id, ngrams) — پایه‌ی candidate-fetch
		// ارزانِ جستجوی فازی، همیشه محدود به instanceِ خودش.
		{"files", mongo.IndexModel{
			Keys: bson.D{{Key: "instance_id", Value: 1}, {Key: "ngrams", Value: 1}},
		}},
	}
	for _, m := range models {
		if _, err := s.db.Collection(m.coll).Indexes().CreateOne(ctx, m.index); err != nil {
			return err
		}
	}
	return nil
}

// ── اسناد داخلی Mongo (فقط همین فایل) ──────────────────────

type userDoc struct {
	ID         string    `bson:"_id"`
	InstanceID string    `bson:"instance_id"`
	CreatedAt  time.Time `bson:"created_at"`
	UpdatedAt  time.Time `bson:"updated_at"`
	TelegramID int64     `bson:"telegram_id"`
	Username   string    `bson:"username"`
	FirstName  string    `bson:"first_name"`
	IsBlocked  bool      `bson:"is_blocked"`
}

func (d *userDoc) toModel() *models.User {
	u := &models.User{
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
		TelegramID: d.TelegramID,
		Username:   d.Username,
		FirstName:  d.FirstName,
		IsBlocked:  d.IsBlocked,
	}
	u.ID, _ = uuid.Parse(d.ID)
	return u
}

type categoryDoc struct {
	ID         string    `bson:"_id"`
	InstanceID string    `bson:"instance_id"`
	CreatedAt  time.Time `bson:"created_at"`
	UpdatedAt  time.Time `bson:"updated_at"`
	Name       string    `bson:"name"`
}

func (d *categoryDoc) toModel() *models.Category {
	c := &models.Category{CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt, Name: d.Name}
	c.ID, _ = uuid.Parse(d.ID)
	return c
}

type fileDoc struct {
	ID          string    `bson:"_id"`
	InstanceID  string    `bson:"instance_id"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
	FileID      string    `bson:"file_id"`
	FileType    string    `bson:"file_type"`
	Title       string    `bson:"title"`
	Tags        string    `bson:"tags"`
	Description string    `bson:"description"`
	CategoryID  *string   `bson:"category_id"`
	UploaderID  int64     `bson:"uploader_id"`
	// Ngrams رجوع بالای فایل — پیش‌محاسبه‌شده هنگام نوشتن، صرفاً برای Search.
	Ngrams []string `bson:"ngrams"`
}

func (d *fileDoc) toModel() models.File {
	f := models.File{
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
		FileID: d.FileID, FileType: d.FileType, Title: d.Title,
		Tags: d.Tags, Description: d.Description, UploaderID: d.UploaderID,
	}
	f.ID, _ = uuid.Parse(d.ID)
	if d.CategoryID != nil {
		if cid, err := uuid.Parse(*d.CategoryID); err == nil {
			f.CategoryID = &cid
		}
	}
	return f
}

func (s *Store) fileToDoc(f *models.File) fileDoc {
	d := fileDoc{
		ID: f.ID.String(), InstanceID: s.instanceID, CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt,
		FileID: f.FileID, FileType: f.FileType, Title: f.Title,
		Tags: f.Tags, Description: f.Description, UploaderID: f.UploaderID,
		Ngrams: computeNgrams(f.Title, f.Tags, f.Description),
	}
	if f.CategoryID != nil {
		cid := f.CategoryID.String()
		d.CategoryID = &cid
	}
	return d
}

func computeNgrams(title, tags, description string) []string {
	normalized := search.Normalize(title + " " + tags + " " + description)
	return search.Trigrams(normalized)
}

// ---- User ----

func (s *Store) FindUserByTelegramID(ctx context.Context, id int64) (*models.User, error) {
	var d userDoc
	err := s.db.Collection("users").FindOne(ctx, s.scoped(bson.M{"telegram_id": id})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

// UpsertUser کاربر را با (instance_id,TelegramID) پیدا یا می‌سازد و *u را با
// سندِ کاملِ نهایی پر می‌کند — معادلِ FirstOrCreate قبلی، ولی اتمیک.
func (s *Store) UpsertUser(ctx context.Context, u *models.User) error {
	now := time.Now()
	filter := s.scoped(bson.M{"telegram_id": u.TelegramID})
	update := bson.M{
		"$set": bson.M{
			"username":   u.Username,
			"first_name": u.FirstName,
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"_id":         uuid.New().String(),
			"instance_id": s.instanceID,
			"telegram_id": u.TelegramID,
			"is_blocked":  false,
			"created_at":  now,
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var d userDoc
	if err := s.db.Collection("users").FindOneAndUpdate(ctx, filter, update, opts).Decode(&d); err != nil {
		return err
	}
	*u = *d.toModel()
	return nil
}

// UpsertUserByID الگوی fire-and-forget قبلی — بدون بازگرداندنِ خطا، چون
// فراخوان (onStart) هیچ‌وقت نتیجه را بررسی نمی‌کرد.
func (s *Store) UpsertUserByID(ctx context.Context, telegramID int64, username, firstName string) {
	u := &models.User{TelegramID: telegramID, Username: username, FirstName: firstName}
	_ = s.UpsertUser(ctx, u)
}

// ---- Category ----

func (s *Store) ListCategories(ctx context.Context) ([]models.Category, error) {
	cur, err := s.db.Collection("categories").Find(ctx, s.scoped(nil))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []categoryDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	cats := make([]models.Category, 0, len(docs))
	for _, d := range docs {
		cats = append(cats, *d.toModel())
	}
	return cats, nil
}

// FindOrCreateCategory اتمیک است — دو کلیک هم‌زمان روی «دسته جدید» با یک نام
// (روی همان instance) هرگز دو رکورد نمی‌سازند.
func (s *Store) FindOrCreateCategory(ctx context.Context, name string) (*models.Category, error) {
	now := time.Now()
	filter := s.scoped(bson.M{"name": name})
	update := bson.M{
		"$setOnInsert": bson.M{
			"_id": uuid.New().String(), "instance_id": s.instanceID, "name": name,
			"created_at": now, "updated_at": now,
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var d categoryDoc
	err := s.db.Collection("categories").FindOneAndUpdate(ctx, filter, update, opts).Decode(&d)
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

func (s *Store) FindCategoryByID(ctx context.Context, idStr string) (*models.Category, error) {
	var d categoryDoc
	err := s.db.Collection("categories").FindOne(ctx, s.scoped(bson.M{"_id": idStr})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

// ---- File ----

func (s *Store) CreateFile(ctx context.Context, f *models.File) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	now := time.Now()
	f.CreatedAt, f.UpdatedAt = now, now
	_, err := s.db.Collection("files").InsertOne(ctx, s.fileToDoc(f))
	return err
}

func (s *Store) FindFilesByCategory(ctx context.Context, catIDStr string) ([]models.File, error) {
	cur, err := s.db.Collection("files").Find(ctx,
		s.scoped(bson.M{"category_id": catIDStr}),
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []fileDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	files := make([]models.File, 0, len(docs))
	for _, d := range docs {
		files = append(files, d.toModel())
	}
	return files, nil
}

func (s *Store) DeleteFile(ctx context.Context, idStr string) error {
	_, err := s.db.Collection("files").DeleteOne(ctx, s.scoped(bson.M{"_id": idStr}))
	return err
}

func (s *Store) FindFileByID(ctx context.Context, idStr string) (*models.File, error) {
	var d fileDoc
	err := s.db.Collection("files").FindOne(ctx, s.scoped(bson.M{"_id": idStr})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m := d.toModel()
	return &m, nil
}

// scoredFile یک نامزدِ جستجو به‌همراه امتیازِ Jaccard محاسبه‌شده.
type scoredFile struct {
	file  models.File
	score float64
}

// Search جستجوی فازیِ متنی — معادلِ app-side برای pg_trgm.similarity() قبلی:
// ۱) query نرمال‌سازی و به trigram تبدیل می‌شود، ۲) با ایندکسِ (instance_id,
// ngrams) یک candidate set ارزان و محدود به همین instance گرفته می‌شود
// ($in)، ۳) در Go امتیازِ Jaccard دقیق محاسبه، فیلترِ >0.1 اعمال و نتایج
// نزولی مرتب و به limit محدود می‌شوند.
//
// instance_id در فیلتر این‌جا **حیاتی** است: بدونش، جستجوی یک مشتری فایل‌های
// یک instanceِ archive-bot کاملاً متفاوت را هم برمی‌گرداند — نشتِ دیتا بینِ
// دو مشتری، نه فقط یک باگِ معمولی.
func (s *Store) Search(ctx context.Context, query string, limit int) ([]models.File, error) {
	queryTrigrams := search.Trigrams(search.Normalize(query))
	if len(queryTrigrams) == 0 {
		return nil, nil
	}

	cur, err := s.db.Collection("files").Find(ctx, s.scoped(bson.M{"ngrams": bson.M{"$in": queryTrigrams}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []fileDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}

	scored := make([]scoredFile, 0, len(docs))
	for _, d := range docs {
		sc := search.Similarity(queryTrigrams, d.Ngrams)
		if sc > similarityThreshold {
			scored = append(scored, scoredFile{file: d.toModel(), score: sc})
		}
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].score > scored[j].score })

	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}
	files := make([]models.File, 0, len(scored))
	for _, sf := range scored {
		files = append(files, sf.file)
	}
	return files, nil
}

// RelatedFiles فایل‌های مرتبط را با CategoryID یا اولین تگ مشترک پیدا می‌کند —
// migrate شده برای parity؛ مثل قبل در هیچ handler ای فعلاً سیم‌کشی نشده.
func (s *Store) RelatedFiles(ctx context.Context, file models.File, limit int) ([]models.File, error) {
	filter := s.scoped(bson.M{"_id": bson.M{"$ne": file.ID.String()}})
	if file.CategoryID != nil {
		filter["category_id"] = file.CategoryID.String()
	} else if file.Tags != "" {
		firstTag := file.Tags
		if idx := indexOfComma(file.Tags); idx >= 0 {
			firstTag = file.Tags[:idx]
		}
		filter["tags"] = bson.M{"$regex": firstTag}
	} else {
		return nil, nil
	}

	cur, err := s.db.Collection("files").Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []fileDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	files := make([]models.File, 0, len(docs))
	for _, d := range docs {
		files = append(files, d.toModel())
	}
	return files, nil
}

func indexOfComma(s string) int {
	for i, r := range s {
		if r == ',' {
			return i
		}
	}
	return -1
}
