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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mrjvadi/creatorbot/member-bot/internal/models"
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
	dbName := "memberbot_test_" + uuid.NewString()[:8]
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

// TestAddBotMembership_InsertThenUpdate یک عضویتِ تازه push می‌شود (چون
// عضویتی برای این channel وجود نداشت)، و فراخوانیِ دومِ همان (bot, channel)
// باید فقط last_verified را به‌روز کند، نه یک عنصرِ دومِ تکراری بسازد.
func TestAddBotMembership_InsertThenUpdate(t *testing.T) {
	s := testStore(t, "bot_test1")
	ctx := context.Background()

	bot := &models.CheckBot{Token: "enc-token", Username: "checkbot1", IsActive: true, RateLimit: 20}
	if err := s.CreateCheckBot(ctx, bot); err != nil {
		t.Fatalf("create bot: %v", err)
	}

	m := &models.BotChannelMembership{BotID: bot.ID, ChannelID: 555, JoinedAt: time.Now()}
	if err := s.AddBotMembership(ctx, m); err != nil {
		t.Fatalf("add membership (insert): %v", err)
	}
	// دومین فراخوانیِ همان (bot, channel) — نباید عنصر تکراری بسازد.
	if err := s.AddBotMembership(ctx, m); err != nil {
		t.Fatalf("add membership (update): %v", err)
	}

	bots, err := s.FindActiveBots(ctx)
	if err != nil {
		t.Fatalf("find active bots: %v", err)
	}
	if len(bots) != 1 {
		t.Fatalf("expected 1 bot, got %d", len(bots))
	}
	if len(bots[0].Memberships) != 1 {
		t.Fatalf("expected exactly 1 membership (no duplicate on second call), got %d", len(bots[0].Memberships))
	}
	if bots[0].Memberships[0].ChannelID != 555 {
		t.Errorf("expected channel_id 555, got %d", bots[0].Memberships[0].ChannelID)
	}
}

// TestAddBotMembership_MultipleChannels چند کانالِ متفاوت برای همان bot باید
// همه به آرایه اضافه شوند (نه جایگزینِ هم).
func TestAddBotMembership_MultipleChannels(t *testing.T) {
	s := testStore(t, "bot_test1")
	ctx := context.Background()

	bot := &models.CheckBot{Token: "enc-token", Username: "checkbot2", IsActive: true}
	if err := s.CreateCheckBot(ctx, bot); err != nil {
		t.Fatalf("create bot: %v", err)
	}

	for _, ch := range []int64{111, 222, 333} {
		m := &models.BotChannelMembership{BotID: bot.ID, ChannelID: ch, JoinedAt: time.Now()}
		if err := s.AddBotMembership(ctx, m); err != nil {
			t.Fatalf("add membership %d: %v", ch, err)
		}
	}

	bots, err := s.FindActiveBots(ctx)
	if err != nil {
		t.Fatalf("find active bots: %v", err)
	}
	if len(bots[0].Memberships) != 3 {
		t.Fatalf("expected 3 memberships, got %d", len(bots[0].Memberships))
	}
}

// TestClearBotMemberships همه‌ی آرایه‌های memberships را روی همه‌ی bot های
// *همین instance* خالی می‌کند.
func TestClearBotMemberships(t *testing.T) {
	s := testStore(t, "bot_test1")
	ctx := context.Background()

	bot := &models.CheckBot{Token: "t", Username: "b", IsActive: true}
	if err := s.CreateCheckBot(ctx, bot); err != nil {
		t.Fatalf("create bot: %v", err)
	}
	if err := s.AddBotMembership(ctx, &models.BotChannelMembership{BotID: bot.ID, ChannelID: 1, JoinedAt: time.Now()}); err != nil {
		t.Fatalf("add membership: %v", err)
	}
	if err := s.ClearBotMemberships(ctx); err != nil {
		t.Fatalf("clear: %v", err)
	}
	bots, _ := s.FindActiveBots(ctx)
	if len(bots[0].Memberships) != 0 {
		t.Fatalf("expected 0 memberships after clear, got %d", len(bots[0].Memberships))
	}
}

