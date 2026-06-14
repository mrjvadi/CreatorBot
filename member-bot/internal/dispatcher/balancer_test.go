package dispatcher

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mrjvadi/creatorbot/member-bot/internal/models"
)

func TestSelectLeastLoaded_Basic(t *testing.T) {
	b := &Balancer{}

	bots := []models.CheckBot{
		{Base: models.Base{ID: uuid.New()}, RateLimit: 20},
		{Base: models.Base{ID: uuid.New()}, RateLimit: 20},
		{Base: models.Base{ID: uuid.New()}, RateLimit: 20},
	}

	loads := map[string]int{
		bots[0].ID.String(): 10,
		bots[1].ID.String(): 2,  // کمترین load
		bots[2].ID.String(): 7,
	}

	selected := b.selectLeastLoaded(bots, loads, 1)
	if len(selected) != 1 {
		t.Fatalf("expected 1, got %d", len(selected))
	}
	if selected[0].String() != bots[1].ID.String() {
		t.Errorf("expected least-loaded bot, got %s", selected[0].String())
	}
}

func TestSelectLeastLoaded_NBotsSelected(t *testing.T) {
	b := &Balancer{}

	bots := []models.CheckBot{
		{Base: models.Base{ID: uuid.New()}},
		{Base: models.Base{ID: uuid.New()}},
		{Base: models.Base{ID: uuid.New()}},
	}
	loads := map[string]int{
		bots[0].ID.String(): 5,
		bots[1].ID.String(): 1,
		bots[2].ID.String(): 3,
	}

	// انتخاب ۲ تا از ۳ تا — باید ۲ تا با کمترین load باشند
	selected := b.selectLeastLoaded(bots, loads, 2)
	if len(selected) != 2 {
		t.Fatalf("expected 2, got %d", len(selected))
	}
}

func TestSelectLeastLoaded_MoreThanBots(t *testing.T) {
	b := &Balancer{}

	bots := []models.CheckBot{
		{Base: models.Base{ID: uuid.New()}},
	}
	loads := map[string]int{bots[0].ID.String(): 0}

	// n بیشتر از تعداد bot → همه انتخاب شوند
	selected := b.selectLeastLoaded(bots, loads, 5)
	if len(selected) != 1 {
		t.Errorf("expected 1 (only available bot), got %d", len(selected))
	}
}

func TestRedundancyCalculation(t *testing.T) {
	// با ۳ bot → redundancy باید min(2, 3)=2 باشه
	bots := 3
	redundancy := minBotsPerChannel
	if bots < redundancy {
		redundancy = bots
	}
	if redundancy != 2 {
		t.Errorf("expected redundancy=2, got %d", redundancy)
	}

	// با ۱ bot → redundancy باید ۱ باشه
	bots = 1
	redundancy = minBotsPerChannel
	if bots < redundancy {
		redundancy = bots
	}
	if redundancy != 1 {
		t.Errorf("expected redundancy=1 with single bot, got %d", redundancy)
	}
}
