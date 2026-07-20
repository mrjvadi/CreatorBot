//go:build integration

// این تست‌ها نیازمند یک MongoDB واقعی هستند (MONGO_TEST_URI، پیش‌فرض
// mongodb://mongouser:m0ng0_s3cr3t_2024@127.0.0.1:27017/?authSource=admin&directConnection=true).
// اجرا: go test -tags=integration ./internal/store/...
package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mrjvadi/creatorbot/archive-bot/internal/models"
)

// testDB یک دیتابیسِ Mongo موقت می‌سازد — چند instanceِ Store می‌توانند روی
// همین یک db ساخته شوند تا رفتارِ واقعیِ «چند instance، یک دیتابیسِ مشترک»
// شبیه‌سازی شود (دقیقاً همان مدلِ production).
func testDB(t *testing.T) *mongo.Database {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		uri = "mongodb://mongouser:m0ng0_s3cr3t_2024@127.0.0.1:27017/?authSource=admin&directConnection=true"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("mongo connect: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("mongo ping: %v", err)
	}
	dbName := "archivebot_test_" + uuid.NewString()[:8]
	db := client.Database(dbName)
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})
	return db
}

func testStore(t *testing.T, instanceID string) *Store {
	t.Helper()
	db := testDB(t)
	s := New(db, instanceID)
	if err := s.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	return s
}

// TestSearch_RanksBySimilarity چند فایل با عنوان‌های نزدیک/دور ذخیره می‌شوند،
// و جستجو باید نزدیک‌ترین را اول برگرداند و کاملاً بی‌ربط را حذف کند —
// معادلِ رفتارِ pg_trgm.similarity() > 0.1 قبلی.
func TestSearch_RanksBySimilarity(t *testing.T) {
	s := testStore(t, "bot_test1")
	ctx := context.Background()

	files := []*models.File{
		{Title: "آموزش زبان برنامه‌نویسی گو", Tags: "golang,backend", UploaderID: 1},
		{Title: "آموزش زبان برنامه نویسی گو", Tags: "golang", UploaderID: 1}, // با نیم‌فاصله/بدون
		{Title: "آموزش پایتون مقدماتی", Tags: "python,backend", UploaderID: 1},
		{Title: "فیلم سینمایی اکشن", Tags: "movie,action", UploaderID: 1}, // کاملاً بی‌ربط
	}
	for _, f := range files {
		if err := s.CreateFile(ctx, f); err != nil {
			t.Fatalf("create file %q: %v", f.Title, err)
		}
	}

	results, err := s.Search(ctx, "آموزش زبان برنامه‌نویسی گلنگ", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	// اولین نتیجه باید یکی از دو فایلِ «آموزش زبان برنامه‌نویسی گو» باشد،
	// نه فیلمِ کاملاً بی‌ربط.
	top := results[0].Title
	if top != files[0].Title && top != files[1].Title {
		t.Errorf("expected top result to be a golang tutorial, got %q", top)
	}
	for _, r := range results {
		if r.Title == "فیلم سینمایی اکشن" {
			t.Errorf("unrelated file %q should have been filtered by similarity threshold", r.Title)
		}
	}
}

// TestSearch_DiacriticAndZWNJInsensitive تفاوتِ اعراب/نیم‌فاصله بین متنِ
// ذخیره‌شده و کوئری نباید امتیازِ شباهت را صفر کند — چون هر دو با همان
// search.Normalize پردازش می‌شوند.
func TestSearch_DiacriticAndZWNJInsensitive(t *testing.T) {
	s := testStore(t, "bot_test1")
	ctx := context.Background()

	f := &models.File{Title: "کِتاب‌های برنامه‌نویسی", Tags: "کتاب", UploaderID: 1}
	if err := s.CreateFile(ctx, f); err != nil {
		t.Fatalf("create file: %v", err)
	}

	results, err := s.Search(ctx, "کتابهای برنامه نویسی", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected the diacritic/ZWNJ-normalized query to match, got %d results", len(results))
	}
}

// TestFindOrCreateCategory_NoDuplicate دو فراخوانیِ هم‌زمان با یک نام باید
// فقط یک رکورد بسازند (findOneAndUpdate اتمیک، نه race مثلِ FirstOrCreate قبلی).
func TestFindOrCreateCategory_NoDuplicate(t *testing.T) {
	s := testStore(t, "bot_test1")
	ctx := context.Background()

	const n = 10
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			_, err := s.FindOrCreateCategory(ctx, "همان دسته")
			errs <- err
		}()
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("find or create category: %v", err)
		}
	}

	cats, err := s.ListCategories(ctx)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(cats) != 1 {
		t.Fatalf("expected exactly 1 category after concurrent calls, got %d", len(cats))
	}
}

