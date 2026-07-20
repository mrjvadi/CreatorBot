// authz.go — تأیید اصالت/تازگی/یک‌بارمصرف‌بودن دستورهای deploy.
//
// deploy بالاترین امتیاز پلتفرم است (اجرای container واقعی روی سرور). این
// verifier همان الگوی source-service/internal/bus.authorize را پیاده می‌کند —
// HMAC هویت سرویس + پنجره‌ی تازگی + nonce یک‌بارمصرف — ولی nonce store
// درون‌پردازه‌ای است چون agentmanager به Redis وصل نیست.
package queue

import (
	"fmt"
	"sync"
	"time"

	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
)

// authWindow حداکثر عمر مجاز یک دستور از لحظه‌ی issued_at (همتای
// requestAuthWindow در source-service).
const authWindow = 5 * time.Minute

// futureSkew حداکثر انحراف مجاز رو به آینده برای اختلاف ساعت فرستنده/گیرنده.
const futureSkew = 30 * time.Second

// Verifier اصالت دستورهای deploy را بررسی می‌کند.
type Verifier struct {
	hmacSecret string
	nonces     *nonceStore
}

// NewVerifier یک Verifier با کلید مشترک HMAC می‌سازد. کلید خالی یعنی همه‌ی
// دستورها رد می‌شوند (fail-closed).
func NewVerifier(hmacSecret string) *Verifier {
	return &Verifier{hmacSecret: hmacSecret, nonces: newNonceStore()}
}

// Check اگر دستور معتبر نباشد خطای غیرتهی برمی‌گرداند. همه‌ی شرط‌ها باید
// برقرار باشند: کلید HMAC معتبر، nonce غیرخالی و تازه، و issued_at داخل پنجره.
func (v *Verifier) Check(cmd protocol.DeployCommand) error {
	if v.hmacSecret == "" || !auth.ValidateServiceKey(v.hmacSecret, cmd.ServiceID, cmd.ServiceKey) {
		return fmt.Errorf("unauthorized")
	}
	if cmd.Nonce == "" {
		return fmt.Errorf("nonce required")
	}
	issued := time.Unix(cmd.IssuedAt, 0)
	if cmd.IssuedAt <= 0 || time.Since(issued) > authWindow || time.Until(issued) > futureSkew {
		return fmt.Errorf("request expired")
	}
	if !v.nonces.claim(cmd.ServiceID+":"+cmd.Nonce, authWindow) {
		return fmt.Errorf("replayed request")
	}
	return nil
}

// nonceStore یک مجموعه‌ی درون‌حافظه‌ای از nonceهای دیده‌شده با انقضا است.
// حافظه با پاکسازی تنبل (هنگام هر claim) کران‌دار می‌ماند.
type nonceStore struct {
	mu   sync.Mutex
	seen map[string]time.Time // key → زمان انقضا
}

func newNonceStore() *nonceStore {
	return &nonceStore{seen: make(map[string]time.Time)}
}

// claim اگر key تازه باشد آن را ثبت و true برمی‌گرداند؛ اگر قبلاً (و هنوز
// منقضی‌نشده) دیده شده باشد false برمی‌گرداند (replay).
func (n *nonceStore) claim(key string, ttl time.Duration) bool {
	now := time.Now()
	n.mu.Lock()
	defer n.mu.Unlock()

	// پاکسازی تنبلِ ورودی‌های منقضی تا map رشد نامحدود نکند.
	for k, exp := range n.seen {
		if now.After(exp) {
			delete(n.seen, k)
		}
	}

	if exp, ok := n.seen[key]; ok && now.Before(exp) {
		return false
	}
	n.seen[key] = now.Add(ttl)
	return true
}
