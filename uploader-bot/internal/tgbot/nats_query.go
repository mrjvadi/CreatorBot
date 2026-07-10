package tgbot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// startQueryNATS اشتراک‌های request/reply برای مدیریتِ محتوا از بیرون (وب پنل apimanager،
// بدون بازکردن خودِ تلگرام) را برقرار می‌کند. بازخورد کاربر ۲۰۲۶-۰۷-۰۵ (رجوع
// apimanager/NEEDS.md، بخش «از uploader-bot»): قبلاً این ربات هیچ responder ای برای این‌کار
// نداشت — این اولین قدم است (فقط Code/Folder، طبق اولویتی که کاربر انتخاب کرد؛ Backup و
// ForceJoinChannel در دورهای بعدی).
//
// الگوی نام‌گذاری subject: "uploader.<چیز>.<عملیات>.<botID>" — هماهنگ با
// uploader.settings.<botID> که از قبل وجود داشت، تا apimanager بتواند دقیقاً همان ربات را
// (نه بقیه‌ی instance های uploader-bot را) هدف بگیرد.
//
// از Client.Respond (که در shared/pkg/adapters/nats از قبل پیاده‌سازی شده) استفاده می‌شود:
// خطا به‌صورت {"error": "..."} پاسخ داده می‌شود، موفقیت هم مقدار برگشتی handler عیناً
// json-marshal می‌شود.
func (h *Handler) startQueryNATS() {
	if h.Eng == nil || h.Eng.Nats == nil {
		return
	}
	botID := h.Eng.BotID

	respond := func(subject string, fn func(data []byte) (any, error)) {
		if err := h.Eng.Nats.Respond(subject, fn); err != nil {
			h.Log.Error("nats respond subscribe failed", ports.F("subject", subject), ports.F("err", err))
		}
	}

	respond(fmt.Sprintf("uploader.codes.list.%d", botID), func(data []byte) (any, error) {
		var req struct {
			FolderID string `json:"folder_id"`
			Page     int    `json:"page"`
			Limit    int    `json:"limit"`
		}
		_ = json.Unmarshal(data, &req)
		if req.Page < 1 {
			req.Page = 1
		}
		if req.Limit <= 0 || req.Limit > 200 {
			req.Limit = 50
		}
		codes, total, err := h.Store.ListCodes(context.Background(), req.FolderID, req.Page, req.Limit)
		if err != nil {
			return nil, err
		}
		if codes == nil {
			codes = []models.Code{}
		}
		return map[string]any{"codes": codes, "total": total, "page": req.Page, "limit": req.Limit}, nil
	})

	respond(fmt.Sprintf("uploader.codes.delete.%d", botID), func(data []byte) (any, error) {
		var req struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &req); err != nil || req.ID == "" {
			return nil, fmt.Errorf("id is required")
		}
		if err := h.Store.DeleteCode(context.Background(), req.ID); err != nil {
			return nil, err
		}
		return map[string]bool{"success": true}, nil
	})

	respond(fmt.Sprintf("uploader.folders.list.%d", botID), func(data []byte) (any, error) {
		var req struct {
			ParentID string `json:"parent_id"`
		}
		_ = json.Unmarshal(data, &req)
		folders, err := h.Store.ListFolders(context.Background(), req.ParentID)
		if err != nil {
			return nil, err
		}
		if folders == nil {
			folders = []models.Folder{}
		}
		return map[string]any{"folders": folders}, nil
	})

	respond(fmt.Sprintf("uploader.folders.create.%d", botID), func(data []byte) (any, error) {
		var f models.Folder
		if err := json.Unmarshal(data, &f); err != nil || f.Name == "" {
			return nil, fmt.Errorf("name is required")
		}
		if err := h.Store.CreateFolder(context.Background(), &f); err != nil {
			return nil, err
		}
		return f, nil
	})

	respond(fmt.Sprintf("uploader.folders.delete.%d", botID), func(data []byte) (any, error) {
		var req struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &req); err != nil || req.ID == "" {
			return nil, fmt.Errorf("id is required")
		}
		if err := h.Store.DeleteFolder(context.Background(), req.ID); err != nil {
			return nil, err
		}
		return map[string]bool{"success": true}, nil
	})

	h.Log.Info("nats content-query listeners started", ports.F("bot_id", botID))
}
