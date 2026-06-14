// Package tgbot ربات تلگرام botpay — مدیریت کیف پول TON.
package tgbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/google/uuid"
	"github.com/mrjvadi/creatorbot/botpay/internal/store"
	"github.com/mrjvadi/creatorbot/botpay/internal/wallet"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// ── Handler ────────────────────────────────────────────────

type Handler struct {
	wallet  *wallet.Service
	st      *store.Store
	ownerID int64
	log     ports.Logger
	bot     *tele.Bot // برای ارسال push notification
}

func New(w *wallet.Service, st *store.Store, ownerID int64, log ports.Logger) *Handler {
	return &Handler{wallet: w, st: st, ownerID: ownerID, log: log}
}

// SetBot ربات را تنظیم و notifier را فعال می‌کند.
func (h *Handler) SetBot(b *tele.Bot) {
	h.bot = b
	h.wallet.SetNotifier(h)
}

// SendHTML پیاده‌سازی wallet.Notifier.
func (h *Handler) SendHTML(ctx context.Context, telegramID int64, html string) error {
	if h.bot == nil {
		return nil
	}
	chat := &tele.Chat{ID: telegramID}
	_, err := h.bot.Send(chat, html, tele.ModeHTML)
	return err
}

func Register(b *tele.Bot, h *Handler) {
	b.Handle("/start",    h.onStart)
	b.Handle("/wallet",   h.onWallet)
	b.Handle("/deposit",  h.onDeposit)
	b.Handle("/withdraw", h.onWithdraw)
	b.Handle("/history",  h.onHistory)
	b.Handle("/help",     h.onHelp)
	// ادمین
	b.Handle("/admin",    h.onAdmin)
	b.Handle("/addcredit",h.onAddCredit)
	b.Handle("/withdraws",h.onAdminWithdraws)
	b.Handle("/approve",  h.onApproveWithdraw)
	b.Handle("/reject",   h.onRejectWithdraw)

	b.Handle(tele.OnText,     h.onText)
	b.Handle(tele.OnCallback, h.onCallback)
}

// ── دکمه‌ها ────────────────────────────────────────────────

const (
	btnWallet   = "💳 کیف پول"
	btnDeposit  = "📥 واریز TON"
	btnWithdraw = "📤 برداشت"
	btnHistory  = "📋 تاریخچه"
	btnHelp     = "❓ راهنما"
	btnTransfer = "🔄 انتقال"
	btnCancel   = "❌ انصراف"
	btnCheck    = "🔄 بررسی مجدد"
)

func kbMain() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(btnWallet)),
		kb.Row(kb.Text(btnDeposit), kb.Text(btnWithdraw)),
		kb.Row(kb.Text(btnTransfer), kb.Text(btnHistory)),
		kb.Row(kb.Text(btnHelp)),
	)
	return kb
}

func kbCancelOnly() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(btnCancel)))
	return kb
}

// ── state برداشت ───────────────────────────────────────────

type withdrawState struct{ step, addr string }

var wStates = map[int64]*withdrawState{}

type transferState struct{ step string; toID int64 }

var tStates = map[int64]*transferState{}

// ── /start ────────────────────────────────────────────────

func (h *Handler) onStart(c tele.Context) error {
	ctx := context.Background()
	w, err := h.wallet.GetOrCreate(ctx, c.Sender().ID)
	if err != nil {
		return c.Send("❌ خطا در بارگذاری کیف پول.")
	}
	return c.Send(fmt.Sprintf(
		"👛 <b>BotPay</b>\n\n"+
			"سلام <b>%s</b>!\n\n"+
			"💎 موجودی TON: <b>%.4f</b>\n"+
			"🎁 اعتبار: <b>%.4f</b>\n"+
			"📊 مجموع: <b>%.4f TON</b>\n\n"+
			"با دکمه‌های زیر کیف پول خود را مدیریت کنید:",
		c.Sender().FirstName,
		w.BalanceTON(), w.CreditTON(), w.TotalTON(),
	), tele.ModeHTML, kbMain())
}

