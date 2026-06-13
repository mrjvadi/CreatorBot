package configstore

import (
	"context"
	"encoding/json"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

const SubjectConfigUpdated = "config.updated"

// ConfigUpdatedEvent رویداد config.updated در NATS.
type ConfigUpdatedEvent struct {
	BotID string `json:"bot_id"`
	Type  string `json:"type"`
}

// RegisterNATSHandler برای config.updated subscribe می‌کند.
func RegisterNATSHandler(store *Store, nc *natsclient.Client, log ports.Logger) {
	nc.Subscribe(SubjectConfigUpdated, func(data []byte) {
		var event ConfigUpdatedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return
		}
		if event.BotID != store.botID {
			return
		}
		ctx := context.Background()
		if err := store.Reload(ctx); err != nil {
			log.Error("config.updated: reload failed",
				ports.F("bot_id", store.botID),
				ports.F("err", err))
			return
		}
		log.Info("config reloaded",
			ports.F("bot_id", store.botID))
	})
}

// PublishConfigUpdated رویداد config.updated را publish می‌کند.
func PublishConfigUpdated(nc *natsclient.Client, botID, botType string) error {
	return nc.PublishCore(SubjectConfigUpdated, ConfigUpdatedEvent{
		BotID: botID,
		Type:  botType,
	})
}
