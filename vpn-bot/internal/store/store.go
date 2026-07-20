// Package store لایه‌ی دسترسی داده‌ی vpn-bot روی MongoDB (بدون Postgres).
//
// این بازنویسی از نسخه‌ی قبلیِ GORM/Postgres است؛ نام و امضای متدها عمداً
// یکی نگه داشته شده تا کد لایه‌ی tgbot/scheduler بدون تغییر کار کند. هر
// documentِ Mongo یک ساختار خصوصی («*doc») دارد که فقط داخل همین فایل دیده
// می‌شود؛ مدل‌های عمومی (models.User و...) هیچ وابستگی‌ای به driver ندارند.
//
// چندمستأجری: همه‌ی instanceهای vpn-bot یک دیتابیسِ Mongo مشترک دارند
// (MONGO_DB=vpn_bot، برای بهترین عملکرد به‌جای دیتابیسِ جدا به‌ازای هر
// instance) — جداسازیِ دیتای هر instance با فیلدِ instance_id روی هر سند
// انجام می‌شود، دقیقاً همان الگویی که uploader-bot با shared-core/docstore
// پیاده می‌کند (رجوع Base.baseFilter آنجا). instanceID یک‌بار در startup از
// BOT_TOKEN مشتق و به New پاس داده می‌شود؛ همه‌ی متدهای این فایل باید از
// scoped() برای فیلتر/ثبتِ instance_id استفاده کنند.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mrjvadi/creatorbot/vpn-bot/internal/models"
)

type Store struct {
	db         *mongo.Database
	instanceID string
}

func New(db *mongo.Database, instanceID string) *Store {
	return &Store{db: db, instanceID: instanceID}
}

// scoped فیلترِ instance_id را به filter اضافه می‌کند — هر خواندن/نوشتن در
// این فایل باید از این (یا notDeleted ترکیب‌شده با این) عبور کند، وگرنه
// instanceهای دیگرِ vpn-bot که همین دیتابیس را شریک‌اند دیتای هم را
// می‌بینند/رد می‌کنند.
func (s *Store) scoped(extra bson.M) bson.M {
	if extra == nil {
		extra = bson.M{}
	}
	extra["instance_id"] = s.instanceID
	return extra
}

// notDeleted فیلتر استانداردِ حذف‌نرم: هم سندهایی که فیلد deleted_at ندارند
// (هرگز حذف نشده‌اند) و هم آن‌هایی که صراحتاً null هستند را می‌پذیرد — رفتار
// معادل GORM که خودکار deleted_at IS NULL اضافه می‌کرد.
func notDeleted(extra bson.M) bson.M {
	if extra == nil {
		extra = bson.M{}
	}
	extra["deleted_at"] = nil
	return extra
}

