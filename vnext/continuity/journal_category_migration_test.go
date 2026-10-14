package continuity

import "testing"

func TestJournalMigrationCategoriesPreserveObservedLabels(t *testing.T) {
	for _, category := range []JournalCategory{"pr", "review", "progress"} {
		if err := (JournalContent{Category: category, Text: "observed event"}).Validate(); err != nil {
			t.Errorf("%s: %v", category, err)
		}
	}
	if err := (JournalContent{Category: "task", Text: "retired work"}).Validate(); err == nil {
		t.Fatal("retired task category accepted")
	}
}
