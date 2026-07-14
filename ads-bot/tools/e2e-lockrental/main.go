// e2e-lockrental — تست end-to-end مدل اقتصادی «اجاره‌ی قفل کانال»، بدون تلگرام.
//
// چون ساخت/تأیید کمپین فقط از طریق wizard تلگرامی ads-bot ممکن است (مسیر NATS
// ندارد)، این ابزار مثل tools/e2e-provision عمل می‌کند ولی ترکیبی است: مستقیم با
// همان متدهای واقعیِ ads-bot/internal/store روی دیتابیس adsbot کار می‌کند و برای
// پول از botpay واقعی (NATS + HMAC) استفاده می‌کند. کل چرخه‌ی حیات را مرحله‌به‌مرحله
// با PASS/FAIL می‌سنجد:
//
//	۱. seed: چند FreeBotSlot آزاد + یک LockRentalCampaign (pending_review)
//	۲. approve: pay.deduct بودجه از خریدار → active → اتصال slot ها
//	۳. join کاربر: TryRecordJoinReward (رزرو با تأخیر) + idempotency دو-باره
//	۴. fraud: ReversePendingRewardByUser → reward = reversed، بودجه برمی‌گردد
//	۵. settlement: settle_at را به گذشته می‌بریم → pay.credit به کاربر → settled
//	۶. پایان کمپین: end_at گذشته → MarkRentalDoneIfFinished → done + آزادسازی slot
//
// چون این ابزار خودش store-level منطق handler/scheduler را بازتولید می‌کند، به
// اجرای خودِ ads-bot (که به تلگرام واقعی وصل می‌شود) نیاز ندارد — فقط NATS، botpay
// و Postgres adsbot لازم‌اند. با فلگ -emit-nats رویدادها روی core NATS هم منتشر
// می‌شوند تا اگر یک نمونه‌ی ads-bot در حال اجرا باشد، آن هم exercise شود (به‌خاطر
// idempotency، پردازش دوباره بی‌خطر است).
//
// اجرا (از پوشه‌ی ads-bot/tools/e2e-lockrental):
//
//	go run . -hmac <SERVICE_HMAC_SECRET> -dsn 'postgres://botuser:...@127.0.0.1:5434/adsbot?sslmode=disable'
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/ads-bot/internal/store"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
)

const membershipJoinedSubject = "membership.joined"