// ── 💳 کیف پول ────────────────────────────────────────────

func (h *Handler) onWallet(c tele.Context) error {
	ctx := context.Background()
	w, _ := h.wallet.GetOrCreate(ctx, c.Sender().ID)

	frozen := ""
	if w.Frozen > 0 {
		frozen = fmt.Sprintf("\n🔒 در انتظار: <b>%.4f TON</b>", wallet.NanoToTON(w.Frozen))
	}

	return c.Send(fmt.Sprintf(
		"💳 <b>کیف پول شما</b>\n\n"+
			"💎 موجودی TON: <b>%.4f</b>\n"+
			"🎁 اعتبار داخلی: <b>%.4f</b>\n"+
			"📊 مجموع قابل استفاده: <b>%.4f TON</b>%s\n\n"+
			"📥 آدرس دریافت (همین آدرس برای همه):\n<code>%s</code>\n\n"+
			"⚠️ هنگام پرداخت <b>comment</b> را حتماً وارد کنید.",
		w.BalanceTON(), w.CreditTON(), w.TotalTON(), frozen, w.TONAddress,
	), tele.ModeHTML, kbMain())
}

// ── 📥 واریز ─────────────────────────────────────────────

func (h *Handler) onDeposit(c tele.Context) error {
	ctx := context.Background()

	code, payURL, err := h.wallet.DepositInstructions(
		ctx, c.Sender().ID, 0, "manual", "direct",
	)
	if err != nil {
		return c.Send("❌ خطا در ساخت فاکتور.")
	}

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.URL("💎 باز کردن TON Wallet", payURL)),
		kb.Row(kb.Data("🔄 بررسی دریافت", "check_deposit:"+code)),
	)

	return c.Send(fmt.Sprintf(
		"<b>📥 واریز TON</b>\n\n"+
			"مراحل:\n"+
			"1️⃣ روی «باز کردن TON Wallet» کلیک کنید\n"+
			"2️⃣ مقدار دلخواه را وارد کنید\n"+
			"3️⃣ در قسمت <b>Comment</b> کد زیر را وارد کنید\n"+
			"4️⃣ تراکنش را تأیید کنید\n\n"+
			"🔑 کد واریز:\n<code>%s</code>\n\n"+
			"⚠️ بدون این کد واریز شناسایی <b>نمی‌شود</b>\n"+
			"⏰ اعتبار کد: ۳۰ دقیقه",
		code,
	), tele.ModeHTML, kb)
}

// ── 📤 برداشت ────────────────────────────────────────────

func (h *Handler) onWithdraw(c tele.Context) error {
	ctx := context.Background()
	w, _ := h.wallet.GetOrCreate(ctx, c.Sender().ID)

	minTON := float64(wallet.MinWithdrawNano) / 1e9
	feeTON := float64(wallet.NetworkFeeNano) / 1e9

	if w.TotalTON() < minTON+feeTON {
		return c.Send(fmt.Sprintf(
			"❌ موجودی کافی نیست.\n\nحداقل برداشت: <b>%.1f TON</b>\nکارمزد شبکه: <b>%.2f TON</b>",
			minTON, feeTON,
		), tele.ModeHTML, kbMain())
	}

	wStates[c.Sender().ID] = &withdrawState{step: "addr"}
	return c.Send(fmt.Sprintf(
		"<b>📤 برداشت TON</b>\n\n"+
			"موجودی: <b>%.4f TON</b>\n"+
			"کارمزد شبکه: <b>%.2f TON</b>\n\n"+
			"آدرس TON مقصد را وارد کنید:",
		w.TotalTON(), feeTON,
	), tele.ModeHTML, kbCancelOnly())
}

// ── 📋 تاریخچه ───────────────────────────────────────────

