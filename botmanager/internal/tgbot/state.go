package tgbot

import (
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/state"
)

// aliasهای محلی به پکیجِ state تا کدِ tgbot (router/handleStep) بدون تغییر بماند.
// متدهای ذخیره/بازیابی در core هستند؛ این‌ها فقط typeها و ثابت‌ها.
type userState = state.UserState

const (
	stepIdle = state.StepIdle

	stepServerName = state.StepServerName
	stepServerIP   = state.StepServerIP

	stepTmplType  = state.StepTmplType
	stepTmplImage = state.StepTmplImage
	stepTmplTag   = state.StepTmplTag
	stepTmplName  = state.StepTmplName

	stepPlanTmpl   = state.StepPlanTmpl
	stepPlanName   = state.StepPlanName
	stepPlanDays   = state.StepPlanDays
	stepPlanPrice  = state.StepPlanPrice
	stepPlanLimits = state.StepPlanLimits

	stepUserAction      = state.StepUserAction
	stepAdminUserSearch = state.StepAdminUserSearch

	stepWizardToken = state.StepWizardToken
	stepLangSelect  = state.StepLangSelect

	stepAdminCreditAmount = state.StepAdminCreditAmount
	stepWalletTopupAmount = state.StepWalletTopupAmount
	stepBroadcastText     = state.StepBroadcastText
	stepAdminTestToken    = state.StepAdminTestToken

	stepBroadcastForwardWait = state.StepBroadcastForwardWait

	stepSWAppID   = state.StepSWAppID
	stepSWAppHash = state.StepSWAppHash
	stepSWPhone   = state.StepSWPhone
	stepSWLabel   = state.StepSWLabel

	stepPromoRedeem       = state.StepPromoRedeem
	stepPromoAdminCode    = state.StepPromoAdminCode
	stepPromoAdminAmount  = state.StepPromoAdminAmount
	stepPromoAdminMaxUses = state.StepPromoAdminMaxUses
	stepPromoAdminDays    = state.StepPromoAdminDays

	stepWizardConfig   = state.StepWizardConfig
	stepTmplSchemaJSON = state.StepTmplSchemaJSON
)
