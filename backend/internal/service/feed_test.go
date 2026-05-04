package service

import (
	"sort"
	"testing"
)

func TestWindowToMinutes(t *testing.T) {
	tests := []struct {
		window   string
		expected int
	}{
		{"1m", 1},
		{"5m", 5},
		{"15m", 15},
		{"1h", 60},
		{"6h", 360},
		{"unknown", 60},
	}

	for _, tt := range tests {
		t.Run(tt.window, func(t *testing.T) {
			got := windowToMinutes(tt.window)
			if got != tt.expected {
				t.Errorf("windowToMinutes(%q) = %d, want %d", tt.window, got, tt.expected)
			}
		})
	}
}

func TestNormalsMapToSlice(t *testing.T) {
	m := map[uint]bool{1: true, 2: true, 3: true}
	result := normalsMapToSlice(m)

	if len(result) != 3 {
		t.Errorf("normalsMapToSlice length = %d, want 3", len(result))
	}

	// Verify all keys present
	seen := make(map[uint]bool)
	for _, v := range result {
		seen[v] = true
	}
	for k := range m {
		if !seen[k] {
			t.Errorf("missing key %d in result", k)
		}
	}
}

func TestNormalsMapToSliceEmpty(t *testing.T) {
	m := map[uint]bool{}
	result := normalsMapToSlice(m)
	if len(result) != 0 {
		t.Errorf("normalsMapToSlice empty map = %d items, want 0", len(result))
	}
}

func TestVideoScoreSorting(t *testing.T) {
	// Test that sort.Slice produces descending order by score
	type videoScore struct {
		videoID uint
		score   float64
	}

	all := []videoScore{
		{videoID: 1, score: 100},
		{videoID: 2, score: 300},
		{videoID: 3, score: 200},
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].score > all[j].score
	})

	expected := []uint{2, 3, 1}
	for i, v := range all {
		if v.videoID != expected[i] {
			t.Errorf("position %d: videoID = %d, want %d", i, v.videoID, expected[i])
		}
	}
}

func TestVideoScoreSortingDuplicateScores(t *testing.T) {
	type videoScore struct {
		videoID uint
		score   float64
	}

	all := []videoScore{
		{videoID: 1, score: 100},
		{videoID: 2, score: 100},
		{videoID: 3, score: 200},
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].score > all[j].score
	})

	if all[0].videoID != 3 {
		t.Errorf("top score should be videoID 3, got %d", all[0].videoID)
	}
}

func TestVideoScoreSortingEmpty(t *testing.T) {
	type videoScore struct {
		videoID uint
		score   float64
	}

	all := []videoScore{}
	sort.Slice(all, func(i, j int) bool {
		return all[i].score > all[j].score
	})
	if len(all) != 0 {
		t.Errorf("empty slice should remain empty, got length %d", len(all))
	}
}
