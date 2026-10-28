package textx

import "testing"

func TestCountsWords(t *testing.T) {
	// 对应 Rust 的 assert_eq!(word_count("  hello   world  "), 2);
	if got := WordCount("  hello   world  "); got != 2 {
		t.Fatalf("WordCount(%q) = %d, want 2", "  hello   world  ", got)
	}
}

func TestEmptyText(t *testing.T) {
	if got := WordCount(""); got != 0 {
		t.Fatalf("WordCount(%q) = %d, want 0", "", got)
	}
}

func TestShoutWorks(t *testing.T) {
	if got := Shout("ok"); got != "OK" {
		t.Fatalf("Shout(%q) = %q, want %q", "ok", got, "OK")
	}
}
