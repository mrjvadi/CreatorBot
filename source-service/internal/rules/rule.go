// Package rules is the generic trigger+condition+action engine: instead of
// one hand-written Go task per combination (like watch_channel/watch_nats),
// a Rule is data — a trigger, optional conditions, and an action — created
// at runtime via the create_rule task (internal/userbot), with no new code
// or redeploy needed for a new combination.
//
// The "run_task" action type is what makes this compose with everything
// else in the worker: it re-dispatches into the same task.Registry every
// other task type is registered on, so any existing (or future) task
// automatically becomes usable as a rule's action, with zero extra glue.
//
// Templates (used by send_text and run_task) are plain Go text/template,
// deliberately not a scripting language — a template can only read Event
// fields, it can't call functions or reach outside data. That keeps rules
// safe to store and run as plain data instead of arbitrary code.
package rules

import "encoding/json"

// Condition is one predicate a rule's action must pass before running.
// Multiple conditions are ANDed together.
type Condition struct {
	Type  string `json:"type"` // "text_contains" | "text_regex" | "sender_is"
	Value string `json:"value"`
}

// Action is what runs when a rule's trigger fires and its conditions pass.
type Action struct {
	Type   string          `json:"type"` // "forward" | "send_text" | "run_task"
	Config json.RawMessage `json:"config"`
}

// Event is what a trigger produces and conditions/actions consume — plain
// data, no gotd/td types, so this package stays transport-agnostic.
type Event struct {
	Text          string
	Sender        string
	SourceChannel string
	Subject       string
	MessageID     int
}

// StoredRule is a Rule as loaded from persistence, with its JSON blobs
// already decoded into the typed pieces the engine needs.
type StoredRule struct {
	ID          string
	Phone       string
	TriggerType string
	TriggerRaw  json.RawMessage
	Conditions  []Condition
	Action      Action
}

// ChannelPostTrigger is TriggerRaw's shape when TriggerType == "channel_post".
type ChannelPostTrigger struct {
	Channel string `json:"channel"`
}

// NatsMessageTrigger is TriggerRaw's shape when TriggerType == "nats_message".
type NatsMessageTrigger struct {
	Subject string `json:"subject"`
}
