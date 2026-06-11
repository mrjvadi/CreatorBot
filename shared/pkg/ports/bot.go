package ports

import "context"

// BotSender is the interface for sending Telegram messages.
// Default implementation: TelebotSender (adapters/telebot).
// Swap to a different Telegram library by implementing this interface.
type BotSender interface {
	// Send sends a text message to a chat.
	Send(ctx context.Context, chatID int64, text string, opts ...SendOption) error

	// SendPhoto sends a photo with optional caption.
	SendPhoto(ctx context.Context, chatID int64, fileID, caption string) error

	// Answer answers an inline query.
	AnswerInlineQuery(ctx context.Context, queryID string, results []InlineResult) error

	// GetChatMember returns the membership status of a user in a chat.
	GetChatMember(ctx context.Context, chatID, userID int64) (MemberStatus, error)
}

// SendOption configures a Send call (parse mode, reply markup, etc.).
type SendOption func(*SendConfig)

// SendConfig holds options for a send call.
type SendConfig struct {
	ParseMode   string // "HTML" | "Markdown" | ""
	ReplyMarkup any    // inline keyboard, reply keyboard, etc.
	Silent      bool
}

func WithHTML() SendOption        { return func(c *SendConfig) { c.ParseMode = "HTML" } }
func WithMarkdown() SendOption    { return func(c *SendConfig) { c.ParseMode = "Markdown" } }
func WithSilent() SendOption      { return func(c *SendConfig) { c.Silent = true } }

// MemberStatus represents a Telegram chat membership status.
type MemberStatus string

const (
	MemberStatusMember        MemberStatus = "member"
	MemberStatusAdministrator MemberStatus = "administrator"
	MemberStatusCreator       MemberStatus = "creator"
	MemberStatusLeft          MemberStatus = "left"
	MemberStatusKicked        MemberStatus = "kicked"
	MemberStatusRestricted    MemberStatus = "restricted"
)

func (s MemberStatus) IsActive() bool {
	return s == MemberStatusMember || s == MemberStatusAdministrator || s == MemberStatusCreator
}

// InlineResult is a single result for an inline query response.
type InlineResult struct {
	ID          string
	Title       string
	Description string
	// For article results
	MessageText string
	// For file results
	FileID   string
	FileType string // document, video, audio, etc.
}
