package server

import (
	"reflect"
	"testing"

	"github.com/howznguyen/knowns/internal/models"
)

func TestDeriveOpenCodePortCandidates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		browserPort int
		defaultPort int
		want        []int
	}{
		{
			name:        "derives browser based ports",
			browserPort: 6420,
			defaultPort: 4096,
			want:        []int{64200, 64201, 64202},
		},
		{
			name:        "falls back to default range when derived ports overflow",
			browserPort: 7000,
			defaultPort: 4096,
			want:        []int{4096, 4097, 4098},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := deriveOpenCodePortCandidates(tt.browserPort, tt.defaultPort)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("deriveOpenCodePortCandidates(%d, %d) = %v, want %v", tt.browserPort, tt.defaultPort, got, tt.want)
			}
		})
	}
}

func TestResolveOpenCodeConfig(t *testing.T) {
	t.Parallel()

	t.Run("uses explicit configured port as-is", func(t *testing.T) {
		t.Parallel()

		resolution := resolveOpenCodeConfig(6420, &models.OpenCodeServerConfig{
			Host: "127.0.0.1",
			Port: 5001,
		})

		if !resolution.configured {
			t.Fatal("expected configuration to be enabled")
		}
		if !resolution.explicitPort {
			t.Fatal("expected explicitPort to be true")
		}
		if resolution.cfg.Port != 5001 {
			t.Fatalf("expected explicit port 5001, got %d", resolution.cfg.Port)
		}
	})

	t.Run("derives port from browser port when port is unset", func(t *testing.T) {
		t.Parallel()

		resolution := resolveOpenCodeConfig(6420, &models.OpenCodeServerConfig{
			Host: "127.0.0.1",
		})

		if !resolution.configured {
			t.Fatal("expected configuration to be enabled")
		}
		if resolution.explicitPort {
			t.Fatal("expected explicitPort to be false")
		}
		if resolution.cfg.Port != 64200 {
			t.Fatalf("expected derived port 64200, got %d", resolution.cfg.Port)
		}
	})

	t.Run("derives config even when opencodeServer is missing", func(t *testing.T) {
		t.Parallel()

		resolution := resolveOpenCodeConfig(6420, nil)

		if !resolution.configured {
			t.Fatal("expected configuration to be enabled")
		}
		if resolution.explicitPort {
			t.Fatal("expected explicitPort to be false")
		}
		if resolution.cfg.Host != "127.0.0.1" {
			t.Fatalf("expected default host 127.0.0.1, got %s", resolution.cfg.Host)
		}
		if resolution.cfg.Port != 64200 {
			t.Fatalf("expected derived port 64200, got %d", resolution.cfg.Port)
		}
	})
}
