package main

import "testing"

func TestLogoPixels(t *testing.T) {
	narrow, wide := logoPixels(false), logoPixels(true)
	if len(narrow) != 16 || len(wide) != 16 {
		t.Fatalf("logo should be 16 pixel rows, got %d and %d", len(narrow), len(wide))
	}
	w := 0
	for _, row := range wide {
		w = max(w, len(row))
	}
	if w <= len(chairArt[0]) || w > 76 {
		t.Errorf("wide logo is %d px; want wider than the chair and at most 76 to fit 80 columns", w)
	}
	for i, row := range chairArt {
		if len(row) != len(chairArt[0]) {
			t.Errorf("chair row %d is ragged", i)
		}
	}
}
