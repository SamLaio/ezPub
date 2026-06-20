package convert

import "testing"

func TestSubsetCharacters(t *testing.T) {
	got := subsetCharacters("乙 甲\n甲\t乙")
	want := " 乙甲"
	if got != want {
		t.Fatalf("subset characters = %q, want %q", got, want)
	}
}
