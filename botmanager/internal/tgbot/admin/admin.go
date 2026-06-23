// Package admin منطقِ پنل ادمین را نگه می‌دارد.
// Admin وابستگی‌ها و helperهای مشترک را از core.Deps می‌گیرد.
package admin

import (
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/core"
)

// Admin هندلرِ بخشِ ادمین.
type Admin struct {
	*core.Deps
}

// New یک Admin می‌سازد.
func New(d *core.Deps) *Admin { return &Admin{Deps: d} }
