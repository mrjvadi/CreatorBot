// Package vpn factory برای ساخت VPNPanel از نوع رشته.
package vpn

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/hiddify"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/marzban"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/marzneshin"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/xui"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// NewPanel یک VPNPanel از نوع داده‌شده می‌سازد.
// panelType: "marzban" | "marzneshin" | "hiddify" | "xui"
// config: رشته‌ی تنظیمات اضافی جدا شده با "|"
//   - hiddify: "<adminPath>|<apiKey>"
//   - xui:     "<inboundID>"
//   - marzban/marzneshin: نیازی به config ندارند
func NewPanel(panelType, baseURL, username, password, config string) (ports.VPNPanel, error) {
	switch panelType {
	case "marzban":
		return marzban.New(baseURL, username, password), nil

	case "marzneshin":
		return marzneshin.New(baseURL, username, password), nil

	case "hiddify":
		// config = "<adminPath>|<apiKey>"
		parts := strings.SplitN(config, "|", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("hiddify config format: '<adminPath>|<apiKey>'")
		}
		return hiddify.New(baseURL, parts[0], parts[1]), nil

	case "xui":
		// config = "<inboundID>"
		inboundID := 1
		if config != "" {
			id, err := strconv.Atoi(config)
			if err != nil {
				return nil, fmt.Errorf("xui inboundID must be integer: %s", config)
			}
			inboundID = id
		}
		return xui.New(baseURL, username, password, inboundID), nil

	default:
		return nil, fmt.Errorf("unknown panel type: %s (valid: marzban, marzneshin, hiddify, xui)", panelType)
	}
}

// SupportedPanels لیست panel های پشتیبانی‌شده.
func SupportedPanels() []string {
	return []string{"marzban", "marzneshin", "hiddify", "xui"}
}