func (h *Handler) onHistory(c tele.Context) error {
	ctx := context.Background()
	w, _ := h.wallet.GetOrCreate(ctx, c.Sender().ID)

	txs, err := h.st.ListTransactions(ctx, w.ID, 15)
	if err != nil || len(txs) == 0 {
		return c.Send("📋 تاریخچه‌ای وجود ندارد.", kbMain())
	}

	lines := []string{"<b>📋 ۱۵ تراکنش اخیر</b>", ""}
	for _, tx := range txs {
		icon, label := txIcon(tx.Type)
		amt := wallet.NanoToTON(tx.Amount)
		sign := "+"
		if amt < 0 {
			sign = ""
			amt = -amt
		}
		status := ""
		if tx.Status == store.TxPending {
			status = " ⏳"
		}
		desc := tx.Description
		if desc == "" {
			desc = label
		}
		if len(desc) > 25 {
			desc = desc[:25] + "..."
		}
		lines = append(lines, fmt.Sprintf(
			"%s %s%.4f TON%s — %s\n   <i>%s</i>",
			icon, sign, amt, status, tx.CreatedAt.Format("01/02 15:04"), desc,
		))
	}

	return c.Send(strings.Join(lines, "\n"), tele.ModeHTML, kbMain())
}

func txIcon(t store.TxType) (icon, label string) {
	switch t {
	case store.TxDeposit:
		return "📥", "واریز TON"
	case store.TxWithdraw:
		return "📤", "برداشت TON"
	case store.TxCreditAdd:
		return "🎁", "افزایش اعتبار"
	case store.TxPayment:
		return "💸", "پرداخت"
	case store.TxRefund:
		return "↩️", "بازگشت وجه"
	}
	return "💰", string(t)
}

// ── ❓ راهنما ────────────────────────────────────────────

func (h *Handler) onHelp(c tele.Context) error {
	return c.Send(
		"<b>❓ راهنمای BotPay</b>\n\n"+
			"<b>💎 موجودی TON:</b>\n"+
			"از blockchain واریز می‌شود.\n"+
			"یک آدرس مشترک برای همه کاربران.\n"+
			"هر واریز باید <b>comment</b> اختصاصی داشته باشد.\n\n"+
			"<b>🎁 اعتبار داخلی:</b>\n"+
			"توسط ادمین اضافه می‌شود.\n"+
			"در پرداخت‌ها اول از اعتبار کم می‌شود.\n\n"+
			"<b>📥 واریز TON:</b>\n"+
			"دکمه «واریز» → کد دریافت کنید → به آدرس با comment بفرستید\n\n"+
			"<b>📤 برداشت:</b>\n"+
			"دکمه «برداشت» → آدرس → مبلغ → تأیید ادمین\n"+
			"کارمزد شبکه: ۰.۰۱ TON\n\n"+
			"<b>💳 موجودی برای سرویس‌ها:</b>\n"+
			"هر بار که پلن یا سرویس می‌خرید از کیف پول کم می‌شود.\n\n"+
			"پشتیبانی: @support",
		tele.ModeHTML, kbMain(),
	)
}

// ── onText ────────────────────────────────────────────────

func (h *Handler) onText(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	text := strings.TrimSpace(c.Text())

	// state برداشت
	if st, ok := wStates[uid]; ok {
		return h.handleWithdrawState(ctx, c, st, text)
	}
	// state انتقال
	if st, ok := tStates[uid]; ok {
		return h.handleTransferState(ctx, c, st, text)
	}

	switch text {
	case btnWallet:   return h.onWallet(c)
	case btnDeposit:  return h.onDeposit(c)
	case btnWithdraw: return h.onWithdraw(c)
	case btnTransfer: return h.onTransfer(c)
	case btnHistory:  return h.onHistory(c)
	case btnHelp:     return h.onHelp(c)
	}
	return nil
}

