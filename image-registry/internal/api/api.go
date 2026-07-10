// Package api — HTTP API سرویس image-registry.
package api

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/image-registry/internal/ipallow"
	"github.com/mrjvadi/creatorbot/image-registry/internal/store"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Handler struct {
	store    *store.Store
	checker  *ipallow.Checker
	log      ports.Logger
	adminKey string // برای bootstrap مدیریت callerها — رجوع به README

	// storageDir محل ذخیره‌ی فایل‌های واقعی image (خروجی `docker save`) روی
	// دیسک این سرویس. agentmanager دیگر image را از یک registry بیرونی
	// pull نمی‌کند؛ آن را مستقیماً از اینجا (GET /v1/images/:id/download)
	// دانلود و `docker load` می‌کند — رجوع README.
	storageDir string

	// maxFileSizeBytes سقف اندازه‌ی فایل قابل‌آپلود. <= 0 یعنی بدون سقف
	// (عمداً — کسی که این را صفر/منفی گذاشته، صریحاً محدودیت را خاموش کرده).
	maxFileSizeBytes int64
}

func New(st *store.Store, checker *ipallow.Checker, log ports.Logger, adminKey, storageDir string, maxFileSizeBytes int64) *Handler {
	return &Handler{
		store: st, checker: checker, log: log, adminKey: adminKey,
		storageDir: storageDir, maxFileSizeBytes: maxFileSizeBytes,
	}
}

func ok(c *gin.Context, data any) { c.JSON(http.StatusOK, gin.H{"ok": true, "data": data}) }
func fail(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(code, gin.H{"ok": false, "message": msg})
}

const ctxKeyCanWrite = "ir:can_write"

// ipGate تنها مکانیزم امنیتی اصلی این سرویس: IP واقعیِ اتصال TCP (نه هدر
// قابل‌جعل) را با جدول AllowedCaller مقایسه می‌کند. Fail-closed: بدون
// مطابقت، ۴۰۳ — دقیقاً همان فلسفه‌ای که در کل این پروژه برای whitelist ها
// استفاده شده (خالی/نامطابق = رد، نه پیش‌فرض باز).
func (h *Handler) ipGate(c *gin.Context) {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		host = c.Request.RemoteAddr
	}
	res, err := h.checker.Check(c.Request.Context(), host)
	if err != nil {
		h.log.Error("ip check failed", ports.F("err", err), ports.F("ip", host))
		fail(c, http.StatusInternalServerError, "internal error")
		return
	}
	if !res.Allowed {
		h.log.Warn("rejected: ip not allow-listed", ports.F("ip", host))
		fail(c, http.StatusForbidden, "your IP is not allow-listed for this service")
		return
	}
	c.Set(ctxKeyCanWrite, res.CanWrite)
	c.Next()
}

// requireWrite علاوه بر ipGate، فقط callerهایی که CanWrite=true دارند را
// رد نمی‌کند — برای ثبت/حذف image و مدیریت خودِ لیست caller ها.
func (h *Handler) requireWrite(c *gin.Context) {
	canWrite, _ := c.Get(ctxKeyCanWrite)
	if b, _ := canWrite.(bool); !b {
		fail(c, http.StatusForbidden, "this IP is read-only on image-registry")
		return
	}
	c.Next()
}

// adminKeyGate برای bootstrap مدیریت caller ها استفاده می‌شود — چون قبل از
// اینکه اولین AllowedCaller ثبت شود، هیچ IP ای CanWrite ندارد (مشکل
// مرغ‌وتخم‌مرغ). یک X-Admin-Key ثابت از env، فقط برای همین بوت‌استرپ.
func (h *Handler) adminKeyGate(c *gin.Context) {
	if h.adminKey == "" || c.GetHeader("X-Admin-Key") != h.adminKey {
		fail(c, http.StatusUnauthorized, "invalid or missing X-Admin-Key")
		return
	}
	c.Next()
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "service": "image-registry"})
	})

	v1 := r.Group("/v1", h.ipGate)
	v1.GET("/check", h.checkImage)
	v1.GET("/images", h.listImages)
	// دانلود فایل واقعی image — read-only (نیازی به CanWrite نیست، همان
	// callerهایی که اجازه‌ی /v1/check دارند باید بتوانند فایل را هم بگیرند).
	v1.GET("/images/:id/download", h.downloadImageFile)
	v1.POST("/images", h.requireWrite, h.createImage)
	// آپلود فایل واقعی image (خروجی `docker save`) روی یک ردیف از قبل ثبت‌شده.
	v1.POST("/images/:id/file", h.requireWrite, h.uploadImageFile)
	v1.PATCH("/images/:id", h.requireWrite, h.setImageActive)
	v1.DELETE("/images/:id", h.requireWrite, h.deleteImage)

	// مدیریت caller ها — پشت adminKey (نه ipGate) تا bootstrap ممکن باشد.
	admin := r.Group("/v1/callers", h.adminKeyGate)
	admin.GET("", h.listCallers)
	admin.POST("", h.createCaller)
	admin.PATCH("/:id", h.setCallerActive)
	admin.DELETE("/:id", h.deleteCaller)
}

