package i18n

// کلیدهای ترجمه — به‌صورت ثابت تعریف شده‌اند تا از اشتباه تایپی در زمان کامپایل
// جلوگیری شود. مقدار هر ثابت دقیقاً با کلید متناظر در فایل‌های locales/*.json یکی است.
const (
	// عمومی
	KErrGeneric = "err.generic"
	KCancelled  = "common.cancelled"
	KBack       = "common.back"

	// دکمه‌های کیبورد اصلی
	KBtnWallet   = "btn.wallet"
	KBtnDeposit  = "btn.deposit"
	KBtnWithdraw = "btn.withdraw"
	KBtnTransfer = "btn.transfer"
	KBtnHistory  = "btn.history"
	KBtnHelp     = "btn.help"
	KBtnLanguage = "btn.language"
	KBtnCancel   = "btn.cancel"

	// دکمه‌های inline
	KBtnOpenWallet   = "btn.open_wallet"
	KBtnCheckDeposit = "btn.check_deposit"
	KBtnDeposit2     = "btn.deposit_inline"
	KBtnWithdraw2    = "btn.withdraw_inline"
	KBtnHistory2     = "btn.history_inline"

	// start
	KStart = "start.body"

	// wallet
	KWallet       = "wallet.body"
	KWalletFrozen = "wallet.frozen_line"

	// deposit
	KDepositErr  = "deposit.error"
	KDepositBody = "deposit.body"

	// withdraw
	KWithdrawInsufficient = "withdraw.insufficient"
	KWithdrawAskAddr      = "withdraw.ask_addr"
	KWithdrawBadAddr      = "withdraw.bad_addr"
	KWithdrawAskAmount    = "withdraw.ask_amount"
	KWithdrawBadAmount    = "withdraw.bad_amount"
	KWithdrawSubmitted    = "withdraw.submitted"
	KWithdrawError        = "withdraw.error"

	// history
	KHistoryEmpty = "history.empty"
	KHistoryTitle = "history.title"
	KHistoryLine  = "history.line"

	// برچسب نوع تراکنش
	KTxDeposit   = "tx.deposit"
	KTxWithdraw  = "tx.withdraw"
	KTxCreditAdd = "tx.credit_add"
	KTxPayment   = "tx.payment"
	KTxRefund    = "tx.refund"

	// help
	KHelp = "help.body"

	// transfer
	KTransferInsufficient = "transfer.insufficient"
	KTransferAskID        = "transfer.ask_id"
	KTransferBadID        = "transfer.bad_id"
	KTransferNoRecipient  = "transfer.no_recipient"
	KTransferAskAmount    = "transfer.ask_amount"
	KTransferBadAmount    = "transfer.bad_amount"
	KTransferDone         = "transfer.done"
	KTransferReceived     = "transfer.received"
	KTransferError        = "transfer.error"

	// check deposit (callback)
	KCheckPending     = "check.pending"
	KCheckUnconfirmed = "check.unconfirmed"
	KCheckConfirmed   = "check.confirmed"

	// deposit notification (push)
	KDepositConfirmed = "deposit.confirmed"

	// language
	KLanguageAsk     = "language.ask"
	KLanguageChanged = "language.changed"

	// admin
	KAdminPanel          = "admin.panel"
	KAdminNoWithdraws    = "admin.no_withdraws"
	KAdminWithdrawsTitle = "admin.withdraws_title"
	KAdminWithdrawLine   = "admin.withdraw_line"
	KAdminApproveUsage   = "admin.approve_usage"
	KAdminRejectUsage    = "admin.reject_usage"
	KAdminAddCreditUsage = "admin.addcredit_usage"
	KAdminBadID          = "admin.bad_id"
	KAdminBadParam       = "admin.bad_param"
	KAdminWalletNotFound = "admin.wallet_not_found"
	KAdminError          = "admin.error"
	KAdminApproved       = "admin.approved"
	KAdminRejected       = "admin.rejected"
	KAdminCreditAdded    = "admin.credit_added"
	KAdminCreditDesc     = "admin.credit_desc"

	// پنل ادمین تعاملی — دکمه‌ها
	KBtnAdminStats     = "btn.admin_stats"
	KBtnAdminWithdraws = "btn.admin_withdraws"
	KBtnAdminCredit    = "btn.admin_credit"
	KBtnAdminRefresh   = "btn.admin_refresh"
	KBtnAdminBack      = "btn.admin_back"
	KBtnApprove        = "btn.approve"
	KBtnReject         = "btn.reject"

	// پنل ادمین تعاملی — متن‌ها و جریان‌ها
	KAdminMenu            = "admin.menu"
	KAdminStats           = "admin.stats"
	KAdminWithdrawItem    = "admin.withdraw_item"
	KAdminAskTxHash       = "admin.ask_txhash"
	KAdminAskReason       = "admin.ask_reason"
	KAdminAskCreditUserID = "admin.ask_credit_userid"
	KAdminAskCreditAmount = "admin.ask_credit_amount"
	KAdminNotFoundReq     = "admin.req_not_found"
)
