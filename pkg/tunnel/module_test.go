package tunnel

import (
	"testing"
)

func TestNormalizeAutoRelayConfig(t *testing.T) {
	tests := []struct {
		name              string
		cfgMaxCandidates  int
		relayNums         int
		wantMinCandidates int
		wantMaxCandidates int
		wantNumRelays     int
	}{
		{
			// root cause scenario from issue #583: single relay with default MaxCandidates;
			// after the fix minCandidates must be >= 1 and numRelays must be 1
			name:              "single relay – min candidates fixed to 1, numRelays set to 1",
			cfgMaxCandidates:  4,
			relayNums:         1,
			wantMinCandidates: 1,
			wantMaxCandidates: 4,
			wantNumRelays:     1,
		},
		{
			// multiple relays, maxCandidates is large enough
			name:              "multi relay – maxCandidates sufficient, numRelays stays 0",
			cfgMaxCandidates:  4,
			relayNums:         3,
			wantMinCandidates: 1,
			wantMaxCandidates: 4,
			wantNumRelays:     0,
		},
		{
			// multiple relays, maxCandidates is too small and must be boosted to relayNums
			name:              "multi relay – maxCandidates boosted to relayNums",
			cfgMaxCandidates:  2,
			relayNums:         5,
			wantMinCandidates: 1,
			wantMaxCandidates: 5,
			wantNumRelays:     0,
		},
		{
			// cfgMaxCandidates=0 with single relay:
			// maxCandidates should be boosted to relayNums=1, minCandidates=1
			name:              "zero maxCandidates with single relay – both boosted to 1",
			cfgMaxCandidates:  0,
			relayNums:         1,
			wantMinCandidates: 1,
			wantMaxCandidates: 1,
			wantNumRelays:     1,
		},
		{
			// no relay nodes configured; minCandidates=1 must still hold
			name:              "no relay nodes",
			cfgMaxCandidates:  4,
			relayNums:         0,
			wantMinCandidates: 1,
			wantMaxCandidates: 4,
			wantNumRelays:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMin, gotMax, gotNum := normalizeAutoRelayConfig(tt.cfgMaxCandidates, tt.relayNums)
			if gotMin != tt.wantMinCandidates {
				t.Errorf("minCandidates: got %d, want %d", gotMin, tt.wantMinCandidates)
			}
			if gotMax != tt.wantMaxCandidates {
				t.Errorf("maxCandidates: got %d, want %d", gotMax, tt.wantMaxCandidates)
			}
			if gotNum != tt.wantNumRelays {
				t.Errorf("numRelays: got %d, want %d", gotNum, tt.wantNumRelays)
			}
		})
	}
}

func TestBuildAutoRelayOpts(t *testing.T) {
	t.Run("without numRelays", func(t *testing.T) {
		opts := buildAutoRelayOpts(1, 4, 0)
		// should contain 3 options: WithMinCandidates, WithMaxCandidates, WithBackoff
		if len(opts) != 3 {
			t.Errorf("expected 3 options, got %d", len(opts))
		}
	})

	t.Run("with numRelays", func(t *testing.T) {
		opts := buildAutoRelayOpts(1, 1, 1)
		// should contain 4 options: WithMinCandidates, WithMaxCandidates, WithBackoff, WithNumRelays
		if len(opts) != 4 {
			t.Errorf("expected 4 options, got %d", len(opts))
		}
	})
}