// GET /v1/check?name=creatorbot/uploader-bot&tag=v1.4.0
// این endpoint اصلی است که agentmanager قبل از هر deploy صدا می‌زند.
func (h *Handler) checkImage(c *gin.Context) {
	name := c.Query("name")
	tag := c.Query("tag")
	if name == "" || tag == "" {
		fail(c, http.StatusBadRequest, "name and tag are required")
		return
	}
	img, err := h.store.FindImage(c.Request.Context(), name, tag)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if img == nil {
		ok(c, gin.H{"allowed": false})
		return
	}
	resp := gin.H{
		"allowed":      true,
		"service_type": img.ServiceType,
		"description":  img.Description,
		"has_file":     img.HasFile(),
	}
	// اگر فایل واقعی آپلود شده باشد، agentmanager دیگر لازم نیست خودش را با
	// هیچ registry بیرونی درگیر کند — همین‌جا لینک دانلود و چک‌سام را می‌دهیم
	// تا با `docker load` بعد از دانلود مستقیم از این سرویس، ایمیج را بار کند.
	if img.HasFile() {
		resp["download_url"] = "/v1/images/" + img.ID.String() + "/download"
		resp["sha256"] = img.FileSHA256
		resp["size"] = img.FileSize
	}
	ok(c, resp)
}

func (h *Handler) listImages(c *gin.Context) {
	list, err := h.store.ListImages(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, list)
}

type createImageReq struct {
	Name        string `json:"name" binding:"required"`
	Tag         string `json:"tag" binding:"required"`
	ServiceType string `json:"service_type"`
	Description string `json:"description"`
}

func (h *Handler) createImage(c *gin.Context) {
	var req createImageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	img := &store.RegisteredImage{
		Name: req.Name, Tag: req.Tag,
		ServiceType: req.ServiceType, Description: req.Description,
		IsActive: true,
	}
	if err := h.store.CreateImage(c.Request.Context(), img); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, img)
}

func (h *Handler) setImageActive(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.SetImageActive(c.Request.Context(), id, req.IsActive); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"id": id, "is_active": req.IsActive})
}

func (h *Handler) deleteImage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	// اگر فایلی روی دیسک برای این ردیف هست، قبل از حذف رکورد آن را هم پاک
	// می‌کنیم — وگرنه یک فایل یتیم (بدون هیچ رکوردی که به آن اشاره کند) روی
	// دیسک باقی می‌ماند و کسی هرگز آن را جمع نمی‌کند.
	if img, err := h.store.FindImageByID(c.Request.Context(), id); err == nil && img != nil && img.FilePath != "" {
		if rmErr := os.Remove(img.FilePath); rmErr != nil && !os.IsNotExist(rmErr) {
			h.log.Warn("failed to remove image file from disk", ports.F("err", rmErr), ports.F("path", img.FilePath))
		}
	}
	if err := h.store.DeleteImage(c.Request.Context(), id); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"deleted": true})
}

// POST /v1/images/:id/file — multipart/form-data، فیلد "file" باید خروجی
// `docker save <name>:<tag> -o file.tar` باشد. جایگزین می‌کند: اگر قبلاً
// فایلی برای این ردیف بود، رونویسی می‌شود (نسخه‌ی جدید همان name:tag).
func (h *Handler) uploadImageFile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	img, err := h.store.FindImageByID(c.Request.Context(), id)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if img == nil {
		fail(c, http.StatusNotFound, "image not registered — create it via POST /v1/images first")
		return
	}

	fh, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "multipart field \"file\" is required (output of `docker save`)")
		return
	}
	// چک اندازه قبل از نوشتن روی دیسک — تا یک آپلود عمدی/اشتباهِ خیلی بزرگ
	// حتی شروع به پر کردن دیسک هم نکند (رجوع NEEDS.md، این دقیقاً همان گپ
	// «بدون سقف اندازه‌ی آپلود» بود که حالا با MAX_IMAGE_FILE_SIZE_MB بسته شده).
	if h.maxFileSizeBytes > 0 && fh.Size > h.maxFileSizeBytes {
		fail(c, http.StatusRequestEntityTooLarge, fmt.Sprintf(
			"file too large: %d bytes > limit %d bytes (MAX_IMAGE_FILE_SIZE_MB)", fh.Size, h.maxFileSizeBytes))
		return
	}

	if err := os.MkdirAll(h.storageDir, 0o755); err != nil {
		h.log.Error("storage dir", ports.F("err", err))
		fail(c, http.StatusInternalServerError, "storage not available")
		return
	}
	dst := filepath.Join(h.storageDir, id.String()+".tar")
	if err := c.SaveUploadedFile(fh, dst); err != nil {
		h.log.Error("save uploaded image file failed", ports.F("err", err))
		fail(c, http.StatusInternalServerError, "failed to store file")
		return
	}

	// اعتبارسنجی محتوا: باید واقعاً یک آرشیو `docker save` باشد و شامل همان
	// name:tag ثبت‌شده. اگر نه، فایل رد و پاک می‌شود — قبل از این، سرویس فقط
	// چک‌سام می‌گرفت بدون هیچ تضمینی که محتوا اصلاً یک image واقعی است.
	if err := validateDockerSaveTar(dst, img.Name, img.Tag); err != nil {
		_ = os.Remove(dst)
		fail(c, http.StatusBadRequest, "uploaded file rejected: "+err.Error())
		return
	}

	sum, size, err := sha256File(dst)
	if err != nil {
		_ = os.Remove(dst)
		h.log.Error("checksum image file failed", ports.F("err", err))
		fail(c, http.StatusInternalServerError, "failed to checksum stored file")
		return
	}

	if err := h.store.SetImageFile(c.Request.Context(), id, dst, sum, size); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{
		"id":           id,
		"sha256":       sum,
		"size":         size,
		"download_url": "/v1/images/" + id.String() + "/download",
	})
}

