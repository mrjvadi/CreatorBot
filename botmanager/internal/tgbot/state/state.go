package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Step string

const (
	StepIdle Step = ""
	StepServerName   Step = "server:name"
	StepServerIP     Step = "server:ip"
	StepTmplType     Step = "tmpl:type"
	StepTmplImage    Step = "tmpl:image"
	StepTmplTag      Step = "tmpl:tag"
	StepLinkType     Step = "link:type"
	StepLinkLimit    Step = "link:limit"
	StepLinkLabel    Step = "link:label"
	StepPlanTemplate Step = "plan:template"
	StepPlanName     Step = "plan:name"
	StepPlanDays     Step = "plan:days"
	StepPlanPrice    Step = "plan:price"
	StepWizardToken  Step = "wizard:token"
)

type State struct {
	Step Step              `json:"step"`
	Data map[string]string `json:"data"`
}

const ttl = 10 * time.Minute

func key(uid int64) string { return fmt.Sprintf("bm:state:%d", uid) }

func Get(ctx context.Context, cache ports.Cache, uid int64) *State {
	raw, err := cache.Get(ctx, key(uid))
	if err != nil || raw == "" {
		return &State{Step: StepIdle, Data: map[string]string{}}
	}
	var s State
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return &State{Step: StepIdle, Data: map[string]string{}}
	}
	if s.Data == nil {
		s.Data = map[string]string{}
	}
	return &s
}

func Set(ctx context.Context, cache ports.Cache, uid int64, s *State) {
	data, _ := json.Marshal(s)
	cache.Set(ctx, key(uid), string(data), ttl)
}

func Clear(ctx context.Context, cache ports.Cache, uid int64) {
	cache.Del(ctx, key(uid))
}

func SetStep(ctx context.Context, cache ports.Cache, uid int64, step Step, kv ...string) {
	s := Get(ctx, cache, uid)
	s.Step = step
	for i := 0; i+1 < len(kv); i += 2 {
		s.Data[kv[i]] = kv[i+1]
	}
	Set(ctx, cache, uid, s)
}