func (h *Handler) handleWithdrawState(ctx context.Context, c tele.Context, st *withdrawState, text string) error {
	uid := c.Sender().ID

	if text == btnCancel || text == "/cancel" {
		delete(wStates, uid)
		return c.Send("لغو شد.", kbMain())
	}

	switch st.step {
	case "addr":
		// validate آدرس TON (شروع با UQ یا EQ و ۴۸ کاراکتر)
		if len(text) < 48 || (!strings.HasPrefix(text, "UQ") && !strings.HasPrefix(text, "EQ")) {
			return c.Send(
				"❌ آدرس TON نامعتبر.\n\nآدرس باید با <code>UQ</code> یا <code>EQ</code> شروع شود.",
				tele.ModeHTML,
			)
		}
		st.addr = text
		st.step = "amount"

		w, _ := h.wallet.GetOrCreate(ctx, uid)
		feeTON := float64(wallet.NetworkFeeNano) / 1e9
		maxAmount := w.TotalTON() - feeTON

		return c.Send(fmt.Sprintf(
			"آدرس: <code>%s...%s</code>\n\n"+
				"حداکثر قابل برداشت: <b>%.4f TON</b>\n"+
				"(کارمزد %.2f TON کسر می‌شود)\n\n"+
				"مقدار برداشت را وارد کنید:",
			text[:6], text[len(text)-4:],
			maxAmount, feeTON,
		), tele.ModeHTML, kbCancelOnly())

	case "amount":
		amt, err := strconv.ParseFloat(text, 64)
		if err != nil || amt <= 0 {
			return c.Send("❌ عدد معتبر وارد کنید.")
		}

		delete(wStates, uid)
		amountNano := wallet.TONToNano(amt)

		req, err := h.wallet.RequestWithdraw(ctx, uid, st.addr, amountNano, "")
		if err != nil {
			return c.Send("❌ " + err.Error())
		}

		return c.Send(fmt.Sprintf(
			"✅ <b>درخواست برداشت ثبت شد</b>\n\n"+
				"💰 مبلغ: <b>%.4f TON</b>\n"+
				"🔧 کارمزد: <b>%.2f TON</b>\n"+
				"📥 آدرس: <code>%s</code>\n"+
				"🆔 ID: <code>%s</code>\n\n"+
				"ادمین در اسرع وقت پردازش خواهد کرد.",
			amt,
			float64(wallet.NetworkFeeNano)/1e9,
			req.ToAddress,
			req.ID,
		), tele.ModeHTML, kbMain())
	}

	return nil
}

// ── onCallback ───────────────────────────────────────────

func (h *Handler) onCallback(c tele.Context) error {
	ctx := context.Background()
	data := c.Callback().Data
	if len(data) > 0 && data[0] == '\f' {
		data = data[1:]
	}
	defer c.Respond()

	parts := strings.SplitN(data, ":", 2)
	switch parts[0] {
	case "check_deposit":
		if len(parts) < 2 {
			return nil
		}
		code := parts[1]
		// بررسی وضعیت invoice
		inv, _ := h.st.FindInvoiceByCode(ctx, code)
		if inv == nil {
			return c.Edit("⏳ هنوز دریافت نشده.\n\nطبق معمول ۱-۵ دقیقه طول می‌کشد.")
		}
		if inv.Status == store.InvoicePaid {
			w, _ := h.wallet.GetOrCreate(ctx, c.Sender().ID)
			return c.Edit(fmt.Sprintf(
				"✅ <b>واریز تأیید شد!</b>\n\n"+
					"💰 مبلغ: <b>%.4f TON</b>\n"+
					"💳 موجودی جدید: <b>%.4f TON</b>",
				wallet.NanoToTON(inv.Amount),
				w.TotalTON(),
			), tele.ModeHTML)
		}
		return c.Edit("⏳ هنوز تأیید نشده.\n\nچند دقیقه دیگر دوباره بررسی کنید.")
	}
	return nil
}

// ════════════════════════════════════════════════════════════
// ادمین
// ════════════════════════════════════════════════════════════

func (h *Handler) isAdmin(c tele.Context) bool {
	return c.Sender().ID == h.ownerID
}

