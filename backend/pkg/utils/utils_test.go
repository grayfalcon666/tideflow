package utils

import (
	"strings"
	"testing"
)

func TestParseCursor(t *testing.T) {
	tests := []struct {
		name    string
		cursor  string
		wantOk  bool
		wantVal int64
	}{
		{"empty", "", false, 0},
		{"valid number", "1715000000000", true, 1715000000000},
		{"zero", "0", true, 0},
		{"leading zeros", "00123", true, 123},
		{"non-numeric", "abc", false, 0},
		{"mixed", "123abc", false, 0},
		{"negative not supported", "-1", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := ParseCursor(tt.cursor)
			if ok != tt.wantOk {
				t.Errorf("ParseCursor(%q) ok=%v, want %v", tt.cursor, ok, tt.wantOk)
			}
			if ok && val != tt.wantVal {
				t.Errorf("ParseCursor(%q) val=%d, want %d", tt.cursor, val, tt.wantVal)
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	id1 := GenerateID()
	id2 := GenerateID()

	// Should be non-empty
	if id1 == "" {
		t.Error("GenerateID() returned empty string")
	}

	// Should be unique
	if id1 == id2 {
		t.Error("GenerateID() generated duplicate IDs")
	}

	// Should start with timestamp format (14 digits)
	if len(id1) < 14 {
		t.Errorf("GenerateID() too short: %s", id1)
	}
	prefix := id1[:14]
	if _, err := ParseCursor(prefix); !err {
		t.Errorf("GenerateID() prefix %q not parseable as cursor", prefix)
	}

	// Should have random suffix
	suffix := id1[14:]
	if len(suffix) != 8 {
		t.Errorf("GenerateID() suffix length = %d, want 8", len(suffix))
	}
}

func TestParseCursorInt64(t *testing.T) {
	var v int64
	ok, err := parseCursorInt64("", &v)
	if ok || err == nil {
		t.Errorf("parseCursorInt64 empty: ok=%v err=%v, want false non-nil", ok, err)
	}

	ok, err = parseCursorInt64("12345", &v)
	if !ok || err != nil {
		t.Errorf("parseCursorInt64 12345: ok=%v err=%v, want true nil", ok, err)
	}
	if v != 12345 {
		t.Errorf("parseCursorInt64 12345: v=%d, want 12345", v)
	}

	ok, err = parseCursorInt64("12abc", &v)
	if ok {
		t.Error("parseCursorInt64 12abc should return ok=false")
	}
}

func TestRandomStringLength(t *testing.T) {
	for n := 1; n <= 16; n++ {
		s := randomString(n)
		if len(s) != n {
			t.Errorf("randomString(%d) length=%d", n, len(s))
		}
	}
}

func TestRandomStringCharacterSet(t *testing.T) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	s := randomString(200)
	for _, c := range s {
		if !strings.Contains(letters, string(c)) {
			t.Errorf("random char %c not in expected set", c)
			break
		}
	}
}
