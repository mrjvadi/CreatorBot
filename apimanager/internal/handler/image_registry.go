package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ── Image Registry proxy ─────────────────────────────────────
//
// بازخورد کاربر ۲۰۲۶-۰۷-۰۵: «باید یه صفحه برای آپلود ایمیج داشته باشم». طبق معماریِ جدیدِ
// پلتفرم (که کاربر توضیح داد)، agentmanager دیگر خودش لیستِ ایمیج مجاز را نگه نمی‌دارد —
// یک سرویسِ جداگانه به اسم "image-registry" این کار را می‌کند: agentmanager قبل از هر
// deploy با GET /v1/check از آن می‌پرسد آیا ایمیج مجاز است، و اگر فایلش آماده باشد آن را
// دانلود می‌کند.
//
// ⚠️ به‌روزرسانی بعد از بررسی ۲۰۲۶-۰۷-۰۵: مسیرهای proxy‌شده‌ی زیر با کد واقعیِ
// image-registry/internal/api/api.go تطبیق داده شدند و درست‌اند (GET/POST /v1/images,
// POST /v1/images/:id/file, PATCH/DELETE /v1/images/:id, GET /v1/check) — نگرانیِ قبلیِ این
// کامنت درباره‌ی «مشخصاتِ خراب/ناخوانا» روی خودِ اسمِ endpoint ها بی‌مورد بود.
//
// ولی یک مشکلِ واقعی و جدی پیدا شد: هدرِ X-Admin-Key که این‌جا فرستاده می‌شود، سمتِ
// image-registry فقط روی /v1/callers/* چک می‌شود؛ /v1/images* (همینایی که این فایل صدا می‌زند)
// فقط بر اساسِ IP واقعیِ فراخوان + فیلدِ CanWrite تصمیم می‌گیرند. یعنی تا وقتی IP خروجیِ
// خودِ apimanager به‌عنوانِ یک AllowedCaller با CanWrite=true در image-registry ثبت نشود، هر
// درخواستِ ثبت/آپلود/حذف از این پنل با ۴۰۳ شکست می‌خورد — صرف‌نظر از درست‌بودنِ X-Admin-Key.
// جزئیات کامل و راه‌حل‌ها: apimanager/NEEDS.md بخش ۰، و image-registry/README.md.
//
// معماری همچنان همان است: apimanager یک «پروکسیِ کور» است (بدنه/status عیناً از
// image-registry برگردانده می‌شود، بدون parse دقیقِ فیلدها) — اگر مسیر/فیلدی در آینده عوض شد،
// خودِ image-registry پیام خطا می‌دهد، نه یک شکستِ بی‌صدا. IMAGE_REGISTRY_URL/
// IMAGE_REGISTRY_ADMIN_KEY باید در env تنظیم شوند؛ تا وقتی تنظیم نشده‌اند همه‌ی این
// endpoint ها 503 برمی‌گردانند.
func (h *Handler) proxyImageRegistry(c *gin.Context, method, path string) {
	if h.imageRegistryURL == "" {
		fail(c, http.StatusServiceUnavailable, "image registry not configured (IMAGE_REGISTRY_URL)")
		return
	}

	url := strings.TrimRight(h.imageRegistryURL, "/") + path
	req, err := http.NewRequestWithContext(c.Request.Context(), method, url, c.Request.Body)
	if err != nil {
		fail(c, http.StatusInternalServerError, "request build failed")
		return
	}
	if h.imageRegistryAdminKey != "" {
		req.Header.Set("X-Admin-Key", h.imageRegistryAdminKey)
	}
	if ct := c.GetHeader("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	req.ContentLength = c.Request.ContentLength

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fail(c, http.StatusBadGateway, "image registry unreachable: "+err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fail(c, http.StatusBadGateway, "image registry response read failed")
		return
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, body)
}

// GET /api/v1/admin/images
func (h *Handler) ListRegistryImages(c *gin.Context) {
	h.proxyImageRegistry(c, http.MethodGet, "/v1/images")
}

// POST /api/v1/admin/images — بدنه‌ی JSON عیناً به image-registry فرستاده می‌شود (بهترین
// حدسِ ما از فیلدها: name/tag/service_type/description/is_active).
func (h *Handler) CreateRegistryImage(c *gin.Context) {
	h.proxyImageRegistry(c, http.MethodPost, "/v1/images")
}

// POST /api/v1/admin/images/:id/file — آپلودِ multipart فایلِ ایمیج (خروجیِ docker save)؛
// بدنه‌ی درخواست (شاملِ multipart boundary) عیناً proxy می‌شود.
func (h *Handler) UploadRegistryImageFile(c *gin.Context) {
	h.proxyImageRegistry(c, http.MethodPost, "/v1/images/"+c.Param("id")+"/file")
}

// PATCH /api/v1/admin/images/:id
func (h *Handler) UpdateRegistryImage(c *gin.Context) {
	h.proxyImageRegistry(c, http.MethodPatch, "/v1/images/"+c.Param("id"))
}

// DELETE /api/v1/admin/images/:id
func (h *Handler) DeleteRegistryImage(c *gin.Context) {
	h.proxyImageRegistry(c, http.MethodDelete, "/v1/images/"+c.Param("id"))
}

// GET /api/v1/admin/images/check?name=&tag= — همان GET /v1/check که agentmanager صدا
// می‌زند؛ برای این‌که ادمین بتواند از داخل پنل هم وضعیت یک ایمیج را چک کند.
func (h *Handler) CheckRegistryImage(c *gin.Context) {
	h.proxyImageRegistry(c, http.MethodGet, "/v1/check?"+c.Request.URL.RawQuery)
}

// ── AllowedCaller management (رفعِ باگِ NEEDS.md بخش ۰) ──────
//
// /v1/images* روی image-registry با IPِ فراخوان (نه X-Admin-Key) تصمیم می‌گیرد — یعنی تا
// وقتی IP خروجیِ apimanager به‌عنوان یک AllowedCaller با CanWrite=true ثبت نشود، همه‌ی
// درخواست‌های ثبت/آپلود/حذف ایمیج از این پنل با ۴۰۳ شکست می‌خورند، حتی با X-Admin-Key درست.
// طبق همان بررسی، /v1/callers/* بر خلافِ /v1/images*، واقعاً X-Admin-Key را چک می‌کند — یعنی
// همین «پروکسیِ کور» برای این مسیرها هم درست کار می‌کند. با این‌ها، خودِ ادمین از داخل پنل
// می‌تواند IP سرورِ apimanager را به‌عنوان caller مجاز ثبت کند، بدون نیاز به curl دستی.

// GET /api/v1/admin/image-callers
func (h *Handler) ListRegistryCallers(c *gin.Context) {
	h.proxyImageRegistry(c, http.MethodGet, "/v1/callers")
}

// POST /api/v1/admin/image-callers — بدنه عیناً proxy می‌شود (بهترین حدس از فیلدها طبق
// NEEDS.md: label/cidr/domain/can_write/is_active).
func (h *Handler) CreateRegistryCaller(c *gin.Context) {
	h.proxyImageRegistry(c, http.MethodPost, "/v1/callers")
}

// PATCH /api/v1/admin/image-callers/:id
func (h *Handler) UpdateRegistryCaller(c *gin.Context) {
	h.proxyImageRegistry(c, http.MethodPatch, "/v1/callers/"+c.Param("id"))
}

// DELETE /api/v1/admin/image-callers/:id
func (h *Handler) DeleteRegistryCaller(c *gin.Context) {
	h.proxyImageRegistry(c, http.MethodDelete, "/v1/callers/"+c.Param("id"))
}
