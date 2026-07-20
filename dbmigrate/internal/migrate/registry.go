package migrate

import (
	"fmt"
	"sort"
	"strings"
)

// Service یک سرویس دیتابیس‌دار پلتفرم را توصیف می‌کند.
type Service struct {
	// Name اسم سرویس — همان اسم پوشه در migrations/ و مقدار فلگ -service.
	Name string
	// DBName اسم دیتابیس این سرویس روی سرور Postgres
	// (رجوع deploy/migrations/000_create_databases.sql و 000_create_product_databases.sql).
	DBName string
	// Aliases اسم‌های دیگری که کاربر ممکن است در -service بدهد.
	Aliases []string
	// Note توضیح کوتاه برای خروجی `dbmigrate list`.
	Note string
}

// Registry لیست کامل سرویس‌های Postgres پلتفرم.
//
// سرویس‌های Mongo (uploader-bot, vpn-bot, archive-bot, member-bot,
// fraud-engine, log-collector, admanager-bot) این‌جا نیستند — Mongo schema
// ندارد و index هایش را خود سرویس در startup می‌سازد (رجوع
// dbmigrate/migrations/{vpn-bot,archive-bot,member-bot}/RETIRED.md برای
// تاریخچه‌ی cutover از Postgres، ۲۰۲۶-۰۷-۱۷). agentmanager و webhook-gateway
// هم اصلاً دیتابیس ندارند.
var Registry = []Service{
	{
		Name: "botmanager", DBName: "botmanager",
		Aliases: []string{"apimanager"},
		Note:    "مشترک با apimanager (دو رابط روی یک داده — رجوع CLAUDE.md بخش ۲)",
	},
	{Name: "botpay", DBName: "botpay", Note: "کیف‌پول مرکزی (لجر هش‌شده)"},
	{Name: "ads-bot", DBName: "adsbot", Aliases: []string{"adsbot"}, Note: "تبلیغات CPJ + اجاره‌ی قفل کانال"},
	{Name: "community-service", DBName: "community", Aliases: []string{"community"}, Note: "تقسیم درآمد گروه‌ها"},
	{Name: "revenue-service", DBName: "revenue", Aliases: []string{"revenue"}, Note: "قوانین کمیسیون و واریز"},
	{Name: "license-service", DBName: "license", Aliases: []string{"license"}, Note: "لایسنس ضدکپی ربات‌ها"},
	{Name: "image-registry", DBName: "imageregistry", Aliases: []string{"imageregistry"}, Note: "رجیستری imageهای مجاز"},
	{Name: "source-service", DBName: "source_svc", Note: "فوروارد MTProto (فعلاً stub)"},
}

// Find سرویس را با اسم یا alias پیدا می‌کند.
func Find(name string) (Service, error) {
	n := strings.ToLower(strings.TrimSpace(name))
	for _, s := range Registry {
		if s.Name == n {
			return s, nil
		}
		for _, a := range s.Aliases {
			if a == n {
				return s, nil
			}
		}
	}
	names := make([]string, 0, len(Registry))
	for _, s := range Registry {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	return Service{}, fmt.Errorf("سرویس ناشناخته %q — سرویس‌های معتبر: %s", name, strings.Join(names, ", "))
}
