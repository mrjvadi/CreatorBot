// Package store لایه‌ی دسترسی داده‌ی member-bot روی MongoDB (بدون Postgres).
//
// بازنویسی از نسخه‌ی قبلیِ GORM/Postgres؛ نام/امضای متدها عمداً یکی نگه داشته
// شده تا کد لایه‌ی tgbot/dispatcher/scheduler بدون تغییر کار کند.
//
// چندمستأجری: همه‌ی instanceهای member-bot یک دیتابیسِ Mongo مشترک دارند
// (MONGO_DB=member_bot، برای بهترین عملکرد به‌جای دیتابیسِ جدا به‌ازای هر
// instance) — جداسازیِ دیتای هر instance با فیلدِ instance_id روی هر سند
// انجام می‌شود، دقیقاً همان الگویی که uploader-bot با shared-core/docstore
// پیاده می‌کند. instanceID یک‌بار در startup از BOT_TOKEN مشتق و به New پاس
// داده می‌شود؛ همه‌ی متدهای این فایل باید از scoped() استفاده کنند.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mrjvadi/creatorbot/member-bot/internal/models"
)

type Store struct {
	db         *mongo.Database
	instanceID string
}

func New(db *mongo.Database, instanceID string) *Store {
	return &Store{db: db, instanceID: instanceID}
}

// scoped فیلترِ instance_id را به filter اضافه می‌کند — هر خواندن/نوشتن در
// این فایل باید از این عبور کند، وگرنه instanceهای دیگرِ member-bot که همین
// دیتابیس را شریک‌اند دیتای هم را می‌بینند/دستکاری می‌کنند.
func (s *Store) scoped(extra bson.M) bson.M {
	if extra == nil {
		extra = bson.M{}
	}
	extra["instance_id"] = s.instanceID
	return extra
}

func notDeleted(extra bson.M) bson.M {
	if extra == nil {
		extra = bson.M{}
	}
	extra["deleted_at"] = nil
	return extra
}