func (h *Handler) onAdmin(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()
	stats, _ := h.st.GetStats(ctx)
	pending, _ := h.st.ListPendingWithdrawals(ctx)

	return c.Send(fmt.Sprintf(
		"<b>🔧 پنل ادمین BotPay</b>\n\n"+
			"👥 کل کیف‌پول‌ها: %d\n"+
			"📥 کل واریزها: %.4f TON\n"+
			"💸 کل پرداخت‌ها: %.4f TON\n"+
			"⏳ برداشت منتظر: %d\n\n"+
			"<b>دستورات:</b>\n"+
			"/withdraws — لیست برداشت‌های منتظر\n"+
			"/approve &lt;id&gt; &lt;txhash&gt; — تأیید برداشت\n"+
			"/reject &lt;id&gt; &lt;دلیل&gt; — رد برداشت\n"+
			"/addcredit &lt;telegram_id&gt; &lt;amount&gt; — اضافه کردن اعتبار",
		stats.TotalWallets,
		stats.TotalDeposits,
		stats.TotalPayments,
		len(pending),
	), tele.ModeHTML)
}

func (h *Handler) onAdminWithdraws(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()
	reqs, _ := h.st.ListPendingWithdrawals(ctx)

	if len(reqs) == 0 {
		return c.Send("هیچ درخواست برداشت منتظری وجود ندارد.")
	}

	lines := []string{fmt.Sprintf("<b>⏳ برداشت‌های منتظر (%d)</b>", len(reqs)), ""}
	for _, r := range reqs {
		lines = append(lines, fmt.Sprintf(
			"🆔 <code>%s</code>\n"+
				"💰 %.4f TON + %.2f کارمزد\n"+
				"📥 <code>%s</code>\n"+
				"📅 %s",
			r.ID,
			wallet.NanoToTON(r.Amount),
			wallet.NanoToTON(r.Fee),
			r.ToAddress,
			r.CreatedAt.Format("01/02 15:04"),
		))
		lines = append(lines, "")
	}

	return c.Send(strings.Join(lines, "\n"), tele.ModeHTML)
}

func (h *Handler) onApproveWithdraw(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	parts := strings.Fields(c.Message().Payload)
	if len(parts) < 2 {
		return c.Send("استفاده: /approve <id> <txhash>")
	}

	ctx := context.Background()
	withdrawID, err := uuid.Parse(parts[0])
	if err != nil {
		return c.Send("❌ ID نامعتبر.")
	}

	txHash := parts[1]
	if err := h.st.CompleteWithdraw(ctx, withdrawID, txHash); err != nil {
		return c.Send("❌ خطا: " + err.Error())
	}

	return c.Send(fmt.Sprintf("✅ برداشت <code>%s</code> تأیید شد.", parts[0]), tele.ModeHTML)
}

func (h *Handler) onRejectWithdraw(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	payload := strings.TrimSpace(c.Message().Payload)
	if payload == "" {
		return c.Send("استفاده: /reject <id> <دلیل>")
	}

	spaceIdx := strings.Index(payload, " ")
	if spaceIdx < 0 {
		return c.Send("استفاده: /reject <id> <دلیل>")
	}

	idStr := payload[:spaceIdx]
	reason := payload[spaceIdx+1:]

	ctx := context.Background()
	withdrawID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Send("❌ ID نامعتبر.")
	}

	if err := h.st.RejectWithdraw(ctx, withdrawID, reason); err != nil {
		return c.Send("❌ خطا: " + err.Error())
	}

	return c.Send(fmt.Sprintf("🚫 برداشت <code>%s</code> رد شد.", idStr), tele.ModeHTML)
}

