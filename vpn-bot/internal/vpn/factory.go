// Package vpn factory برای ساخت VPNPanel از نوع رشته.
package vpn

import (
	"fmt"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/vpn-bot/internal/adapters/marzban"
)

// NewPanel یک VPNPanel از نوع داده‌شده می‌سازد.
func NewPanel(panelType, baseURL, username, password string) (ports.VPNPanel, error) {
	switch panelType {
	case "marzban":
		return marzban.New(baseURL, username, password), nil
	case "marzneshin":
		// TODO: پیاده‌سازی MarzNeshin adapter
		return nil, fmt.Errorf("marzneshin adapter not implemented yet")
	case "hiddify":
		// TODO: پیاده‌سازی Hiddify adapter
		return nil, fmt.Errorf("hiddify adapter not implemented yet")
	case "xui":
		// TODO: پیاده‌سازی X-UI adapter
		return nil, fmt.Errorf("xui adapter not implemented yet")
	default:
		return nil, fmt.Errorf("unknown panel type: %s", panelType)
	}
}