// EnsureIndexes ایندکس‌های لازم را idempotent می‌سازد — باید یک‌بار در startup
// صدا زده شود (معادل AutoMigrate + CREATE UNIQUE INDEX قبلی). instance_id
// کلیدِ پیشروی هر ایندکسِ یکتاست تا instanceهای مختلف روی این دیتابیسِ
// مشترک دیتای هم را رد نکنند.
func (s *Store) EnsureIndexes(ctx context.Context) error {
	models := []struct {
		coll  string
		index mongo.IndexModel
	}{
		{"users", mongo.IndexModel{
			Keys:    bson.D{{Key: "instance_id", Value: 1}, {Key: "telegram_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"discount_codes", mongo.IndexModel{
			Keys:    bson.D{{Key: "instance_id", Value: 1}, {Key: "code", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"subscriptions", mongo.IndexModel{
			Keys: bson.D{{Key: "instance_id", Value: 1}, {Key: "user_id", Value: 1}},
		}},
		// یکتاییِ پرداخت‌های آنلاین — معادل partial unique index قبلیِ Postgres:
		// از فعال‌سازی چندباره‌ی اشتراک با کلیک تکراری روی «پرداخت کردم» جلوگیری می‌کند.
		{"payments", mongo.IndexModel{
			Keys: bson.D{
				{Key: "instance_id", Value: 1},
				{Key: "gateway", Value: 1},
				{Key: "ref_code", Value: 1},
			},
			// نکته: partial index در MongoDB فقط زیرمجموعه‌ای محدود از عملگرها را
			// می‌پذیرد (eq/gt/gte/lt/lte/exists/type) — $ne پشتیبانی نمی‌شود. برای
			// «ref_code غیرخالی» به‌جای $ne از $gt استفاده می‌شود؛ چون رشته‌ی خالی
			// از نظر ترتیب لغوی کوچک‌ترینِ رشته‌هاست، gt:"" دقیقاً معادل «غیرخالی»ست.
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.D{
				{Key: "gateway", Value: bson.D{{Key: "$in", Value: bson.A{"zarinpal", "nowpayments"}}}},
				{Key: "ref_code", Value: bson.D{{Key: "$gt", Value: ""}}},
			}),
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
	ID         string     `bson:"_id"`
	InstanceID string     `bson:"instance_id"`
	CreatedAt  time.Time  `bson:"created_at"`
	UpdatedAt  time.Time  `bson:"updated_at"`
	DeletedAt  *time.Time `bson:"deleted_at"`
	TelegramID int64      `bson:"telegram_id"`
	Username   string     `bson:"username"`
	FirstName  string     `bson:"first_name"`
	Balance    float64    `bson:"balance"`
	IsBlocked  bool       `bson:"is_blocked"`
	ResellerID *string    `bson:"reseller_id"`
	Discount   float64    `bson:"discount"`
}

func (d *userDoc) toModel() *models.User {
	u := &models.User{
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
		TelegramID: d.TelegramID,
		Username:   d.Username,
		FirstName:  d.FirstName,
		Balance:    d.Balance,
		IsBlocked:  d.IsBlocked,
		Discount:   d.Discount,
	}
	u.ID, _ = uuid.Parse(d.ID)
	if d.ResellerID != nil {
		if rid, err := uuid.Parse(*d.ResellerID); err == nil {
			u.ResellerID = &rid
		}
	}
	return u
}

type panelDoc struct {
	ID          string     `bson:"_id"`
	InstanceID  string     `bson:"instance_id"`
	CreatedAt   time.Time  `bson:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at"`
	DeletedAt   *time.Time `bson:"deleted_at"`
	Name        string     `bson:"name"`
	Type        string     `bson:"type"`
	BaseURL     string     `bson:"base_url"`
	Username    string     `bson:"username"`
	Password    string     `bson:"password"`
	Capacity    int        `bson:"capacity"`
	ActiveCount int        `bson:"active_count"`
	IsActive    bool       `bson:"is_active"`
}

func (d *panelDoc) toModel() *models.Panel {
	p := &models.Panel{
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
		Name: d.Name, Type: d.Type, BaseURL: d.BaseURL,
		Username: d.Username, Password: d.Password,
		Capacity: d.Capacity, ActiveCount: d.ActiveCount, IsActive: d.IsActive,
	}
	p.ID, _ = uuid.Parse(d.ID)
	return p
}

func (s *Store) panelToDoc(p *models.Panel) panelDoc {
	return panelDoc{
		ID: p.ID.String(), InstanceID: s.instanceID, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		Name: p.Name, Type: p.Type, BaseURL: p.BaseURL,
		Username: p.Username, Password: p.Password,
		Capacity: p.Capacity, ActiveCount: p.ActiveCount, IsActive: p.IsActive,
	}
}

type planDoc struct {
	ID          string     `bson:"_id"`
	InstanceID  string     `bson:"instance_id"`
	CreatedAt   time.Time  `bson:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at"`
	DeletedAt   *time.Time `bson:"deleted_at"`
	Name        string     `bson:"name"`
	DurationDay int        `bson:"duration_day"`
	DataGB      float64    `bson:"data_gb"`
	Price       float64    `bson:"price"`
	IsActive    bool       `bson:"is_active"`
}

func (d *planDoc) toModel() *models.Plan {
	p := &models.Plan{
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt, Name: d.Name,
		DurationDay: d.DurationDay, DataGB: d.DataGB, Price: d.Price, IsActive: d.IsActive,
	}
	p.ID, _ = uuid.Parse(d.ID)
	return p
}

type subscriptionDoc struct {
	ID         string     `bson:"_id"`
	InstanceID string     `bson:"instance_id"`
	CreatedAt  time.Time  `bson:"created_at"`
	UpdatedAt  time.Time  `bson:"updated_at"`
	DeletedAt  *time.Time `bson:"deleted_at"`
	UserID     string     `bson:"user_id"`
	// UserTelegramID دنرمالایزشده — تنها فیلد User که در تاریخچه‌ی این پروژه
	// از طریق Preload("User") خوانده می‌شد (رجوع scheduler.go). به‌جای
	// $lookup، مستقیم روی خودِ سند نگه داشته می‌شود.
	UserTelegramID int64                     `bson:"user_telegram_id"`
	PanelID        string                    `bson:"panel_id"`
	PlanID         string                    `bson:"plan_id"`
	Username       string                    `bson:"username"`
	Status         models.SubscriptionStatus `bson:"status"`
	ExpiresAt      time.Time                 `bson:"expires_at"`
	DataLimit      float64                   `bson:"data_limit"`
	UsedData       float64                   `bson:"used_data"`
}

func (d *subscriptionDoc) toModel() models.Subscription {
	s := models.Subscription{
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
		Username: d.Username, Status: d.Status, ExpiresAt: d.ExpiresAt,
		DataLimit: d.DataLimit, UsedData: d.UsedData,
	}
	s.ID, _ = uuid.Parse(d.ID)
	s.UserID, _ = uuid.Parse(d.UserID)
	s.PanelID, _ = uuid.Parse(d.PanelID)
	s.PlanID, _ = uuid.Parse(d.PlanID)
	return s
}

type discountCodeDoc struct {
	ID         string     `bson:"_id"`
	InstanceID string     `bson:"instance_id"`
	CreatedAt  time.Time  `bson:"created_at"`
	UpdatedAt  time.Time  `bson:"updated_at"`
	DeletedAt  *time.Time `bson:"deleted_at"`
	Code       string     `bson:"code"`
	Percent    float64    `bson:"percent"`
	MaxUse     int        `bson:"max_use"`
	UsedCount  int        `bson:"used_count"`
	IsActive   bool       `bson:"is_active"`
}

func (d *discountCodeDoc) toModel() *models.DiscountCode {
	dc := &models.DiscountCode{
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt, Code: d.Code,
		Percent: d.Percent, MaxUse: d.MaxUse, UsedCount: d.UsedCount, IsActive: d.IsActive,
	}
	dc.ID, _ = uuid.Parse(d.ID)
	return dc
}

type paymentDoc struct {
	ID         string     `bson:"_id"`
	InstanceID string     `bson:"instance_id"`
	CreatedAt  time.Time  `bson:"created_at"`
	UpdatedAt  time.Time  `bson:"updated_at"`
	DeletedAt  *time.Time `bson:"deleted_at"`
	UserID     string     `bson:"user_id"`
	Amount     float64    `bson:"amount"`
	Gateway    string     `bson:"gateway"`
	Status     string     `bson:"status"`
	RefCode    string     `bson:"ref_code"`
	Receipt    string     `bson:"receipt"`
	PlanID     *string    `bson:"plan_id"`
}

func (d *paymentDoc) toModel() *models.Payment {
	p := &models.Payment{
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
		Amount: d.Amount, Gateway: d.Gateway, Status: d.Status,
		RefCode: d.RefCode, Receipt: d.Receipt,
	}
	p.ID, _ = uuid.Parse(d.ID)
	p.UserID, _ = uuid.Parse(d.UserID)
	if d.PlanID != nil {
		if pid, err := uuid.Parse(*d.PlanID); err == nil {
			p.PlanID = &pid
		}
	}
	return p
}

func (s *Store) paymentToDoc(p *models.Payment) paymentDoc {
	d := paymentDoc{
		ID: p.ID.String(), InstanceID: s.instanceID, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		UserID: p.UserID.String(), Amount: p.Amount, Gateway: p.Gateway,
		Status: p.Status, RefCode: p.RefCode, Receipt: p.Receipt,
	}
	if p.PlanID != nil {
		pid := p.PlanID.String()
		d.PlanID = &pid
	}
	return d
}

// ---- User ----

func (s *Store) FindUserByTelegramID(ctx context.Context, id int64) (*models.User, error) {
	var d userDoc
	err := s.db.Collection("users").FindOne(ctx, s.scoped(notDeleted(bson.M{"telegram_id": id}))).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

// UpsertUser کاربر را با (instance_id,TelegramID) پیدا یا می‌سازد و *u را با
// سندِ کاملِ نهایی (شامل ID/Balance/IsBlocked واقعی) پر می‌کند — دقیقاً همان
// رفتاری که GORM FirstOrCreate قبلاً می‌داد (فراخوان u را بدون خواندنِ دوباره
// برمی‌گرداند). فقط username/first_name همیشه به‌روزرسانی می‌شوند؛ سایر
// فیلدها (balance و ...) فقط در insert اول با پیش‌فرض ست می‌شوند تا هیچ‌وقت
// overwrite نشوند.
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
			"balance":     0.0,
			"is_blocked":  false,
			"discount":    0.0,
			"deleted_at":  nil,
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

func (s *Store) UpdateBalance(ctx context.Context, userID uuid.UUID, delta float64) error {
	_, err := s.db.Collection("users").UpdateOne(ctx,
		s.scoped(bson.M{"_id": userID.String()}),
		bson.M{"$inc": bson.M{"balance": delta}, "$set": bson.M{"updated_at": time.Now()}})
	return err
}

// DeductBalanceIfEnough به‌صورت اتمیک amount را کسر می‌کند فقط اگر موجودی
// کافی باشد. شرط balance>=amount داخل همان findOneAndUpdate است — دو درخواست
// هم‌زمان نمی‌توانند هر دو از یک موجودی کسر کنند (double-spend).
func (s *Store) DeductBalanceIfEnough(ctx context.Context, userID uuid.UUID, amount float64) (bool, error) {
	filter := s.scoped(bson.M{"_id": userID.String(), "balance": bson.M{"$gte": amount}})
	update := bson.M{"$inc": bson.M{"balance": -amount}, "$set": bson.M{"updated_at": time.Now()}}
	err := s.db.Collection("users").FindOneAndUpdate(ctx, filter, update).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	cur, err := s.db.Collection("users").Find(ctx, s.scoped(notDeleted(nil)),
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []userDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	users := make([]models.User, 0, len(docs))
	for _, d := range docs {
		users = append(users, *d.toModel())
	}
	return users, nil
}

func (s *Store) FindUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var d userDoc
	err := s.db.Collection("users").FindOne(ctx, s.scoped(bson.M{"_id": id.String()})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

// ---- Plan ----

func (s *Store) ListPlans(ctx context.Context) ([]models.Plan, error) {
	cur, err := s.db.Collection("plans").Find(ctx, s.scoped(notDeleted(bson.M{"is_active": true})))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []planDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	plans := make([]models.Plan, 0, len(docs))
	for _, d := range docs {
		plans = append(plans, *d.toModel())
	}
	return plans, nil
}

func (s *Store) FindPlan(ctx context.Context, id uuid.UUID) (*models.Plan, error) {
	var d planDoc
	err := s.db.Collection("plans").FindOne(ctx, s.scoped(bson.M{"_id": id.String()})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

// ---- Subscription ----

// SubscriptionWithUser یک اشتراک به‌همراه TelegramID مالکش — همان شکلِ قبلی
// (denormalized به‌جای $lookup، رجوع subscriptionDoc.UserTelegramID).
type SubscriptionWithUser struct {
	models.Subscription
	User models.User
}

func (s *Store) CreateSubscription(ctx context.Context, sub *models.Subscription) error {
	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	now := time.Now()
	sub.CreatedAt, sub.UpdatedAt = now, now

	// UserTelegramID برای denormalization لازم است؛ کاربر را می‌خوانیم.
	var userTelegramID int64
	if u, err := s.FindUserByID(ctx, sub.UserID); err == nil && u != nil {
		userTelegramID = u.TelegramID
	}

	doc := subscriptionDoc{
		ID: sub.ID.String(), InstanceID: s.instanceID, CreatedAt: sub.CreatedAt, UpdatedAt: sub.UpdatedAt,
		UserID: sub.UserID.String(), UserTelegramID: userTelegramID,
		PanelID: sub.PanelID.String(), PlanID: sub.PlanID.String(),
		Username: sub.Username, Status: sub.Status, ExpiresAt: sub.ExpiresAt,
		DataLimit: sub.DataLimit, UsedData: sub.UsedData,
	}
	_, err := s.db.Collection("subscriptions").InsertOne(ctx, doc)
	return err
}

func (s *Store) FindActiveSubscriptions(ctx context.Context) ([]models.Subscription, error) {
	return s.findSubscriptions(ctx, s.scoped(notDeleted(bson.M{"status": models.SubActive})))
}

func (s *Store) FindExpiredSubscriptions(ctx context.Context) ([]models.Subscription, error) {
	return s.findSubscriptions(ctx, s.scoped(notDeleted(bson.M{
		"status": models.SubActive, "expires_at": bson.M{"$lt": time.Now()},
	})))
}

func (s *Store) findSubscriptions(ctx context.Context, filter bson.M) ([]models.Subscription, error) {
	cur, err := s.db.Collection("subscriptions").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []subscriptionDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	subs := make([]models.Subscription, 0, len(docs))
	for _, d := range docs {
		subs = append(subs, d.toModel())
	}
	return subs, nil
}

func (s *Store) FindSubscriptionsExpiringIn(ctx context.Context, d time.Duration) ([]SubscriptionWithUser, error) {
	deadline := time.Now().Add(d)
	cur, err := s.db.Collection("subscriptions").Find(ctx, s.scoped(notDeleted(bson.M{
		"status":     models.SubActive,
		"expires_at": bson.M{"$gt": time.Now(), "$lt": deadline},
	})))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []subscriptionDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	results := make([]SubscriptionWithUser, 0, len(docs))
	for _, sd := range docs {
		results = append(results, SubscriptionWithUser{
			Subscription: sd.toModel(),
			User:         models.User{TelegramID: sd.UserTelegramID},
		})
	}
	return results, nil
}

func (s *Store) UpdateSubscriptionStatus(ctx context.Context, id uuid.UUID, status models.SubscriptionStatus) error {
	_, err := s.db.Collection("subscriptions").UpdateOne(ctx,
		s.scoped(bson.M{"_id": id.String()}),
		bson.M{"$set": bson.M{"status": status, "updated_at": time.Now()}})
	return err
}

func (s *Store) UpdateSubscriptionUsage(ctx context.Context, id uuid.UUID, usedBytes int64) error {
	_, err := s.db.Collection("subscriptions").UpdateOne(ctx,
		s.scoped(bson.M{"_id": id.String()}),
		bson.M{"$set": bson.M{
			"used_data":  float64(usedBytes) / 1024 / 1024 / 1024,
			"updated_at": time.Now(),
		}})
	return err
}

func (s *Store) FindSubscriptionsByUserID(ctx context.Context, userID uuid.UUID) ([]models.Subscription, error) {
	cur, err := s.db.Collection("subscriptions").Find(ctx,
		s.scoped(notDeleted(bson.M{"user_id": userID.String()})),
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []subscriptionDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	subs := make([]models.Subscription, 0, len(docs))
	for _, d := range docs {
		subs = append(subs, d.toModel())
	}
	return subs, nil
}

func (s *Store) FindSubscriptionByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	var d subscriptionDoc
	err := s.db.Collection("subscriptions").FindOne(ctx, s.scoped(bson.M{"_id": id.String()})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m := d.toModel()
	return &m, nil
}

// ---- Payment ----

func (s *Store) CreatePayment(ctx context.Context, p *models.Payment) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	p.CreatedAt, p.UpdatedAt = now, now
	if p.Status == "" {
		p.Status = "pending"
	}
	_, err := s.db.Collection("payments").InsertOne(ctx, s.paymentToDoc(p))
	return err
}

// ClaimOnlinePayment یک پرداخت آنلاین را به‌صورت اتمیک بر اساس
// (instance_id, gateway, ref_code) «claim» می‌کند. اگر قبلاً برای همین
// refCode رکوردی ثبت شده باشد (کلیک تکراری روی «پرداخت کردم»)، خطای
// duplicate-key مانع insert می‌شود و claimed=false برمی‌گردد — یکتایی توسط
// ایندکس partial تضمین می‌شود (رجوع EnsureIndexes).
func (s *Store) ClaimOnlinePayment(ctx context.Context, p *models.Payment) (bool, error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	p.CreatedAt, p.UpdatedAt = now, now
	if p.Status == "" {
		p.Status = "pending"
	}
	_, err := s.db.Collection("payments").InsertOne(ctx, s.paymentToDoc(p))
	if mongo.IsDuplicateKeyError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) FindPaymentByID(ctx context.Context, id uuid.UUID) (*models.Payment, error) {
	var d paymentDoc
	err := s.db.Collection("payments").FindOne(ctx, s.scoped(bson.M{"_id": id.String()})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

func (s *Store) FindPendingPayments(ctx context.Context) ([]models.Payment, error) {
	cur, err := s.db.Collection("payments").Find(ctx,
		s.scoped(bson.M{"status": "pending"}),
		options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
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
		payments = append(payments, *d.toModel())
	}
	return payments, nil
}

func (s *Store) UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := s.db.Collection("payments").UpdateOne(ctx,
		s.scoped(bson.M{"_id": id.String()}),
		bson.M{"$set": bson.M{"status": status, "updated_at": time.Now()}})
	return err
}

// ClaimPendingPayment پرداخت را فقط اگر هنوز pending باشد به status جدید
// می‌برد (findOneAndUpdate اتمیک) و سند به‌روزشده را برمی‌گرداند. matched=false
// یعنی پرداخت قبلاً پردازش شده بود (رد شود، دوباره اعتبار داده نشود) — این
// رفع باگ واقعی دوبار-اعتبار (duplicate credit) از دو کلیک هم‌زمان روی
// «تأیید» است که در نسخه‌ی Postgres وجود داشت (چک-سپس-نوشتنِ غیراتمیک).
func (s *Store) ClaimPendingPayment(ctx context.Context, id uuid.UUID, newStatus string) (*models.Payment, error) {
	filter := s.scoped(bson.M{"_id": id.String(), "status": "pending"})
	update := bson.M{"$set": bson.M{"status": newStatus, "updated_at": time.Now()}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var d paymentDoc
	err := s.db.Collection("payments").FindOneAndUpdate(ctx, filter, update, opts).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

// ---- Panel ----

func (s *Store) ListPanels(ctx context.Context) ([]models.Panel, error) {
	cur, err := s.db.Collection("panels").Find(ctx, s.scoped(notDeleted(nil)),
		options.Find().SetSort(bson.D{{Key: "active_count", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []panelDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	panels := make([]models.Panel, 0, len(docs))
	for _, d := range docs {
		panels = append(panels, *d.toModel())
	}
	return panels, nil
}

func (s *Store) CreatePanel(ctx context.Context, p *models.Panel) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	p.CreatedAt, p.UpdatedAt = now, now
	_, err := s.db.Collection("panels").InsertOne(ctx, s.panelToDoc(p))
	return err
}

func (s *Store) FindPanelByID(ctx context.Context, id uuid.UUID) (*models.Panel, error) {
	var d panelDoc
	err := s.db.Collection("panels").FindOne(ctx, s.scoped(bson.M{"_id": id.String()})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

// FindBestPanel کم‌ترین active_count را دارد (load balance).
func (s *Store) FindBestPanel(ctx context.Context) (*models.Panel, error) {
	filter := s.scoped(notDeleted(bson.M{
		"is_active": true,
		"$or": bson.A{
			bson.M{"capacity": 0},
			bson.M{"$expr": bson.M{"$lt": bson.A{"$active_count", "$capacity"}}},
		},
	}))
	opts := options.FindOne().SetSort(bson.D{{Key: "active_count", Value: 1}})
	var d panelDoc
	err := s.db.Collection("panels").FindOne(ctx, filter, opts).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

func (s *Store) UpdatePanel(ctx context.Context, p *models.Panel) error {
	p.UpdatedAt = time.Now()
	_, err := s.db.Collection("panels").ReplaceOne(ctx, s.scoped(bson.M{"_id": p.ID.String()}), s.panelToDoc(p))
	return err
}

func (s *Store) DeletePanel(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := s.db.Collection("panels").UpdateOne(ctx,
		s.scoped(bson.M{"_id": id.String()}),
		bson.M{"$set": bson.M{"deleted_at": now, "updated_at": now}})
	return err
}

func (s *Store) IncrementPanelCount(ctx context.Context, panelID uuid.UUID, delta int) error {
	_, err := s.db.Collection("panels").UpdateOne(ctx,
		s.scoped(bson.M{"_id": panelID.String()}),
		bson.M{"$inc": bson.M{"active_count": delta}, "$set": bson.M{"updated_at": time.Now()}})
	return err
}

// ---- DiscountCode (تعریف‌شده، فعلاً بلااستفاده در مسیر خرید — مثل قبل) ----

func (s *Store) CreateDiscountCode(ctx context.Context, code *models.DiscountCode) error {
	if code.ID == uuid.Nil {
		code.ID = uuid.New()
	}
	now := time.Now()
	code.CreatedAt, code.UpdatedAt = now, now
	doc := discountCodeDoc{
		ID: code.ID.String(), InstanceID: s.instanceID, CreatedAt: now, UpdatedAt: now,
		Code: code.Code, Percent: code.Percent, MaxUse: code.MaxUse,
		UsedCount: code.UsedCount, IsActive: code.IsActive,
	}
	_, err := s.db.Collection("discount_codes").InsertOne(ctx, doc)
	return err
}

func (s *Store) FindDiscountCode(ctx context.Context, code string) (*models.DiscountCode, error) {
	var d discountCodeDoc
	err := s.db.Collection("discount_codes").FindOne(ctx,
		s.scoped(bson.M{"code": code, "is_active": true})).Decode(&d)
	if err != nil {
		return nil, err
	}
	return d.toModel(), nil
}

// UseDiscountCode شمارنده‌ی مصرف را فقط اگر هنوز به سقف نرسیده باشد اتمیک
// افزایش می‌دهد — برخلاف نسخه‌ی Postgresِ قبلی (که هیچ شرطی نداشت)، اینجا از
// ابتدا صحیح طراحی شده چون این مسیر فعلاً بلااستفاده است و بهترین زمان برای
// رفعِ نقصِ over-redemption همین migration است.
func (s *Store) UseDiscountCode(ctx context.Context, id any) error {
	idStr, ok := id.(string)
	if !ok {
		if u, ok2 := id.(uuid.UUID); ok2 {
			idStr = u.String()
		}
	}
	filter := s.scoped(bson.M{"_id": idStr, "$expr": bson.M{"$lt": bson.A{"$used_count", "$max_use"}}})
	_, err := s.db.Collection("discount_codes").UpdateOne(ctx, filter,
		bson.M{"$inc": bson.M{"used_count": 1}, "$set": bson.M{"updated_at": time.Now()}})
	return err
}
