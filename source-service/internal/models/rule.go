package models

// Rule is a persisted trigger+condition+action definition (see
// internal/rules for the engine that runs these). Trigger/Conditions/Action
// are stored as raw JSON so new trigger/condition/action kinds never need a
// schema migration — only internal/rules needs a new case.
type Rule struct {
	Base
	Phone string `gorm:"index;not null"`

	TriggerType string `gorm:"not null"`
	Trigger     []byte `gorm:"type:jsonb;not null"` // shape depends on TriggerType

	Conditions []byte `gorm:"type:jsonb"` // JSON array of {type, value}; may be empty

	ActionType string `gorm:"not null"`
	Action     []byte `gorm:"type:jsonb;not null"` // {type, config} shape depends on ActionType

	Active bool `gorm:"not null;default:true"`
}
