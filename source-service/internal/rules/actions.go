package rules

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/source-service/internal/task"
)

type forwardConfig struct {
	Dest string `json:"dest"`
}

type sendTextConfig struct {
	Dest     string `json:"dest"`
	Template string `json:"template"`
}

type runTaskConfig struct {
	TaskType        string `json:"task_type"`
	PayloadTemplate string `json:"payload_template"`
}

func (e *Engine) executeAction(ctx context.Context, a Action, ev Event) error {
	switch a.Type {
	case "forward":
		return e.actionForward(ctx, a.Config, ev)
	case "send_text":
		return e.actionSendText(ctx, a.Config, ev)
	case "run_task":
		return e.actionRunTask(ctx, a.Config, ev)
	default:
		return fmt.Errorf("unknown action type %q", a.Type)
	}
}

// actionForward only makes sense for a channel_post-triggered event — there
// is no Telegram message to forward from a nats_message trigger.
func (e *Engine) actionForward(ctx context.Context, raw json.RawMessage, ev Event) error {
	var cfg forwardConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}
	if ev.SourceChannel == "" || ev.MessageID == 0 {
		return fmt.Errorf("forward action needs a channel_post-triggered event")
	}
	_, err := e.tg.ForwardMessage(ctx, ev.SourceChannel, cfg.Dest, ev.MessageID)
	return err
}

func (e *Engine) actionSendText(ctx context.Context, raw json.RawMessage, ev Event) error {
	var cfg sendTextConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}
	text, err := renderTemplate(cfg.Template, ev)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}
	_, err = e.tg.SendText(ctx, cfg.Dest, string(text))
	return err
}

// actionRunTask is the composability seam: it re-dispatches into the same
// task.Registry every other capability is registered on, using a payload
// built by rendering PayloadTemplate (the whole JSON blob, as text) against
// the triggering event. Any task type — present or future — is usable here
// with no engine changes.
func (e *Engine) actionRunTask(ctx context.Context, raw json.RawMessage, ev Event) error {
	var cfg runTaskConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}

	payload, err := renderTemplate(cfg.PayloadTemplate, ev)
	if err != nil {
		return fmt.Errorf("render payload template: %w", err)
	}

	envelope, err := json.Marshal(task.Envelope{ID: uuid.New().String(), Type: cfg.TaskType, Payload: payload})
	if err != nil {
		return err
	}

	resultData, _ := e.tasks.Dispatch(ctx, envelope)
	var result task.Result
	if err := json.Unmarshal(resultData, &result); err == nil && !result.OK {
		return fmt.Errorf("run_task %s failed: %s", cfg.TaskType, result.Error)
	}
	return nil
}
