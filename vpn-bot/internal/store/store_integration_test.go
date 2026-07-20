//go:build integration

// این تست‌ها نیازمند یک MongoDB واقعیِ در دسترس هستند (MONGO_TEST_URI، پیش‌فرض
// mongodb://mongouser:m0ng0_s3cr3t_2024@127.0.0.1:27017/?authSource=admin&directConnection=true).
// با تگ build جدا نگه داشته شده‌اند تا `go test ./...` معمولی (بدون DB) را
// نشکنند؛ اجرا با: go test -tags=integration ./internal/store/...
package store

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mrjvadi/creatorbot/vpn-bot/internal/models"
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
	dbName := "vpnbot_test_" + uuid.NewString()[:8]
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

// TestDeductBalanceIfEnough_NoDoubleSpend شبیه‌سازیِ دو کلیک هم‌زمان روی برداشت:
// از موجودی ۱۰۰ فقط باید یکی از دو کسرِ ۷۰تایی موفق شود، نه هر دو.
func TestDeductBalanceIfEnough_NoDoubleSpend(t *testing.T) {
	s := testStore(t, "bot_test1")
	ctx := context.Background()

	u := &models.User{TelegramID: 111222333}
	if err := s.UpsertUser(ctx, u); err != nil {
		t.Fatalf("upsert user: %v", err)
	}
	if err := s.UpdateBalance(ctx, u.ID, 100); err != nil {
		t.Fatalf("seed balance: %v", err)
	}

	var wg sync.WaitGroup
	results := make([]bool, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ok, err := s.DeductBalanceIfEnough(ctx, u.ID, 70)
			if err != nil {
				t.Errorf("deduct: %v", err)
				return
			}
			results[idx] = ok
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, ok := range results {
		if ok {
			successCount++
		}
	}
	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful deduction (double-spend guard), got %d", successCount)
	}

	final, err := s.FindUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if final.Balance != 30 {
		t.Fatalf("expected final balance 30 (100-70), got %v — balance guard failed", final.Balance)
	}
}

// TestClaimOnlinePayment_NoDuplicateActivation شبیه‌سازیِ کلیک تکراری روی
// «پرداخت کردم»: فقط یکی از دو claim با همان (gateway, ref_code) باید موفق شود.
func TestClaimOnlinePayment_NoDuplicateActivation(t *testing.T) {
	s := testStore(t, "bot_test1")
	ctx := context.Background()

	u := &models.User{TelegramID: 444555666}
	if err := s.UpsertUser(ctx, u); err != nil {
		t.Fatalf("upsert user: %v", err)
	}

	refID := "dup-ref-" + uuid.NewString()
	var wg sync.WaitGroup
	claimed := make([]bool, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			p := &models.Payment{UserID: u.ID, Amount: 50, Gateway: "zarinpal", RefCode: refID}
			ok, err := s.ClaimOnlinePayment(ctx, p)
			if err != nil {
				t.Errorf("claim: %v", err)
				return
			}
			claimed[idx] = ok
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, ok := range claimed {
		if ok {
			successCount++
		}
	}
	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful claim (dedup guard), got %d", successCount)
	}
}

