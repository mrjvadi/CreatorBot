package rules

import (
	"regexp"
	"strings"
)

// evaluate returns true only if every condition passes (AND semantics). An
// unknown condition type fails closed (the rule does NOT fire) rather than
// silently matching everything, so a typo in a condition type can't turn
// into an unconditional rule.
func evaluate(conditions []Condition, ev Event) bool {
	for _, c := range conditions {
		if !evaluateOne(c, ev) {
			return false
		}
	}
	return true
}

func evaluateOne(c Condition, ev Event) bool {
	switch c.Type {
	case "text_contains":
		return strings.Contains(ev.Text, c.Value)
	case "text_regex":
		re, err := regexp.Compile(c.Value)
		return err == nil && re.MatchString(ev.Text)
	case "sender_is":
		return ev.Sender == c.Value
	default:
		return false
	}
}