// TestEnsureIndexes_CategoryNameUnique ایندکسِ یکتا روی (instance_id,name) را
// در سطحِ خودِ دیتابیس (نه فقط منطقِ FindOrCreateCategory) تأیید می‌کند.
func TestEnsureIndexes_CategoryNameUnique(t *testing.T) {
	s := testStore(t, "bot_test1")
	ctx := context.Background()

	now := time.Now()
	doc1 := categoryDoc{ID: uuid.NewString(), InstanceID: s.instanceID, Name: "یکتا", CreatedAt: now, UpdatedAt: now}
	if _, err := s.db.Collection("categories").InsertOne(ctx, doc1); err != nil {
		t.Fatalf("insert 1: %v", err)
	}
	doc2 := categoryDoc{ID: uuid.NewString(), InstanceID: s.instanceID, Name: "یکتا", CreatedAt: now, UpdatedAt: now}
	_, err := s.db.Collection("categories").InsertOne(ctx, doc2)
	if !mongo.IsDuplicateKeyError(err) {
		t.Fatalf("expected duplicate-key error for repeated category name, got: %v", err)
	}
}

// TestUpsertUser_PopulatesFullDocument تأیید می‌کند FindOneAndUpdate اتمیک
// دقیقاً همان رفتارِ FirstOrCreate قبلی را دارد: بعد از فراخوانیِ دوم با
// همان TelegramID، *u باید ID اولیه را حفظ کند (نه یک رکورد جدید).
func TestUpsertUser_PopulatesFullDocument(t *testing.T) {
	s := testStore(t, "bot_test1")
	ctx := context.Background()

	u1 := &models.User{TelegramID: 424242, Username: "a"}
	if err := s.UpsertUser(ctx, u1); err != nil {
		t.Fatalf("upsert 1: %v", err)
	}
	firstID := u1.ID

	u2 := &models.User{TelegramID: 424242, Username: "b"}
	if err := s.UpsertUser(ctx, u2); err != nil {
		t.Fatalf("upsert 2: %v", err)
	}
	if u2.ID != firstID {
		t.Fatalf("expected same user ID on second upsert, got %s vs %s", u2.ID, firstID)
	}
	if u2.Username != "b" {
		t.Fatalf("expected username to be updated to 'b', got %q", u2.Username)
	}

	count, err := s.db.Collection("users").CountDocuments(ctx, bson.M{"telegram_id": 424242})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 user document, got %d", count)
	}
}

// TestInstanceIsolation_SearchNeverLeaksAcrossInstances مهم‌ترین تستِ این
// فایل: دو instanceِ archive-bot که یک دیتابیس را شریک‌اند — جستجوی instance A
// هرگز نباید فایلِ آپلودشده در instance B را برگرداند، حتی وقتی عنوان‌ها
// دقیقاً یکسان‌اند (رجوع کامنتِ بالای Search در store.go).
func TestInstanceIsolation_SearchNeverLeaksAcrossInstances(t *testing.T) {
	db := testDB(t)
	sA := New(db, "bot_A")
	sB := New(db, "bot_B")
	if err := sA.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	ctx := context.Background()

	fA := &models.File{Title: "آموزش گلنگ پیشرفته", Tags: "golang", UploaderID: 1}
	if err := sA.CreateFile(ctx, fA); err != nil {
		t.Fatalf("create file on A: %v", err)
	}
	// عنوانِ کاملاً یکسان روی instance دیگر — باید در جستجوی A هرگز دیده نشود.
	fB := &models.File{Title: "آموزش گلنگ پیشرفته", Tags: "golang", UploaderID: 2}
	if err := sB.CreateFile(ctx, fB); err != nil {
		t.Fatalf("create file on B: %v", err)
	}

	resultsA, err := sA.Search(ctx, "آموزش گلنگ پیشرفته", 10)
	if err != nil {
		t.Fatalf("search on A: %v", err)
	}
	if len(resultsA) != 1 {
		t.Fatalf("instance A search leaked instance B's file — expected 1 result, got %d", len(resultsA))
	}
	if resultsA[0].UploaderID != 1 {
		t.Fatalf("instance A search returned instance B's document (uploader_id=%d)", resultsA[0].UploaderID)
	}

	resultsB, err := sB.Search(ctx, "آموزش گلنگ پیشرفته", 10)
	if err != nil {
		t.Fatalf("search on B: %v", err)
	}
	if len(resultsB) != 1 || resultsB[0].UploaderID != 2 {
		t.Fatalf("instance B search leaked instance A's file: %+v", resultsB)
	}
}

// TestInstanceIsolation_CategorySameName دو instance می‌توانند مستقلاً همان
// نامِ دسته را تعریف کنند — یکتاییِ name باید per-instance باشد.
func TestInstanceIsolation_CategorySameName(t *testing.T) {
	db := testDB(t)
	sA := New(db, "bot_A")
	sB := New(db, "bot_B")
	if err := sA.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	ctx := context.Background()

	if _, err := sA.FindOrCreateCategory(ctx, "عمومی"); err != nil {
		t.Fatalf("create category on A: %v", err)
	}
	if _, err := sB.FindOrCreateCategory(ctx, "عمومی"); err != nil {
		t.Fatalf("create same category name on B (must not collide with A's unique index): %v", err)
	}

	catsA, err := sA.ListCategories(ctx)
	if err != nil {
		t.Fatalf("list categories A: %v", err)
	}
	if len(catsA) != 1 {
		t.Fatalf("instance A leaked instance B's category: %+v", catsA)
	}
}