func (h *Handler) onAddCredit(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	parts := strings.Fields(c.Message().Payload)
	if len(parts) < 2 {
		return c.Send("استفاده: /addcredit <telegram_id> <amount_ton>")
	}

	var telegramID int64
	fmt.Sscanf(parts[0], "%d", &telegramID)
	amt, err := strconv.ParseFloat(parts[1], 64)
	if err != nil || telegramID == 0 || amt <= 0 {
		return c.Send("❌ پارامتر نامعتبر.")
	}

	ctx := context.Background()
	w, err := h.wallet.GetOrCreate(ctx, telegramID)
	if err != nil {
		return c.Send("❌ کیف پول یافت نشد.")
	}

	desc := "افزایش اعتبار توسط ادمین"
	if len(parts) > 2 {
		desc = strings.Join(parts[2:], " ")
	}

	if err := h.st.AddCredit(ctx, w.ID, wallet.TONToNano(amt), desc); err != nil {
		return c.Send("❌ خطا: " + err.Error())
	}

	return c.Send(fmt.Sprintf(
		"✅ <b>%.4f TON</b> اعتبار به کاربر <code>%d</code> اضافه شد.",
		amt, telegramID,
	), tele.ModeHTML)
}

// ── helpers ───────────────────────────────────────────────

func parseUUID(s string) (interface{ String() string }, error) {
	return uuid.Parse(s)
}

// ── 💸 انتقال داخلی ──────────────────────────────────────

func (h *Handler) onTransfer(c tele.Context) error {
	ctx := context.Background()
	w, _ := h.wallet.GetOrCreate(ctx, c.Sender().ID)
	if w.TotalTON() <= 0 {
		return c.Send("❌ موجودی شما برای انتقال کافی نیست.", kbMain())
	}

	tStates[c.Sender().ID] = &transferState{step: "to_id"}
	return c.Send(
		"<b>💸 انتقال داخلی</b>\n\n"+
			"Telegram ID کاربر مقصد را وارد کنید:\n"+
			"(عدد شناسه تلگرام — مثال: <code>123456789</code>)",
		tele.ModeHTML, kbCancelOnly(),
	)
}

func (h *Handler) handleTransferState(ctx context.Context, c tele.Context, st *transferState, text string) error {
	uid := c.Sender().ID

	if text == btnCancel || text == "/cancel" {
		delete(tStates, uid)
		return c.Send("لغو شد.", kbMain())
	}

	switch st.step {
	case "to_id":
		var toID int64
		fmt.Sscanf(strings.TrimSpace(text), "%d", &toID)
		if toID <= 0 || toID == uid {
			return c.Send("❌ ID نامعتبر یا نمی‌توانید به خودتان انتقال دهید.")
		}

		// بررسی وجود گیرنده
		toWallet, err := h.wallet.Store().GetWallet(ctx, toID)
		if err != nil || toWallet == nil {
			return c.Send("❌ کاربر مقصد در سیستم ثبت نشده است.")
		}

		st.toID = toID
		st.step = "amount"

		fromWallet, _ := h.wallet.GetOrCreate(ctx, uid)
		return c.Send(
			fmt.Sprintf(
				"گیرنده: <code>%d</code>\n\n"+
					"موجودی شما: <b>%.4f TON</b>\n"+
					"مبلغ انتقال را وارد کنید:",
				toID, fromWallet.TotalTON(),
			),
			tele.ModeHTML, kbCancelOnly(),
		)

	case "amount":
		amt, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil || amt <= 0 {
			return c.Send("❌ عدد معتبر وارد کنید.")
		}

		toID := st.toID
		delete(tStates, uid)

		amountNano := wallet.TONToNano(amt)
		if err := h.wallet.Transfer(ctx, uid, toID, amountNano, fmt.Sprintf("%d→%d", uid, toID)); err != nil {
			return c.Send("❌ " + err.Error())
		}

		return c.Send(
			fmt.Sprintf(
				"✅ <b>انتقال انجام شد</b>\n\n"+
					"💸 %.4f TON به <code>%d</code> ارسال شد.",
				amt, toID,
			),
			tele.ModeHTML, kbMain(),
		)
	}
	return nil
}