// TestOwnerUniqueTelegramID تضمینِ ایندکسِ یکتا روی (instance_id,telegram_id) —
// دومین CreateOwner با همان TelegramID روی همان instance باید خطای
// duplicate-key بدهد.
func TestOwnerUniqueTelegramID(t *testing.T) {
	s := testStore(t, "bot_test1")
	ctx := context.Background()

	o1 := &models.Owner{TelegramID: 999888777}
	if err := s.CreateOwner(ctx, o1); err != nil {
		t.Fatalf("create owner 1: %v", err)
	}
	o2 := &models.Owner{TelegramID: 999888777}
	err := s.CreateOwner(ctx, o2)
	if err == nil {
		t.Fatal("expected duplicate-key error for second owner with same telegram_id")
	}
	if !mongo.IsDuplicateKeyError(err) {
		t.Fatalf("expected duplicate-key error, got: %v", err)
	}
}

// TestInstanceIsolation_ClearBotMembershipsDoesNotAffectOtherInstance رفعِ
// باگِ واقعی: قبلاً ClearBotMemberships بدونِ instance_id، memberships همه‌ی
// instanceها را پاک می‌کرد. اینجا instance B نباید هیچ تاثیری روی داده‌ی
// instance A بگذارد.
func TestInstanceIsolation_ClearBotMembershipsDoesNotAffectOtherInstance(t *testing.T) {
	db := testDB(t)
	sA := New(db, "bot_A")
	sB := New(db, "bot_B")
	if err := sA.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	ctx := context.Background()

	botA := &models.CheckBot{Token: "tA", Username: "botA", IsActive: true}
	if err := sA.CreateCheckBot(ctx, botA); err != nil {
		t.Fatalf("create bot A: %v", err)
	}
	if err := sA.AddBotMembership(ctx, &models.BotChannelMembership{BotID: botA.ID, ChannelID: 42, JoinedAt: time.Now()}); err != nil {
		t.Fatalf("add membership A: %v", err)
	}

	botB := &models.CheckBot{Token: "tB", Username: "botB", IsActive: true}
	if err := sB.CreateCheckBot(ctx, botB); err != nil {
		t.Fatalf("create bot B: %v", err)
	}

	// instance B پاکسازیِ دوره‌ای خودش را اجرا می‌کند — نباید روی instance A اثر بگذارد.
	if err := sB.ClearBotMemberships(ctx); err != nil {
		t.Fatalf("clear on B: %v", err)
	}

	botsA, err := sA.FindActiveBots(ctx)
	if err != nil {
		t.Fatalf("find active bots A: %v", err)
	}
	if len(botsA) != 1 || len(botsA[0].Memberships) != 1 {
		t.Fatalf("instance B's ClearBotMemberships leaked into instance A: %+v", botsA)
	}
}

// TestInstanceIsolation_OwnerSameTelegramID همان کاربر می‌تواند مالکِ قفل در
// دو instanceِ مختلفِ member-bot باشد — یکتاییِ telegram_id باید per-instance
// باشد.
func TestInstanceIsolation_OwnerSameTelegramID(t *testing.T) {
	db := testDB(t)
	sA := New(db, "bot_A")
	sB := New(db, "bot_B")
	if err := sA.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	ctx := context.Background()

	const sharedTelegramID = 3001
	oA := &models.Owner{TelegramID: sharedTelegramID, Username: "owner-on-A"}
	if err := sA.CreateOwner(ctx, oA); err != nil {
		t.Fatalf("create owner on A: %v", err)
	}
	oB := &models.Owner{TelegramID: sharedTelegramID, Username: "owner-on-B"}
	if err := sB.CreateOwner(ctx, oB); err != nil {
		t.Fatalf("create owner on B (must not collide with A's unique index): %v", err)
	}

	ownersA, err := sA.ListOwners(ctx)
	if err != nil {
		t.Fatalf("list owners A: %v", err)
	}
	if len(ownersA) != 1 || ownersA[0].Username != "owner-on-A" {
		t.Fatalf("instance A leaked instance B's owner data: %+v", ownersA)
	}
}
