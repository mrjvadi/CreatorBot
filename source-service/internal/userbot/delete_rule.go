package userbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type DeleteRulePayload struct {
	RuleID string `json:"rule_id"`
}

func (u *Userbot) handleDeleteRule(ctx context.Context, _ string, raw json.RawMessage) (any, error) {
	var p DeleteRulePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.RuleID == "" {
		return nil, errors.New("rule_id is required")
	}

	id, err := uuid.Parse(p.RuleID)
	if err != nil {
		return nil, fmt.Errorf("invalid rule_id: %w", err)
	}

	u.rules.StopRule(p.RuleID)
	if err := u.store.DeactivateRule(ctx, id); err != nil {
		return nil, err
	}
	return map[string]bool{"removed": true}, nil
}