// validateDockerSaveTar بدون استخراج کامل فایل، مطمئن می‌شود که آرشیو
// آپلودشده واقعاً خروجی `docker save` است و شامل tag ثبت‌شده‌ی همین ردیف
// می‌شود — نه هر فایل دلخواهی که کسی با نام .tar آپلود کرده. هم tar ساده
// (فرمت معمول `docker save`) و هم tar.gz را قبول می‌کند.
func validateDockerSaveTar(path, name, tag string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	br := bufio.NewReader(f)
	var r io.Reader = br
	if magic, _ := br.Peek(2); len(magic) == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		gz, gzErr := gzip.NewReader(br)
		if gzErr != nil {
			return fmt.Errorf("looks gzip-compressed but failed to decompress: %w", gzErr)
		}
		defer gz.Close()
		r = gz
	}

	wantRef := name + ":" + tag
	tr := tar.NewReader(r)
	foundIndex := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("not a valid tar archive: %w", err)
		}
		switch hdr.Name {
		case "manifest.json":
			foundIndex = true
			var manifest []struct {
				RepoTags []string `json:"RepoTags"`
			}
			if decErr := json.NewDecoder(tr).Decode(&manifest); decErr == nil {
				for _, m := range manifest {
					for _, rt := range m.RepoTags {
						if rt == wantRef {
							return nil
						}
					}
				}
			}
		case "repositories":
			foundIndex = true
			var repos map[string]map[string]string
			if decErr := json.NewDecoder(tr).Decode(&repos); decErr == nil {
				if tags, ok := repos[name]; ok {
					if _, ok := tags[tag]; ok {
						return nil
					}
				}
			}
		}
	}
	if !foundIndex {
		return fmt.Errorf("no manifest.json/repositories entry found — doesn't look like `docker save` output")
	}
	return fmt.Errorf("archive doesn't contain a RepoTag matching %q", wantRef)
}

// GET /v1/images/:id/download — این همان مسیری است که agentmanager به‌جای
// `docker pull` از یک registry بیرونی صدا می‌زند: فایل تام (docker save) را
// مستقیماً از این سرویس می‌گیرد و بعد محلی `docker load` می‌کند.
func (h *Handler) downloadImageFile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	img, err := h.store.FindImageByID(c.Request.Context(), id)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if img == nil || img.FilePath == "" {
		fail(c, http.StatusNotFound, "no file uploaded for this image yet")
		return
	}
	c.Header("X-Image-SHA256", img.FileSHA256)
	c.Header("X-Image-Name-Tag", img.FullRef())
	// c.FileAttachment از http.ServeFile استفاده می‌کند → Range requests را
	// هم پشتیبانی می‌کند (دانلودهای بزرگ/قابل‌ازسرگیری برای فایل‌های چند
	// صدمگابایتی/چندگیگابایتی image رایج است).
	c.FileAttachment(img.FilePath, img.Name+"_"+img.Tag+".tar")
}

// sha256File چک‌سام و اندازه‌ی یک فایل روی دیسک را بدون بارگذاری کل آن در
// حافظه محاسبه می‌کند (استریم) — مهم چون فایل‌های image می‌توانند بزرگ باشند.
func sha256File(path string) (sum string, size int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func (h *Handler) listCallers(c *gin.Context) {
	list, err := h.store.ListCallers(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, list)
}

type createCallerReq struct {
	Label    string `json:"label" binding:"required"`
	CIDR     string `json:"cidr" binding:"required"`
	Domain   string `json:"domain"`
	CanWrite bool   `json:"can_write"`
}

func (h *Handler) createCaller(c *gin.Context) {
	var req createCallerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	caller := &store.AllowedCaller{
		Label: req.Label, CIDR: req.CIDR, Domain: req.Domain,
		CanWrite: req.CanWrite, IsActive: true,
	}
	if err := h.store.CreateCaller(c.Request.Context(), caller); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, caller)
}

func (h *Handler) setCallerActive(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.SetCallerActive(c.Request.Context(), id, req.IsActive); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"id": id, "is_active": req.IsActive})
}

func (h *Handler) deleteCaller(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteCaller(c.Request.Context(), id); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"deleted": true})
}
