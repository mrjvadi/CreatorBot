// Package telebot implements ports.BotSender using gopkg.in/telebot.v4.
// To swap to a different Telegram library: implement ports.BotSender and wire in main.go.
package telebot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Sender wraps *tele.Bot and implements ports.BotSender.
type Sender struct {
	bot *tele.Bot
}

var _ ports.BotSender = (*Sender)(nil)

// New wraps an existing *tele.Bot.
func New(bot *tele.Bot) *Sender {
	return &Sender{bot: bot}
}

// Bot returns the underlying *tele.Bot for registering handlers.
func (s *Sender) Bot() *tele.Bot { return s.bot }

func (s *Sender) Send(_ context.Context, chatID int64, text string, opts ...ports.SendOption) error {
	cfg := &ports.SendConfig{}
	for _, o := range opts {
		o(cfg)
	}

	var sendOpts []any
	switch cfg.ParseMode {
	case "HTML":
		sendOpts = append(sendOpts, tele.ModeHTML)
	case "Markdown":
		sendOpts = append(sendOpts, tele.ModeMarkdown)
	}
	if cfg.Silent {
		sendOpts = append(sendOpts, tele.Silent)
	}

	_, err := s.bot.Send(&tele.User{ID: chatID}, text, sendOpts...)
	return err
}

func (s *Sender) SendPhoto(_ context.Context, chatID int64, fileID, caption string) error {
	photo := &tele.Photo{File: tele.File{FileID: fileID}, Caption: caption}
	_, err := s.bot.Send(&tele.User{ID: chatID}, photo)
	return err
}

func (s *Sender) AnswerInlineQuery(_ context.Context, queryID string, results []ports.InlineResult) error {
	var teleResults []tele.Result
	for _, r := range results {
		if r.FileID != "" {
			// File result
			article := &tele.ArticleResult{
				ResultBase:  tele.ResultBase{ID: r.ID},
				Title:       r.Title,
				Description: r.Description,
				Text:        r.MessageText,
			}
			teleResults = append(teleResults, article)
		} else {
			article := &tele.ArticleResult{
				ResultBase:  tele.ResultBase{ID: r.ID},
				Title:       r.Title,
				Description: r.Description,
				Text:        r.MessageText,
			}
			teleResults = append(teleResults, article)
		}
	}
	return s.bot.Answer(&tele.Query{ID: queryID}, &tele.QueryResponse{Results: teleResults})
}

func (s *Sender) GetChatMember(_ context.Context, chatID, userID int64) (ports.MemberStatus, error) {
	chat := &tele.Chat{ID: chatID}
	member, err := s.bot.ChatMemberOf(chat, &tele.User{ID: userID})
	if err != nil {
		return ports.MemberStatusLeft, fmt.Errorf("getChatMember: %w", err)
	}
	return ports.MemberStatus(member.Role), nil
}
