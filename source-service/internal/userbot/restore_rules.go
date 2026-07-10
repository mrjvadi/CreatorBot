package userbot

import (
	"context"
	"encoding/json"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/source-service/internal/rules"
)

// RestoreRules reloads this account's persisted, active rules into the live
// rules.Engine after a restart. Call this once the telegram client is
// ready, same as RestoreWatches/RestoreNatsWatches.
func (u *Userbot) RestoreRules(ctx context.Context) {
	rows, err := u.store.ListActiveRules(ctx, u.phone)
	if err != nil {
		u.log.Error("restore rules: list", ports.F("err", err))
		return
	}

	for _, r := range rows {
		var conditions []rules.Condition
		if len(r.Conditions) > 0 {
			if err := json.Unmarshal(r.Conditions, &conditions); err != nil {
				u.log.Error("restore rule: bad conditions", ports.F("rule_id", r.ID.String()), ports.F("err", err))
				continue
			}
		}

		stored := rules.StoredRule{
			ID:          r.ID.String(),
			Phone:       u.phone,
			TriggerType: r.TriggerType,
			TriggerRaw:  r.Trigger,
			Conditions:  conditions,
			Action:      rules.Action{Type: r.ActionType, Config: r.Action},
		}

		if err := u.rules.StartRule(ctx, stored); err != nil {
			u.log.Error("restore rule", ports.F("rule_id", r.ID.String()), ports.F("err", err))
			continue
		}
		u.log.Info("rule restored", ports.F("rule_id", r.ID.String()), ports.F("trigger_type", r.TriggerType))
	}
}