// TestClaimPendingPayment_NoDoubleCredit شبیه‌سازیِ دو کلیک هم‌زمان ادمین روی
// «تأیید پرداخت»: فقط یکی باید بتواند status را از pending به confirmed ببرد
// (رفع باگ واقعیِ duplicate-credit).
func TestClaimPendingPayment_NoDoubleCredit(t *testing.T) {
	s := testStore(t, "bot_test1")
	ctx := context.Background()

	u := &models.User{TelegramID: 777888999}
	if err := s.UpsertUser(ctx, u); err != nil {
		t.Fatalf("upsert user: %v", err)
	}
	p := &models.Payment{UserID: u.ID, Amount: 40, Gateway: "card", Status: "pending"}
	if err := s.CreatePayment(ctx, p); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	var wg sync.WaitGroup
	claimedCount := 0
	var mu sync.Mutex
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := s.ClaimPendingPayment(ctx, p.ID, "confirmed")
			if err != nil {
				t.Errorf("claim pending: %v", err)
				return
			}
			if claimed != nil {
				mu.Lock()
				claimedCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if claimedCount != 1 {
		t.Fatalf("expected exactly 1 successful claim of the pending payment, got %d", claimedCount)
	}
}

// TestEnsureIndexes_PartialUniqueVisible یک چک سریع که ایندکسِ partial unique
// واقعاً روی کالکشن payments ساخته شده — با اسم مشخص قابل‌پیدا کردن است.
func TestEnsureIndexes_PartialUniqueVisible(t *testing.T) {
	s := testStore(t, "bot_test1")
	ctx := context.Background()
	cur, err := s.db.Collection("payments").Indexes().List(ctx)
	if err != nil {
		t.Fatalf("list indexes: %v", err)
	}
	defer cur.Close(ctx)
	var found bool
	for cur.Next(ctx) {
		var idx bson.M
		if err := cur.Decode(&idx); err != nil {
			continue
		}
		if idx["unique"] == true {
			found = true
		}
	}
	if !found {
		t.Fatal("expected a unique index on payments collection")
	}
}

// TestInstanceIsolation_SameTelegramIDDifferentInstances دو instanceِ vpn-bot
// که یک دیتابیسِ Mongo مشترک دارند (رجوع Context بالای store.go) — همان
// کاربرِ تلگرامی می‌تواند هم‌زمان مشتریِ هر دو باشد، پس یکتاییِ telegram_id
// باید per-instance باشد، نه سراسری. این تست دقیقاً همان سناریوی واقعی‌ای که
// باعثِ نوشتنِ این migration شد را بازتولید می‌کند.
func TestInstanceIsolation_SameTelegramIDDifferentInstances(t *testing.T) {
	db := testDB(t)
	sA := New(db, "bot_A")
	sB := New(db, "bot_B")
	if err := sA.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("ensure indexes A: %v", err)
	}
	ctx := context.Background()

	const sharedTelegramID = 55501234
	uA := &models.User{TelegramID: sharedTelegramID, Username: "via-instance-A"}
	if err := sA.UpsertUser(ctx, uA); err != nil {
		t.Fatalf("upsert user on instance A: %v", err)
	}
	uB := &models.User{TelegramID: sharedTelegramID, Username: "via-instance-B"}
	if err := sB.UpsertUser(ctx, uB); err != nil {
		t.Fatalf("upsert user on instance B (must NOT collide with A's unique index): %v", err)
	}
	if uA.ID == uB.ID {
		t.Fatal("expected different user documents per instance, got the same ID")
	}

	// instance A نباید کاربرِ instance B را در ListUsers ببیند.
	usersA, err := sA.ListUsers(ctx)
	if err != nil {
		t.Fatalf("list users A: %v", err)
	}
	if len(usersA) != 1 || usersA[0].Username != "via-instance-A" {
		t.Fatalf("instance A leaked instance B's user data: %+v", usersA)
	}
	usersB, err := sB.ListUsers(ctx)
	if err != nil {
		t.Fatalf("list users B: %v", err)
	}
	if len(usersB) != 1 || usersB[0].Username != "via-instance-B" {
		t.Fatalf("instance B leaked instance A's user data: %+v", usersB)
	}

	// instance A نباید بتواند با ID کاربرِ instance B کار کند (cross-instance FindUserByID).
	if leaked, err := sA.FindUserByID(ctx, uB.ID); err == nil && leaked != nil {
		t.Fatal("instance A could read instance B's user by ID — isolation broken")
	}
}

// TestInstanceIsolation_DiscountCodeSameCode دو instance می‌توانند مستقلاً
// همان کدِ تخفیف را تعریف کنند — یکتاییِ code باید per-instance باشد.
func TestInstanceIsolation_DiscountCodeSameCode(t *testing.T) {
	db := testDB(t)
	sA := New(db, "bot_A")
	sB := New(db, "bot_B")
	if err := sA.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	ctx := context.Background()

	codeA := &models.DiscountCode{Code: "SUMMER50", Percent: 50, MaxUse: 10, IsActive: true}
	if err := sA.CreateDiscountCode(ctx, codeA); err != nil {
		t.Fatalf("create discount code on A: %v", err)
	}
	codeB := &models.DiscountCode{Code: "SUMMER50", Percent: 20, MaxUse: 5, IsActive: true}
	if err := sB.CreateDiscountCode(ctx, codeB); err != nil {
		t.Fatalf("create same code on B (must not collide with A's unique index): %v", err)
	}

	foundOnB, err := sB.FindDiscountCode(ctx, "SUMMER50")
	if err != nil {
		t.Fatalf("find discount code on B: %v", err)
	}
	if foundOnB.Percent != 20 {
		t.Fatalf("instance B got instance A's discount code (percent=%v, want 20) — isolation broken", foundOnB.Percent)
	}
}
