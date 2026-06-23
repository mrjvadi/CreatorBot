package tgbot

import "sync"

// convKind نوع گفتگوی چندمرحله‌ای فعال برای یک کاربر.
type convKind int

const (
	kindNone convKind = iota
	kindWithdraw
	kindTransfer
	kindAdminApprove // ادمین: دریافت txhash برای تأیید برداشت
	kindAdminReject  // ادمین: دریافت دلیل رد برداشت
	kindAdminCredit  // ادمین: افزودن اعتبار (شناسه → مبلغ)
)

// convState وضعیت یک گفتگوی چندمرحله‌ای را نگه می‌دارد.
type convState struct {
	kind convKind
	step string
	addr string // برداشت: آدرس مقصد
	toID int64  // انتقال/اعتبار: شناسه گیرنده
	arg  string // ادمین: شناسه‌ی برداشت (UUID) برای تأیید/رد
}

// stateStore نگه‌دارنده‌ی thread-safe وضعیت گفتگوها بر اساس شناسه‌ی کاربر.
//
// از مقدار (نه اشاره‌گر) استفاده می‌کند تا هیچ ساختار قابل‌تغییری بین goroutineها
// به اشتراک گذاشته نشود؛ این هم crash ناشی از دسترسی همزمان به map را رفع می‌کند
// و هم race روی فیلدهای وضعیت را.
type stateStore struct {
	mu sync.Mutex
	m  map[int64]convState
}

func newStateStore() *stateStore {
	return &stateStore{m: make(map[int64]convState)}
}

func (s *stateStore) get(id int64) (convState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.m[id]
	return st, ok
}

func (s *stateStore) set(id int64, st convState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[id] = st
}

func (s *stateStore) del(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, id)
}