// EnsureIndexes ایندکس‌های لازم را idempotent می‌سازد — باید یک‌بار در
// startup صدا زده شود (معادل AutoMigrate + uniqueIndexهای قبلی). instance_id
// کلیدِ پیشروی هر ایندکس است تا instanceهای مختلف دیتای هم را رد نکنند.
func (s *Store) EnsureIndexes(ctx context.Context) error {
	models := []struct {
		coll  string
		index mongo.IndexModel
	}{
		{"owners", mongo.IndexModel{
			Keys:    bson.D{{Key: "instance_id", Value: 1}, {Key: "telegram_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"locks", mongo.IndexModel{
			Keys:    bson.D{{Key: "instance_id", Value: 1}, {Key: "channel_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"check_bots", mongo.IndexModel{
			Keys: bson.D{{Key: "instance_id", Value: 1}, {Key: "memberships.channel_id", Value: 1}},
		}},
	}
	for _, m := range models {
		if _, err := s.db.Collection(m.coll).Indexes().CreateOne(ctx, m.index); err != nil {
			return err
		}
	}
	return nil
}

// ── اسناد داخلی Mongo ───────────────────────────────────────

type ownerDoc struct {
	ID         string     `bson:"_id"`
	InstanceID string     `bson:"instance_id"`
	CreatedAt  time.Time  `bson:"created_at"`
	UpdatedAt  time.Time  `bson:"updated_at"`
	DeletedAt  *time.Time `bson:"deleted_at"`
	TelegramID int64      `bson:"telegram_id"`
	Username   string     `bson:"username"`
	FirstName  string     `bson:"first_name"`
	WalletAddr string     `bson:"wallet_addr"`
	Balance    float64    `bson:"balance"`
	IsBlocked  bool       `bson:"is_blocked"`
}

func (d *ownerDoc) toModel() *models.Owner {
	o := &models.Owner{
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt, TelegramID: d.TelegramID,
		Username: d.Username, FirstName: d.FirstName, WalletAddr: d.WalletAddr,
		Balance: d.Balance, IsBlocked: d.IsBlocked,
	}
	o.ID, _ = uuid.Parse(d.ID)
	return o
}

func (s *Store) ownerToDoc(o *models.Owner) ownerDoc {
	return ownerDoc{
		ID: o.ID.String(), InstanceID: s.instanceID, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
		TelegramID: o.TelegramID, Username: o.Username, FirstName: o.FirstName,
		WalletAddr: o.WalletAddr, Balance: o.Balance, IsBlocked: o.IsBlocked,
	}
}

type lockDoc struct {
	ID           string            `bson:"_id"`
	InstanceID   string            `bson:"instance_id"`
	CreatedAt    time.Time         `bson:"created_at"`
	UpdatedAt    time.Time         `bson:"updated_at"`
	DeletedAt    *time.Time        `bson:"deleted_at"`
	OwnerID      string            `bson:"owner_id"`
	ChannelID    int64             `bson:"channel_id"`
	ChannelTitle string            `bson:"channel_title"`
	MaxMembers   int               `bson:"max_members"`
	CurrentCount int               `bson:"current_count"`
	DurationDay  int               `bson:"duration_day"`
	PricePerDay  float64           `bson:"price_per_day"`
	Status       models.LockStatus `bson:"status"`
	ExpiresAt    time.Time         `bson:"expires_at"`
}

func (d *lockDoc) toModel() *models.Lock {
	l := &models.Lock{
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
		ChannelID: d.ChannelID, ChannelTitle: d.ChannelTitle,
		MaxMembers: d.MaxMembers, CurrentCount: d.CurrentCount,
		DurationDay: d.DurationDay, PricePerDay: d.PricePerDay,
		Status: d.Status, ExpiresAt: d.ExpiresAt,
	}
	l.ID, _ = uuid.Parse(d.ID)
	l.OwnerID, _ = uuid.Parse(d.OwnerID)
	return l
}

func (s *Store) lockToDoc(l *models.Lock) lockDoc {
	return lockDoc{
		ID: l.ID.String(), InstanceID: s.instanceID, CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt,
		OwnerID: l.OwnerID.String(), ChannelID: l.ChannelID, ChannelTitle: l.ChannelTitle,
		MaxMembers: l.MaxMembers, CurrentCount: l.CurrentCount,
		DurationDay: l.DurationDay, PricePerDay: l.PricePerDay,
		Status: l.Status, ExpiresAt: l.ExpiresAt,
	}
}

type membershipDoc struct {
	BotID        string    `bson:"bot_id"`
	ChannelID    int64     `bson:"channel_id"`
	JoinedAt     time.Time `bson:"joined_at"`
	LastVerified time.Time `bson:"last_verified"`
}

type checkBotDoc struct {
	ID          string          `bson:"_id"`
	InstanceID  string          `bson:"instance_id"`
	CreatedAt   time.Time       `bson:"created_at"`
	UpdatedAt   time.Time       `bson:"updated_at"`
	DeletedAt   *time.Time      `bson:"deleted_at"`
	Token       string          `bson:"token"`
	Username    string          `bson:"username"`
	IsActive    bool            `bson:"is_active"`
	RateLimit   int             `bson:"rate_limit"`
	Memberships []membershipDoc `bson:"memberships"`
}

func (d *checkBotDoc) toModel() models.CheckBot {
	b := models.CheckBot{
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
		Token: d.Token, Username: d.Username, IsActive: d.IsActive, RateLimit: d.RateLimit,
	}
	b.ID, _ = uuid.Parse(d.ID)
	for _, m := range d.Memberships {
		bid, _ := uuid.Parse(m.BotID)
		b.Memberships = append(b.Memberships, models.BotChannelMembership{
			BotID: bid, ChannelID: m.ChannelID, JoinedAt: m.JoinedAt, LastVerified: m.LastVerified,
		})
	}
	return b
}

type paymentDoc struct {
	ID         string     `bson:"_id"`
	InstanceID string     `bson:"instance_id"`
	CreatedAt  time.Time  `bson:"created_at"`
	UpdatedAt  time.Time  `bson:"updated_at"`
	DeletedAt  *time.Time `bson:"deleted_at"`
	OwnerID    string     `bson:"owner_id"`
	LockID     string     `bson:"lock_id"`
	Amount     float64    `bson:"amount"`
	TxHash     string     `bson:"tx_hash"`
	Status     string     `bson:"status"`
}

// ---- Owner ----

func (s *Store) FindOwnerByID(ctx context.Context, id uuid.UUID) (*models.Owner, error) {
	var d ownerDoc
	err := s.db.Collection("owners").FindOne(ctx, s.scoped(bson.M{"_id": id.String()})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

func (s *Store) FindOwnerByTelegramID(ctx context.Context, id int64) (*models.Owner, error) {
	var d ownerDoc
	err := s.db.Collection("owners").FindOne(ctx, s.scoped(notDeleted(bson.M{"telegram_id": id}))).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

func (s *Store) CreateOwner(ctx context.Context, o *models.Owner) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	now := time.Now()
	o.CreatedAt, o.UpdatedAt = now, now
	_, err := s.db.Collection("owners").InsertOne(ctx, s.ownerToDoc(o))
	return err
}

func (s *Store) ListOwners(ctx context.Context) ([]models.Owner, error) {
	cur, err := s.db.Collection("owners").Find(ctx, s.scoped(notDeleted(nil)),
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []ownerDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	owners := make([]models.Owner, 0, len(docs))
	for _, d := range docs {
		owners = append(owners, *d.toModel())
	}
	return owners, nil
}

func (s *Store) UpdateBalance(ctx context.Context, ownerID uuid.UUID, amount float64) error {
	_, err := s.db.Collection("owners").UpdateOne(ctx,
		s.scoped(bson.M{"_id": ownerID.String()}),
		bson.M{"$inc": bson.M{"balance": amount}, "$set": bson.M{"updated_at": time.Now()}})
	return err
}

// ---- Lock ----

func (s *Store) FindLockByChannelID(ctx context.Context, channelID int64) (*models.Lock, error) {
	var d lockDoc
	err := s.db.Collection("locks").FindOne(ctx, s.scoped(notDeleted(bson.M{
		"channel_id": channelID, "status": models.LockActive,
	}))).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

func (s *Store) CreateLock(ctx context.Context, l *models.Lock) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	now := time.Now()
	l.CreatedAt, l.UpdatedAt = now, now
	_, err := s.db.Collection("locks").InsertOne(ctx, s.lockToDoc(l))
	return err
}

func (s *Store) FindLockByID(ctx context.Context, lockID uuid.UUID) (*models.Lock, error) {
	var d lockDoc
	err := s.db.Collection("locks").FindOne(ctx, s.scoped(bson.M{"_id": lockID.String()})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

func (s *Store) ExpireLock(ctx context.Context, lockID any) error {
	idStr, ok := lockID.(string)
	if !ok {
		if u, ok2 := lockID.(uuid.UUID); ok2 {
			idStr = u.String()
		}
	}
	_, err := s.db.Collection("locks").UpdateOne(ctx,
		s.scoped(bson.M{"_id": idStr}),
		bson.M{"$set": bson.M{"status": models.LockExpired, "updated_at": time.Now()}})
	return err
}

func (s *Store) FindExpiredLocks(ctx context.Context) ([]models.Lock, error) {
	cur, err := s.db.Collection("locks").Find(ctx, s.scoped(notDeleted(bson.M{
		"status": models.LockActive,
		"$or": bson.A{
			bson.M{"expires_at": bson.M{"$lt": time.Now()}},
			bson.M{"$and": bson.A{
				bson.M{"max_members": bson.M{"$gt": 0}},
				bson.M{"$expr": bson.M{"$gte": bson.A{"$current_count", "$max_members"}}},
			}},
		},
	})))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []lockDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	locks := make([]models.Lock, 0, len(docs))
	for _, d := range docs {
		locks = append(locks, *d.toModel())
	}
	return locks, nil
}

func (s *Store) FindLocksByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]models.Lock, error) {
	cur, err := s.db.Collection("locks").Find(ctx,
		s.scoped(notDeleted(bson.M{"owner_id": ownerID.String()})),
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []lockDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	locks := make([]models.Lock, 0, len(docs))
	for _, d := range docs {
		locks = append(locks, *d.toModel())
	}
	return locks, nil
}

func (s *Store) DeleteLock(ctx context.Context, lockID uuid.UUID) error {
	now := time.Now()
	_, err := s.db.Collection("locks").UpdateOne(ctx,
		s.scoped(bson.M{"_id": lockID.String()}),
		bson.M{"$set": bson.M{"deleted_at": now, "updated_at": now}})
	return err
}

func (s *Store) ListAllLocks(ctx context.Context) ([]models.Lock, error) {
	cur, err := s.db.Collection("locks").Find(ctx, s.scoped(notDeleted(nil)),
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []lockDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	locks := make([]models.Lock, 0, len(docs))
	for _, d := range docs {
		locks = append(locks, *d.toModel())
	}
	return locks, nil
}

// ---- CheckBot / membership ----

func (s *Store) FindActiveBots(ctx context.Context) ([]models.CheckBot, error) {
	cur, err := s.db.Collection("check_bots").Find(ctx, s.scoped(notDeleted(bson.M{"is_active": true})))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []checkBotDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	bots := make([]models.CheckBot, 0, len(docs))
	for _, d := range docs {
		bots = append(bots, d.toModel())
	}
	return bots, nil
}

func (s *Store) CreateCheckBot(ctx context.Context, b *models.CheckBot) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	now := time.Now()
	b.CreatedAt, b.UpdatedAt = now, now
	doc := checkBotDoc{
		ID: b.ID.String(), InstanceID: s.instanceID, CreatedAt: now, UpdatedAt: now,
		Token: b.Token, Username: b.Username, IsActive: b.IsActive, RateLimit: b.RateLimit,
		// Memberships باید [] باشد نه nil (که BSON آن را null می‌سازد) — $push
		// روی یک فیلدِ null (نه غایب، نه آرایه) رد می‌شود.
		Memberships: []membershipDoc{},
	}
	_, err := s.db.Collection("check_bots").InsertOne(ctx, doc)
	return err
}

// AddBotMembership یک عضویتِ (bot, channel) را idempotent اضافه/به‌روز
// می‌کند — چون Memberships درونِ خودِ سندِ CheckBot embed شده (نه کالکشنِ
// جدا)، ابتدا تلاش می‌شود عنصرِ موجود با positional operator به‌روز شود؛
// اگر عضویتی برای این channel نبود (matchedCount==0)، عنصر تازه push می‌شود.
// معادلِ رفتاریِ دقیقِ FirstOrCreate+Assign قبلی روی composite PK. فیلترِ
// اولیه با instance_id هم اسکوپ می‌شود تا botID یکسان بینِ دو instanceِ
// مختلفِ member-bot (که هرگز نباید رخ دهد چون botID از UUID خودِ instance
// می‌آید، ولی همچنان دفاعِ درست‌رفتاری همان الگوی بقیه‌ی متدهاست) تداخل نکند.
func (s *Store) AddBotMembership(ctx context.Context, m *models.BotChannelMembership) error {
	now := time.Now()
	res, err := s.db.Collection("check_bots").UpdateOne(ctx,
		s.scoped(bson.M{"_id": m.BotID.String(), "memberships.channel_id": m.ChannelID}),
		bson.M{"$set": bson.M{
			"memberships.$.joined_at":     m.JoinedAt,
			"memberships.$.last_verified": now,
			"updated_at":                  now,
		}})
	if err != nil {
		return err
	}
	if res.MatchedCount > 0 {
		return nil
	}
	_, err = s.db.Collection("check_bots").UpdateOne(ctx,
		s.scoped(bson.M{"_id": m.BotID.String()}),
		bson.M{
			"$push": bson.M{"memberships": membershipDoc{
				BotID: m.BotID.String(), ChannelID: m.ChannelID, JoinedAt: m.JoinedAt, LastVerified: now,
			}},
			"$set": bson.M{"updated_at": now},
		})
	return err
}

// ClearBotMemberships همه‌ی آرایه‌های memberships را روی بات‌های *همین
// instance* خالی می‌کند — قبلاً UpdateMany(bson.M{}, ...) بدون هیچ فیلتری
// بود، یعنی sync دوره‌ای یک instance می‌توانست memberships همه‌ی instanceهای
// دیگرِ member-bot را هم پاک کند (باگِ واقعیِ cross-tenant که با اضافه‌شدنِ
// instance_id به این migration رفع شد).
func (s *Store) ClearBotMemberships(ctx context.Context) error {
	_, err := s.db.Collection("check_bots").UpdateMany(ctx, s.scoped(nil),
		bson.M{"$set": bson.M{"memberships": bson.A{}, "updated_at": time.Now()}})
	return err
}

func (s *Store) DeactivateBotByID(ctx context.Context, botIDStr string) error {
	_, err := s.db.Collection("check_bots").UpdateOne(ctx,
		s.scoped(bson.M{"_id": botIDStr}),
		bson.M{"$set": bson.M{"is_active": false, "updated_at": time.Now()}})
	return err
}

// DeleteBotByID یک check-bot را از DB حذف می‌کند (حذفِ سخت، مثل نسخه‌ی قبلی
// که با .Delete مستقیم — نه Base soft-delete — عمل می‌کرد چون این کوئری
// روی id خام بدون هیچ فیلترِ اضافه‌ای بود).
func (s *Store) DeleteBotByID(ctx context.Context, botIDStr string) error {
	_, err := s.db.Collection("check_bots").DeleteOne(ctx, s.scoped(bson.M{"_id": botIDStr}))
	return err
}

// ---- Payment (orphan flow — رجوع کامنتِ models.Payment) ----

func (s *Store) ApprovePayment(ctx context.Context, payID uuid.UUID) error {
	_, err := s.db.Collection("payments").UpdateOne(ctx,
		s.scoped(bson.M{"_id": payID.String()}),
		bson.M{"$set": bson.M{"status": "confirmed", "updated_at": time.Now()}})
	return err
}

func (s *Store) CreatePayment(ctx context.Context, p *models.Payment) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	p.CreatedAt, p.UpdatedAt = now, now
	if p.Status == "" {
		p.Status = "pending"
	}
	doc := paymentDoc{
		ID: p.ID.String(), InstanceID: s.instanceID, CreatedAt: now, UpdatedAt: now,
		OwnerID: p.OwnerID.String(), LockID: p.LockID.String(),
		Amount: p.Amount, TxHash: p.TxHash, Status: p.Status,
	}
	_, err := s.db.Collection("payments").InsertOne(ctx, doc)
	return err
}

func (s *Store) FindPendingPayments(ctx context.Context) ([]models.Payment, error) {
	cur, err := s.db.Collection("payments").Find(ctx, s.scoped(bson.M{"status": "pending"}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []paymentDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	payments := make([]models.Payment, 0, len(docs))
	for _, d := range docs {
		p := models.Payment{
			CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
			Amount: d.Amount, TxHash: d.TxHash, Status: d.Status,
		}
		p.ID, _ = uuid.Parse(d.ID)
		p.OwnerID, _ = uuid.Parse(d.OwnerID)
		p.LockID, _ = uuid.Parse(d.LockID)
		payments = append(payments, p)
	}
	return payments, nil
}
