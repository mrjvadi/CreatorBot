package userbot

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/source-service/internal/models"
	"github.com/mrjvadi/creatorbot/source-service/internal/rules"
)

// CreateRulePayload defines a trigger+condition+action rule. See README
// "موتور قانون عمومی" for the full shape of trigger/conditions/action and
// examples. Conditions is optional (nil/empty means the action always
// runs when the trigger fires).
//
// Example: whenever @some_channel posts something containing "urgent", run
// a bot command with the post's text and report the result to botmanager.
//
//	{
//	  "trigger": {"type": "channel_post", "config": {"channel": "@some_channel"}},
//	  "conditions": [{"type": "text_contains", "value": "urgent"}],
//	  "action": {"type": "run_task", "config": {
//	    "task_type": "run_bot_command",
//	    "payload_template": "{\"bot_username\":\"@mybot\",\"command\":\"/process {{.Text | json}}\"}"
//	  }}
//	}
type CreateRulePayload struct {
	Trigger struct {
		Type   string          `json:"type"`
		Config json.RawMessage `json:"config"`
	} `json:"trigger"`
	Conditions []rules.Condition `json:"conditions"`
	Action     struct {
		Type   string          `json:"type"`
		Config json.RawMessage `json:"config"`
	} `json:"action"`
}

type CreateRuleResult struct {
	RuleID string `json:"rule_id"`
}

func (u *Userbot) handleCreateRule(ctx context.Context, _ string, raw json.RawMessage) (any, error) {
	var p CreateRulePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.Trigger.Type == "" || p.Action.Type == "" {
		return nil, errors.New("trigger.type and action.type are required")
	}

	id := uuid.New()
	conditionsJSON, err := json.Marshal(p.Conditions)
	if err != nil {
		return nil, err
	}
	actionJSON, err := json.Marshal(rules.Action{Type: p.Action.Type, Config: p.Action.Config})
	if err != nil {
		return nil, err
	}

	stored := rules.StoredRule{
		ID:          id.String(),
		Phone:       u.phone,
		TriggerType: p.Trigger.Type,
		TriggerRaw:  p.Trigger.Config,
		Conditions:  p.Conditions,
		Action:      rules.Action{Type: p.Action.Type, Config: p.Action.Config},
	}
	if err := u.rules.StartRule(ctx, stored); err != nil {
		return nil, err
	}

	row := &models.Rule{
		Base:        models.Base{ID: id},
		Phone:       u.phone,
		TriggerType: p.Trigger.Type,
		Trigger:     p.Trigger.Config,
		Conditions:  conditionsJSON,
		ActionType:  p.Action.Type,
		Action:      actionJSON,
		Active:      true,
	}
	if err := u.store.CreateRule(ctx, row); err != nil {
		u.rules.StopRule(id.String())
		return nil, err
	}

	return CreateRuleResult{RuleID: id.String()}, nil
}
