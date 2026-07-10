// e2e-provision — تست end-to-end زنجیره‌ی ساخت ربات، بدون تلگرام.
//
// دقیقاً همان کاری را می‌کند که botmanager در Provision() انجام می‌دهد،
// ولی از بیرون و مرحله‌به‌مرحله با گزارش PASS/FAIL، تا هر حلقه‌ی زنجیره
// جدا تأیید شود:
//
//	۱. pay.credit → pay.balance → pay.deduct   (botpay + HMAC auth)
//	۲. license.issue                            (license-service)
//	۳. publish DeployCommand به deploy.<serverID> (agentmanager → image-registry → docker)
//	۴. انتظار برای agent.<serverID>.result
//
// پیش‌نیاز: سرویس‌های botpay، license-service، image-registry و agentmanager
// در حال اجرا باشند و image ثبت‌شده (uploader:dev) لوکال موجود باشد.
//
// اجرا (از پوشه‌ی tools/e2e-provision):
//
//	go run . -hmac <SERVICE_HMAC_SECRET> -bot-token <token> -server-id cbe9f282-...
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mrjvadi/creatorbot/shared-core/licenseclient"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
)

func main() {
	natsURL := flag.String("nats", "nats://localhost:4222", "آدرس NATS")
	natsUser := flag.String("nats-user", "nats", "")
	natsPass := flag.String("nats-pass", "nats_secret", "")
	hmacSecret := flag.String("hmac", os.Getenv("SERVICE_HMAC_SECRET"), "SERVICE_HMAC_SECRET (همان مقدار سرویس‌ها)")
	botToken := flag.String("bot-token", "", "توکن تلگرام ربات uploader ای که deploy می‌شود")
	serverID := flag.String("server-id", "cbe9f282-06a4-4c23-83b2-fe52b8ff9e17", "SERVER_ID سرور هدف (agentmanager)")
	tgID := flag.Int64("tg", 900000001, "TelegramID کاربر تست برای کیف‌پول")
	image := flag.String("image", "uploader", "")
	tag := flag.String("tag", "dev", "")
	keep := flag.Bool("keep", false, "container را بعد از تست نگه دار (پیش‌فرض: می‌ماند؛ این ابزار پاک نمی‌کند)")
	flag.Parse()
	_ = keep

	if *hmacSecret == "" || *botToken == "" {
		fmt.Fprintln(os.Stderr, "فلگ‌های -hmac و -bot-token لازم‌اند")
		os.Exit(2)
	}
	botID, err := botIDFromToken(*botToken)
	if err != nil {
		fatal("bot token", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	nc, err := natsclient.New(natsclient.Config{
		URL: *natsURL, Username: *natsUser, Password: *natsPass, Name: "e2e-provision",
	})
	if err != nil {
		fatal("NATS connect", err)
	}
	defer nc.Close()
	pass("NATS connect", *natsURL)

	// ── قبل از هر چیز: گوش‌دادن به نتیجه‌ی deploy ─────────────────
	resultCh := make(chan protocol.ResultMsg, 4)
	containerName := fmt.Sprintf("uploader_%d", botID)
	if err := nc.Subscribe(protocol.ResultSubject(*serverID), func(data []byte) {
		var msg protocol.ResultMsg
		if json.Unmarshal(data, &msg) == nil && msg.ContainerName == containerName {
			resultCh <- msg
		}
	}); err != nil {
		fatal("subscribe result", err)
	}

	// ── ۱. کیف‌پول (botpay از طریق NATS + HMAC، مثل خود botmanager) ──
	pay := natspayclient.New(nc, nil, natspayclient.Config{
		ServiceID:  "botmanager",
		ServiceKey: auth.ComputeServiceKey(*hmacSecret, "botmanager"),
		Timeout:    5 * time.Second,
	})
	if err := pay.Credit(ctx, *tgID, 1.5, "e2e-test-credit", `{"src":"e2e-provision"}`); err != nil {
		fatal("pay.credit", err)
	}
	pass("pay.credit", "1.5 TON به کاربر تست")
	bal, err := pay.Balance(ctx, *tgID)
	if err != nil {
		fatal("pay.balance", err)
	}
	pass("pay.balance", fmt.Sprintf("total=%.2f TON", bal.Total))
	newBal, err := pay.Deduct(ctx, *tgID, 0.5, "e2e-test-deduct", fmt.Sprintf("e2e-%d", time.Now().UnixNano()))
	if err != nil {
		fatal("pay.deduct", err)
	}
	pass("pay.deduct", fmt.Sprintf("0.5 TON کسر شد، مانده=%.2f", newBal))

	// ── ۲. صدور لایسنس (license-service) ─────────────────────────
	lic := licenseclient.New(nc, licenseclient.Config{
		ServiceID:  "botmanager",
		ServiceKey: auth.ComputeServiceKey(*hmacSecret, "botmanager"),
	})
	instanceID := "bot_" + strconv.FormatInt(botID, 10)
	licToken, err := lic.Issue(ctx, botID, instanceID, "e2e-owner", *serverID, "")
	if err != nil {
		fatal("license.issue", err)
	}
	pass("license.issue", fmt.Sprintf("JWT صادر شد (%d بایت)", len(licToken)))

	// ── ۳. دستور deploy (همان payload ای که wizard.go می‌سازد) ────
	cmd := protocol.DeployCommand{
		Type:          protocol.MsgDeploy,
		ContainerName: containerName,
		ImageName:     *image,
		ImageTag:      *tag,
		EnvVars: map[string]string{
			"BOT_TOKEN":      *botToken,
			"INSTANCE_ID":    instanceID,
			"OWNER_TELEGRAM": strconv.FormatInt(*tgID, 10),
			"OWNER_ID":       strconv.FormatInt(*tgID, 10),
			"PLAN_ID":        "",
			"JWT_TOKEN":      "",
			"LICENSE_TOKEN":  licToken,
			"SERVER_ID":      *serverID,
		},
	}
	if err := nc.Publish(ctx, protocol.DeploySubject(*serverID), cmd); err != nil {
		fatal("publish deploy", err)
	}
	pass("deploy publish", protocol.DeploySubject(*serverID))

	// ── ۴. انتظار نتیجه از agentmanager ──────────────────────────
	select {
	case msg := <-resultCh:
		if !msg.Success {
			fatal("deploy result", fmt.Errorf("agentmanager خطا داد: %s", msg.Error))
		}
		pass("deploy result", fmt.Sprintf("container=%s success=true", msg.ContainerName))
	case <-time.After(90 * time.Second):
		fatal("deploy result", fmt.Errorf("۹۰ ثانیه گذشت و نتیجه‌ای از agent.%s.result نیامد", *serverID))
	}

	fmt.Println("\n✅ زنجیره کامل شد — حالا وضعیت container را چک کنید:")
	fmt.Printf("   docker logs %s --tail 20\n", containerName)
}

func botIDFromToken(token string) (int64, error) {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("فرمت توکن نامعتبر")
	}
	return strconv.ParseInt(parts[0], 10, 64)
}

func pass(step, detail string) { fmt.Printf("PASS  %-16s %s\n", step, detail) }

func fatal(step string, err error) {
	fmt.Printf("FAIL  %-16s %v\n", step, err)
	os.Exit(1)
}
