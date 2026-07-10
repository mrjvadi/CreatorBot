package telegram

// NormalizePhone strips everything but digits, so the same phone number —
// however it's formatted ("+98 912 000 0000" vs "989120000000") — always
// maps to the same session key, in both the file-storage fallback and the
// database-backed storage.
func NormalizePhone(phone string) string {
	digits := make([]byte, 0, len(phone))
	for i := 0; i < len(phone); i++ {
		if phone[i] >= '0' && phone[i] <= '9' {
			digits = append(digits, phone[i])
		}
	}
	return string(digits)
}

// SessionFileName is only used by the file-storage fallback (cmd/login
// --file). The real worker (cmd/service) always uses DBSessionStorage so a
// lost Docker volume doesn't force a fresh Telegram login.
func SessionFileName(phone string) string {
	return NormalizePhone(phone) + ".json"
}
