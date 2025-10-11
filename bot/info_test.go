package bot

import (
	"strings"
	"testing"
)

func TestSortedEmbeds_Typical(t *testing.T) {
	// arrange
	stats := map[string]int64{
		"total":     100,
		"bad":       20,
		"good_json": 10,
		"other":     0,
	}
	const expectedTotalValue = "100"
	const expectedBadValue = "20 (20%)"
	const expectedGoodValue = "10 (10%)"
	const expectedOtherValue = "0"

	// act
	embeds := toSortedEmbeds(stats)

	// assert
	if len(embeds) != 4 {
		t.Errorf("SortedEmbeds should have 4 embeds, got %d", len(embeds))
	}
	if embeds[0].Name != "Total" {
		t.Errorf("First embed name should be 'Total', got %s", embeds[0].Name)
	}
	if embeds[0].Value != expectedTotalValue {
		t.Errorf("Embed 'Total' value should be %s, got %s", expectedTotalValue, embeds[0].Value)
	}
	if embeds[0].Inline {
		t.Errorf("Embed 'Total' inline should be false")
	}

	if embeds[1].Name != "Bad" {
		t.Errorf("Second embed name should be 'Bad', got %s", embeds[1].Name)
	}
	if embeds[1].Value != expectedBadValue {
		t.Errorf("Embed 'Bad' value should be %s, got %s", expectedBadValue, embeds[1].Value)
	}

	if embeds[2].Name != "Good Json" {
		t.Errorf("Third embed name should be 'Good Json', got %s", embeds[2].Name)
	}
	if embeds[2].Value != expectedGoodValue {
		t.Errorf("Embed 'Good Json' value should be %s, got %s", expectedGoodValue, embeds[2].Value)
	}

	if embeds[3].Value != expectedOtherValue {
		t.Errorf("Embed 'Other' value should be %s, got %s", expectedOtherValue, embeds[3].Value)
	}
}

func TestSortedEmbeds_OneStatRepresentsAll(t *testing.T) {
	// arrange
	stats := map[string]int64{
		"total": 100,
		"bad":   100,
		"other": 0,
	}

	// act
	embeds := toSortedEmbeds(stats)

	// assert
	if strings.Contains(embeds[1].Value, "%") {
		t.Errorf("embed that comprises 100%% of the total should not report the percentage value")
	}
}