func main() {
	natsURL := flag.String("nats", "nats://localhost:4222", "آدرس NATS")
	natsUser := flag.String("nats-user", "nats", "")
	natsPass := flag.String("nats-pass", "nats_secret", "")
	hmacSecret := flag.String("hmac", os.Getenv("SERVICE_HMAC_SECRET"), "SERVICE_HMAC_SECRET (همان مقدار ads-bot/botpay)")
	dsn := flag.String("dsn", os.Getenv("POSTGRES_DSN"), "DSN دیتابیس adsbot (پیش‌فرض از POSTGRES_DSN)")
	buyer := flag.Int64("buyer", 950000001, "TelegramID خریدار اجاره")
	member1 := flag.Int64("member1", 950000101, "کاربر عضوشونده‌ی اول (مسیر settlement)")
	member2 := flag.Int64("member2", 950000102, "کاربر عضوشونده‌ی دوم (مسیر fraud/reversal)")
	channel := flag.Int64("channel", -1009500000001, "TargetChannelID کانال هدف کمپین")
	reviewer := flag.Int64("reviewer", 7631742375, "OWNER_ID پلتفرم (تأییدکننده)")
	budget := flag.Float64("budget", 1.0, "بودجه‌ی کمپین (TON)")
	reward := flag.Float64("reward", 0.1, "پاداش هر join (TON)")
	emitNATS := flag.Bool("emit-nats", false, "علاوه بر مسیر مستقیم، membership.joined/fraud.detected را روی core NATS هم منتشر کن")
	cleanup := flag.Bool("cleanup", true, "داده‌های تست را در پایان پاک کن")
	flag.Parse()

	if *hmacSecret == "" || *dsn == "" {
		fmt.Fprintln(os.Stderr, "فلگ‌های -hmac و -dsn (یا env معادل) لازم‌اند")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// ── اتصال‌ها ─────────────────────────────────────────────────
	db, err := gorm.Open(postgres.Open(*dsn))
	if err != nil {
		fatal("db connect", err)
	}
	if err := store.AutoMigrate(db); err != nil {
		fatal("migrate", err)
	}
	st := store.New(db)
	pass("db connect", "adsbot + AutoMigrate")

	nc, err := natsclient.New(natsclient.Config{
		URL: *natsURL, Username: *natsUser, Password: *natsPass, Name: "e2e-lockrental",
	})
	if err != nil {
		fatal("NATS connect", err)
	}
	defer nc.Close()
	pass("NATS connect", *natsURL)

	// pay client با هویت واقعی ads-bot (همان چیزی که خودِ سرویس می‌فرستد)
	pay := natspayclient.New(nc, nil, natspayclient.Config{
		ServiceID:  "ads-bot",
		ServiceKey: auth.ComputeServiceKey(*hmacSecret, "ads-bot"),
		Timeout:    5 * time.Second,
	})

	// ── ۱. seed: FreeBotSlot های آزاد ────────────────────────────
	// BotID های تست را نشانه‌دار (بازه‌ی بالا) می‌گیریم تا با داده‌ی واقعی قاطی نشوند.
	testBotIDs := []int64{9_500_000_001, 9_500_000_002, 9_500_000_003}
	testSlotIDs := make([]uuid.UUID, 0, len(testBotIDs))
	for _, bid := range testBotIDs {
		slot := &store.FreeBotSlot{ID: uuid.New(), BotInstanceID: uuid.New(), BotID: bid}
		if err := st.UpsertFreeBotSlot(ctx, slot); err != nil {
			fatal("seed slot", err)
		}
		if got, _ := st.FindFreeBotSlotByBotID(ctx, bid); got != nil {
			testSlotIDs = append(testSlotIDs, got.ID)
		}
	}
	pass("seed slots", fmt.Sprintf("%d اسلات آزاد", len(testSlotIDs)))

	// ── ۲. ساخت کمپین (pending_review) ───────────────────────────
	camp := &store.LockRentalCampaign{
		ID:                        uuid.New(),
		BuyerTelegramID:           *buyer,
		TargetChannelID:           *channel,
		Status:                    store.RentalPendingReview,
		RewardPerJoinTON:          *reward,
		Budget:                    *budget,
		FreeBotOwnerRewardPercent: 5,
	}
	if err := st.CreateLockRental(ctx, camp); err != nil {
		fatal("create campaign", err)
	}
	pass("create campaign", fmt.Sprintf("id=%s budget=%.2f reward=%.2f", camp.ID, *budget, *reward))

	if *cleanup {
		defer cleanupTest(db, camp.ID, testBotIDs, *member1, *member2)
	}

	// ── ۳. شارژ خریدار و تأیید (کسر بودجه) ───────────────────────
	if err := pay.Credit(ctx, *buyer, *budget+0.5, "e2e-lockrental-credit", `{"src":"e2e-lockrental"}`); err != nil {
		fatal("pay.credit buyer", err)
	}
	bal0, err := pay.Balance(ctx, *buyer)
	if err != nil {
		fatal("pay.balance buyer", err)
	}
	pass("credit buyer", fmt.Sprintf("total=%.2f TON", bal0.Total))

	// approveLockRental را بازتولید می‌کنیم: deduct بودجه → active → اتصال slot ها
	if _, err := pay.DeductWithMeta(ctx, *buyer, *budget,
		"lock_rental:"+camp.ID.String(),
		"e2e-lockrental-"+camp.ID.String(),
		camp.ID.String(), `{"src":"e2e-lockrental"}`); err != nil {
		fatal("pay.deduct budget", err)
	}
	if err := st.ApproveLockRental(ctx, camp.ID, *reviewer); err != nil {
		fatal("approve campaign", err)
	}
	assigned, err := st.AssignSlotsToRental(ctx, camp.ID, *buyer, 3)
	if err != nil {
		fatal("assign slots", err)
	}
	if len(assigned) == 0 {
		fatal("assign slots", fmt.Errorf("هیچ اسلاتی وصل نشد"))
	}
	afterDeduct, _ := pay.Balance(ctx, *buyer)
	pass("approve", fmt.Sprintf("بودجه کسر شد (مانده=%.2f)، %d اسلات وصل شد", afterDeduct.Total, len(assigned)))

	// چک active بودن کمپین از مسیر واقعی handler
	active, err := st.FindActiveRentalByChannel(ctx, *channel)
	if err != nil || active == nil || active.ID != camp.ID {
		fatal("find active rental", fmt.Errorf("کمپین active برای کانال پیدا نشد"))
	}
	pass("campaign active", "FindActiveRentalByChannel مطابقت دارد")

	// ── ۴. join کاربرها (بازتولید HandleMembershipJoined) ─────────
	// member1 و member2 هر کدام یک بار join؛ member1 برای settlement، member2 برای fraud.
	recordJoin(ctx, st, camp.ID, *member1, *reward)
	recordJoin(ctx, st, camp.ID, *member2, *reward)

	// idempotency: join دوباره‌ی member1 نباید reward دوم بسازد
	if again, err := st.TryRecordJoinReward(ctx, camp.ID, *member1, *reward); err != nil {
		fatal("idempotency", err)
	} else if again {
		fatal("idempotency", fmt.Errorf("join تکراری reward دوم ساخت (باید false باشد)"))
	}
	pass("idempotency", "join تکراری member1 رد شد (firstTime=false)")

	if *emitNATS {
		emit(nc, membershipJoinedSubject, *member1, *channel)
		emit(nc, membershipJoinedSubject, *member2, *channel)
		pass("emit nats", "membership.joined روی core NATS منتشر شد")
	}

	// verify: دو reward pending و Spent = 2*reward
	campAfterJoin, _ := st.FindLockRental(ctx, camp.ID)
	wantSpent := 2 * *reward
	if !approx(campAfterJoin.Spent, wantSpent) {
		fatal("spent after joins", fmt.Errorf("Spent=%.4f انتظار=%.4f", campAfterJoin.Spent, wantSpent))
	}
	if n := countRewards(db, camp.ID, store.RewardPending); n != 2 {
		fatal("pending rewards", fmt.Errorf("تعداد pending=%d انتظار=2", n))
	}
	pass("joins recorded", fmt.Sprintf("۲ reward=pending، Spent=%.2f", campAfterJoin.Spent))

	// ── ۵. fraud روی member2 (قبل از تسویه) ──────────────────────
	if *emitNATS {
		emit(nc, "fraud.detected", *member2, *channel)
		time.Sleep(500 * time.Millisecond)
	}
	if err := st.ReversePendingRewardByUser(ctx, *channel, *member2); err != nil {
		fatal("reverse reward", err)
	}
	campAfterFraud, _ := st.FindLockRental(ctx, camp.ID)
	if !approx(campAfterFraud.Spent, *reward) {
		fatal("spent after reversal", fmt.Errorf("Spent=%.4f انتظار=%.4f (بودجه‌ی member2 باید برگشته باشد)", campAfterFraud.Spent, *reward))
	}
	if countRewardsUser(db, camp.ID, *member2, store.RewardReversed) != 1 {
		fatal("member2 reversed", fmt.Errorf("reward member2 به reversed نرفت"))
	}
	pass("fraud reversal", fmt.Sprintf("reward member2 = reversed، Spent برگشت به %.2f", campAfterFraud.Spent))

	// ── ۶. settlement روی member1 ────────────────────────────────
	// settle_at را به گذشته می‌بریم تا due شود (به‌جای انتظار ۲۴ ساعته).
	if err := db.Exec(
		`UPDATE rental_join_rewards SET settle_at = ? WHERE rental_id = ? AND telegram_id = ? AND status = ?`,
		time.Now().Add(-time.Hour), camp.ID, *member1, store.RewardPending,
	).Error; err != nil {
		fatal("force settle_at", err)
	}
	m1Before, _ := pay.Balance(ctx, *member1)
	// settleDueRewards را بازتولید می‌کنیم: هر reward due → pay.credit → SettleReward
	due, err := st.FindDueRewards(ctx, 100)
	if err != nil {
		fatal("find due rewards", err)
	}
	settled := 0
	for _, r := range due {
		if r.RentalID != camp.ID {
			continue // فقط reward های همین تست
		}
		if err := pay.Credit(ctx, r.TelegramID, r.AmountTON, "lock_rental_reward:"+r.ID.String(), `{"src":"e2e-lockrental"}`); err != nil {
			fatal("pay.credit reward", err)
		}
		if err := st.SettleReward(ctx, r.ID); err != nil {
			fatal("settle reward", err)
		}
		settled++
	}
	if settled != 1 {
		fatal("settlement", fmt.Errorf("تعداد settled=%d انتظار=1 (فقط member1)", settled))
	}
	m1After, _ := pay.Balance(ctx, *member1)
	if !approx(m1After.Total-m1Before.Total, *reward) {
		fatal("member1 credited", fmt.Errorf("افزایش موجودی=%.4f انتظار=%.4f", m1After.Total-m1Before.Total, *reward))
	}
	if countRewardsUser(db, camp.ID, *member1, store.RewardSettled) != 1 {
		fatal("member1 settled", fmt.Errorf("reward member1 به settled نرفت"))
	}
	pass("settlement", fmt.Sprintf("member1 واریز شد (+%.2f)، reward=settled", *reward))

	// ── ۷. پایان کمپین (end_at گذشته) ────────────────────────────
	if err := db.Exec(
		`UPDATE lock_rental_campaigns SET end_at = ? WHERE id = ?`,
		time.Now().Add(-time.Hour), camp.ID,
	).Error; err != nil {
		fatal("force end_at", err)
	}
	justFinished, err := st.MarkRentalDoneIfFinished(ctx, camp.ID)
	if err != nil {
		fatal("mark done", err)
	}
	if !justFinished {
		fatal("mark done", fmt.Errorf("کمپین به done نرفت"))
	}
	// آزادسازی slot ها (کاری که checkCampaignCompletion انجام می‌دهد)
	slots, _ := st.ListSlotsByRental(ctx, camp.ID)
	for _, s := range slots {
		if err := st.ReleaseSlot(ctx, s.ID); err != nil {
			fatal("release slot", err)
		}
	}
	remaining, _ := st.ListSlotsByRental(ctx, camp.ID)
	if len(remaining) != 0 {
		fatal("slots released", fmt.Errorf("هنوز %d اسلات به کمپین وصل است", len(remaining)))
	}
	final, _ := st.FindLockRental(ctx, camp.ID)
	pass("campaign done", fmt.Sprintf("status=%s، همه‌ی اسلات‌ها آزاد شدند", final.Status))

	fmt.Println("\n✅ کل چرخه‌ی اجاره‌ی قفل تأیید شد: reserve → reversal → settlement → completion")
}

// recordJoin همان کاری را می‌کند که Handler.HandleMembershipJoined در سطح store انجام می‌دهد.
func recordJoin(ctx context.Context, st *store.Store, rentalID uuid.UUID, tgID int64, reward float64) {
	rental, err := st.FindActiveRentalByChannel(ctx, mustChannelOf(ctx, st, rentalID))
	if err != nil || rental == nil {
		fatal("join: find active", fmt.Errorf("کمپین active نیست"))
	}
	first, err := st.TryRecordJoinReward(ctx, rentalID, tgID, reward)
	if err != nil {
		fatal("join: record reward", err)
	}
	if !first {
		fatal("join: record reward", fmt.Errorf("reward برای کاربر %d ثبت نشد (تکراری؟)", tgID))
	}
	if err := st.AddRentalJoinCount(ctx, rentalID, 1, reward); err != nil {
		fatal("join: add count", err)
	}
}

// mustChannelOf کانال هدف یک کمپین را می‌خواند (برای شبیه‌سازی مسیر واقعی که با channelID کار می‌کند).
func mustChannelOf(ctx context.Context, st *store.Store, rentalID uuid.UUID) int64 {
	c, err := st.FindLockRental(ctx, rentalID)
	if err != nil || c == nil {
		fatal("lookup campaign", fmt.Errorf("کمپین %s پیدا نشد", rentalID))
	}
	return c.TargetChannelID
}

func emit(nc *natsclient.Client, subject string, tgID, channelID int64) {
	_ = nc.PublishCore(subject, map[string]any{
		"telegram_id":  tgID,
		"community_id": channelID,
		"source":       "organic",
		"joined_at":    time.Now().Unix(),
	})
}

func countRewards(db *gorm.DB, rentalID uuid.UUID, status store.JoinRewardStatus) int64 {
	var n int64
	db.Model(&store.RentalJoinReward{}).Where("rental_id = ? AND status = ?", rentalID, status).Count(&n)
	return n
}

func countRewardsUser(db *gorm.DB, rentalID uuid.UUID, tgID int64, status store.JoinRewardStatus) int64 {
	var n int64
	db.Model(&store.RentalJoinReward{}).
		Where("rental_id = ? AND telegram_id = ? AND status = ?", rentalID, tgID, status).Count(&n)
	return n
}

func cleanupTest(db *gorm.DB, rentalID uuid.UUID, botIDs []int64, members ...int64) {
	db.Where("rental_id = ?", rentalID).Delete(&store.RentalJoinReward{})
	db.Where("rental_id = ?", rentalID).Delete(&store.FreeBotOwnerReward{})
	db.Where("bot_id IN ?", botIDs).Delete(&store.FreeBotSlot{})
	db.Where("id = ?", rentalID).Delete(&store.LockRentalCampaign{})
	fmt.Println("🧹 داده‌های تست پاک شد")
}

func approx(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 1e-6
}

func pass(step, detail string) { fmt.Printf("PASS  %-18s %s\n", step, detail) }

func fatal(step string, err error) {
	fmt.Printf("FAIL  %-18s %v\n", step, err)
	os.Exit(1)
}
