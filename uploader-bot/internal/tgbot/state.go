package tgbot

import (
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/core"
)

// step و userState نام‌های محلیِ (alias) نوع‌های مشترک در پکیج core هستند.
// با alias بودن، همه‌ی کدِ موجود که از step/stepXxx یا userState استفاده می‌کند
// بدون تغییر کامپایل می‌شود، درحالی‌که ماشین حالت واقعاً در core.App زندگی می‌کند.
type step = core.Step
type userState = core.UserState

const (
	stepIdle         step = ""
	stepCodeFiles    step = "code:files"
	stepPassword     step = "password"
	stepSearch       step = "search"
	stepNewFolder    step = "folder:new"
	stepEditCaption  step = "edit:caption"
	stepSetPassword  step = "set:password"
	stepSetLimit     step = "set:limit"
	stepEditSetting  step = "edit:setting"
	stepAddChannel   step = "channel:add"
	stepNewPlan      step = "plan:new"
	stepBroadcast    step = "broadcast"
	stepAddAdmin     step = "admin:add"
	stepSearchUser   step = "search:user"
	stepAddPreview   step = "preview:add"
	stepAddAd        step = "ad:add"
	stepSetLikes     step = "code:likes"
	stepSetDownloads step = "code:downloads"
	stepSetViews     step = "code:views"
	stepSetCover     step = "code:cover"
	stepNewSubfolder step = "folder:sub"
	stepRestore      step = "backup:restore"
	stepAddLock      step = "lock:add"
	stepLockCap      step = "lock:cap"
)
