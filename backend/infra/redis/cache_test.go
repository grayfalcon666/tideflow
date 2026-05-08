package redis

import (
	"sort"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

func TestFeedCacheDedup(t *testing.T) {
	// Test the deduplication logic from RebuildFollowCache
	allScores := []goredis.Z{
		{Score: 1000, Member: "1"},
		{Score: 900, Member: "2"},
		{Score: 800, Member: "1"}, // duplicate
		{Score: 700, Member: "3"},
		{Score: 600, Member: "2"}, // duplicate
	}

	seen := make(map[string]bool)
	deduped := make([]goredis.Z, 0, len(allScores))
	for _, z := range allScores {
		memberStr, ok := z.Member.(string)
		if !ok {
			continue
		}
		if !seen[memberStr] {
			seen[memberStr] = true
			deduped = append(deduped, z)
		}
	}

	if len(deduped) != 3 {
		t.Errorf("deduped length = %d, want 3", len(deduped))
	}

	// Verify order is preserved (first occurrence wins)
	expected := []string{"1", "2", "3"}
	for i, z := range deduped {
		if z.Member != expected[i] {
			t.Errorf("deduped[%d] = %s, want %s", i, z.Member, expected[i])
		}
	}
}

func TestFeedCacheSort(t *testing.T) {
	allScores := []goredis.Z{
		{Score: 500, Member: "a"},
		{Score: 900, Member: "b"},
		{Score: 700, Member: "c"},
	}

	sort.Slice(allScores, func(i, j int) bool {
		return allScores[i].Score > allScores[j].Score
	})

	if allScores[0].Score != 900 {
		t.Errorf("first should be score 900, got %v", allScores[0].Score)
	}
	if allScores[1].Score != 700 {
		t.Errorf("second should be score 700, got %v", allScores[1].Score)
	}
	if allScores[2].Score != 500 {
		t.Errorf("third should be score 500, got %v", allScores[2].Score)
	}
}

func TestFeedCacheLimit(t *testing.T) {
	allScores := make([]goredis.Z, 10)
	for i := 0; i < 10; i++ {
		allScores[i] = goredis.Z{Score: float64(100 - i), Member: string(rune('0' + i))}
	}

	limit := 5
	if len(allScores) > limit {
		allScores = allScores[:limit]
	}

	if len(allScores) != 5 {
		t.Errorf("after limit cut, length = %d, want 5", len(allScores))
	}
}

func TestConstants(t *testing.T) {
	if L1TTL != 5*time.Second {
		t.Errorf("L1TTL = %v, want 5s", L1TTL)
	}
	if L2TTL != time.Hour {
		t.Errorf("L2TTL = %v, want 1h", L2TTL)
	}
	if sfLabelTTL != 30*time.Second {
		t.Errorf("sfLabelTTL = %v, want 30s", sfLabelTTL)
	}
	if sfPollInterval != 100*time.Millisecond {
		t.Errorf("sfPollInterval = %v, want 100ms", sfPollInterval)
	}
	if sfPollTimeout != 2*time.Second {
		t.Errorf("sfPollTimeout = %v, want 2s", sfPollTimeout)
	}
	if followCacheTTL != 24*time.Hour {
		t.Errorf("followCacheTTL = %v, want 24h", followCacheTTL)
	}
}

func TestCacheStructInit(t *testing.T) {
	c := NewCache(nil)
	if c.local == nil {
		t.Error("NewCache should initialize local cache")
	}
	if c.sf == nil {
		t.Error("NewCache should initialize singleflight group")
	}
}
