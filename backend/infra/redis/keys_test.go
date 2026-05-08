package redis

import "testing"

func TestKeyConstructors(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"AccountToken", AccountToken(123), "v1:account:123"},
		{"AccountRefreshToken", AccountRefreshToken(123), "v1:account:123:refresh"},
		{"RefreshToUID", RefreshToUID("token-abc"), "v1:refresh:token-abc"},
		{"VideoEntity", VideoEntity(456), "v1:video:entity:456"},
		{"FeedGlobal", FeedGlobal(), "v1:feed:global"},
		{"HotVideo", HotVideo("1m", "202605051200"), "v1:hot:video:1m:202605051200"},
		{"HotMerge", HotMerge("1h", "202605051200"), "v1:hot:merge:1h:202605051200"},
		{"Outbox", Outbox(10), "v1:outbox:10"},
		{"Inbox", Inbox(20), "v1:inbox:20"},
		{"FeedCache", FeedCache(5, "2026-05-05T12:00:00Z", 20), "v1:feed:followcache:5:before:2026-05-05T12:00:00Z:limit:20"},
		{"SFLabel", SFLabel("entity:1"), "v1:sf:entity:1"},
		{"RateLimit", RateLimit("login", "192.168.1.1"), "v1:ratelimit:login:192.168.1.1"},
		{"BigVMark", BigVMark(7), "v1:bigv:mark:7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestMakeRedisKeys(t *testing.T) {
	ids := []uint{1, 2, 3}
	keys := makeRedisKeys(ids, VideoEntity)

	if len(keys) != 3 {
		t.Fatalf("len(keys) = %d, want 3", len(keys))
	}

	expected := []string{"v1:video:entity:1", "v1:video:entity:2", "v1:video:entity:3"}
	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("keys[%d] = %q, want %q", i, k, expected[i])
		}
	}
}

func TestMakeRedisKeysEmpty(t *testing.T) {
	keys := makeRedisKeys([]uint{}, VideoEntity)
	if len(keys) != 0 {
		t.Errorf("empty input should produce empty output, got %d", len(keys))
	}
}
