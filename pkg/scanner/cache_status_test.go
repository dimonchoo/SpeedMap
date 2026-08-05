package scanner

import "testing"

func TestParsePluginCacheHeader(t *testing.T) {
	tests := []struct {
		name, value, want string
		ok                bool
	}{
		{"x-lightweight-cache", "yes", "HIT", true},
		{"X-Lightweight-Cache", "YES", "HIT", true},
		{"x-lightweight-cache", "no", "MISS", true},
		{"x-lightweight-cache-status", "hit", "HIT", true},
		{"x-redis-cache", "MISS", "MISS", true},
		{"cf-cache-status", "HIT", "", false},
		{"x-lightweight-cache", "", "", false},
	}

	for _, tc := range tests {
		got, ok := parsePluginCacheHeader(tc.name, tc.value)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("%s=%q: got (%q, %v), want (%q, %v)", tc.name, tc.value, got, ok, tc.want, tc.ok)
		}
	}
}
