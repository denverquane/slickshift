package shift

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// minimal version of the SHiFT rewards page, containing just the structure parseRewards depends on
const rewardsHTML = `
<div class="tab-pane well" id="steam">
  <div class="sh_reward_list">
    <div class="shift-secondary-title"><h2>Borderlands 4</h2></div>
    <dl>
      <dt>Golden Key for Borderlands 4</dt>
      <dd><span class="reward_unlocked">Unlocked Sep 28, 2026</span> Use it to open the Golden Chest</dd>
    </dl>
    <dl>
      <dt>Rafa Vault Hunter Skin</dt>
      <dd><span class="reward_unlocked">Unlocked Sep 20, 2026</span> A new look for Rafa</dd>
    </dl>
    <div class="shift-secondary-title"><h2>Tiny Tina's Wonderlands</h2></div>
    <dl>
      <dt>Skeleton Key</dt>
      <dd><span class="reward_unlocked">Unlocked Mar 01, 2022</span> Use it to open the Loot Chest</dd>
    </dl>
  </div>
</div>
<div class="tab-pane well" id="epic">
  <div class="sh_reward_list"></div>
</div>
`

func newRewardsDoc(t *testing.T) *goquery.Document {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rewardsHTML))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestParseRewards_Typical(t *testing.T) {
	rewards := parseRewards(newRewardsDoc(t), Steam, Borderlands4, -1)

	if len(rewards) != 2 {
		t.Fatalf("Should only parse the 2 Borderlands 4 rewards, got %d: %+v", len(rewards), rewards)
	}
	if rewards[0].Title != GoldenKey {
		t.Errorf("First reward title should be %q, got %q", GoldenKey, rewards[0].Title)
	}
	if rewards[0].Date != "Unlocked Sep 28, 2026" {
		t.Errorf("First reward date mismatch: %q", rewards[0].Date)
	}
	if rewards[0].Description != "Use it to open the Golden Chest" {
		t.Errorf("First reward description mismatch: %q", rewards[0].Description)
	}
	if rewards[1].Title != "Rafa Vault Hunter Skin" {
		t.Errorf("Second reward title mismatch: %q", rewards[1].Title)
	}
}

// Regression: returning from the Each callback didn't stop iterating, so the limit was ignored
func TestParseRewards_Limit(t *testing.T) {
	rewards := parseRewards(newRewardsDoc(t), Steam, Borderlands4, 1)

	if len(rewards) != 1 {
		t.Fatalf("Should return 1 reward with limit 1, got %d: %+v", len(rewards), rewards)
	}
	if rewards[0].Title != GoldenKey {
		t.Errorf("Reward should be the most recent (%q), got %q", GoldenKey, rewards[0].Title)
	}
}

// Callers must handle no rewards being listed; indexing into this result is what crashed the bot
func TestParseRewards_NoRewardsListed(t *testing.T) {
	for _, platform := range []Platform{Epic, XboxLive} {
		rewards := parseRewards(newRewardsDoc(t), platform, Borderlands4, 1)
		if len(rewards) != 0 {
			t.Errorf("Platform %s should have no rewards, got %+v", platform, rewards)
		}
	}
}
