package userbot

import (
	"context"
	"encoding/json"
)

type RuleInfo struct {
	RuleID      string `json:"rule_id"`
	TriggerType string `json:"trigger_type"`
	ActionType  string `json:"action_type"`
}

type ListRulesResult struct {
	Rules []RuleInfo `json:"rules"`
}

func (u *Userbot) handleListRules(ctx context.Context, _ string, _ json.RawMessage) (any, error) {
	rows, err := u.store.ListActiveRules(ctx, u.phone)
	if err != nil {
		return nil, err
	}

	out := make([]RuleInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, RuleInfo{RuleID: r.ID.String(), TriggerType: r.TriggerType, ActionType: r.ActionType})
	}
	return ListRulesResult{Rules: out}, nil
}
