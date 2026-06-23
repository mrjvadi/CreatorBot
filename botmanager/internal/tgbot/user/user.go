// Package user منطقِ بخشِ کاربر را نگه می‌دارد.
// User وابستگی‌ها و helperهای مشترک را از core.Deps می‌گیرد.
package user

import (
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/core"
)

// User هندلرِ بخشِ کاربر.
type User struct {
	*core.Deps
}

// New یک User می‌سازد.
func New(d *core.Deps) *User { return &User{Deps: d} }
