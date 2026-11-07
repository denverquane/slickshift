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

// minimal version of a code's entitlement page, with a redemption form per game and platform
func entitlementDoc(t *testing.T, forms ...[2]string) *goquery.Document {
	html := ""
	for _, f := range forms {
		title, service := f[0], f[1]
		html += `<form action="/code_redemptions">` +
			`<input name="authenticity_token" value="token-` + title + `-` + service + `">` +
			`<input name="archway_code_redemption[check]" value="check-` + title + `-` + service + `">` +
			`<input name="archway_code_redemption[service]" value="` + service + `">` +
			`<input name="archway_code_redemption[title]" value="` + title + `">` +
			`</form>`
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func formCheck(form *goquery.Selection) string {
	check, _ := form.Find("input[name='archway_code_redemption[check]']").Attr("value")
	return check
}

func TestFindRedemptionForm_PrefersPlatform(t *testing.T) {
	doc := entitlementDoc(t, [2]string{"oak", "steam"}, [2]string{BL4Title, "epic"}, [2]string{BL4Title, "steam"})

	form, titles := findRedemptionForm(doc, Steam)

	if form == nil {
		t.Fatal("Should find the Borderlands 4 form")
	}
	if check := formCheck(form); check != "check-oak2-steam" {
		t.Errorf("Should pick the Borderlands 4 form for Steam, got %q", check)
	}
	if len(titles) != 3 {
		t.Errorf("Should list every form's title, got %v", titles)
	}
}

func TestFindRedemptionForm_FallsBackToAnyPlatform(t *testing.T) {
	doc := entitlementDoc(t, [2]string{"oak", "psn"}, [2]string{BL4Title, "epic"})

	form, _ := findRedemptionForm(doc, PSN)

	if form == nil || formCheck(form) != "check-oak2-epic" {
		t.Fatal("Should fall back to the Borderlands 4 form for another platform")
	}
}

func TestFindRedemptionForm_OtherGame(t *testing.T) {
	doc := entitlementDoc(t, [2]string{"oak", "steam"}, [2]string{"oak", "epic"})

	form, titles := findRedemptionForm(doc, Steam)

	if form != nil {
		t.Error("Shouldn't return a form when none are for Borderlands 4")
	}
	if len(titles) != 2 || titles[0] != "oak" {
		t.Errorf("Should list the other game's titles, got %v", titles)
	}
}

// If SHiFT stops wrapping the inputs in a form, the whole response is treated as one
func TestFindRedemptionForm_NoFormElement(t *testing.T) {
	html := `<input name="archway_code_redemption[check]" value="c"><input name="archway_code_redemption[title]" value="oak2">`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}

	form, _ := findRedemptionForm(doc, Steam)

	if form == nil || formCheck(form) != "c" {
		t.Fatal("Should use the whole response when there's no form element")
	}
}

// Missing titles means the page changed, which must be an error (retried later), not "wrong game"
func TestFindRedemptionForm_NoTitles(t *testing.T) {
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(`<form><input name="authenticity_token" value="t"></form>`))

	form, titles := findRedemptionForm(doc, Steam)

	if form != nil || len(titles) != 0 {
		t.Errorf("Should find no form or titles, got %v", titles)
	}
}
