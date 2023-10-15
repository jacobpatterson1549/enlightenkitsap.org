package main

import (
	"io"
	"testing"
)

func TestParseArgsAndEnv(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		env    [][]string
		wantOk bool
		want   config
	}{
		{
			name:   "defaults",
			wantOk: true,
			want: config{
				port: "8000",
			},
		},
		{
			name: "all args",
			args: []string{
				"-port=1",
			},
			wantOk: true,
			want: config{
				port: "1",
			},
		},
		{
			name: "all env",
			args: []string{
				"-port=999", // environment wins
			},
			env: [][]string{
				{"PORT", "11"},
			},
			wantOk: true,
			want: config{
				port: "11",
			},
		},
	}
	t.Run("no program name", func(t *testing.T) {
		cfg := new(config)
		if err := cfg.parseArgsAndEnv(io.Discard); err == nil {
			t.Errorf("wanted error parsing args without program name")
		}
	})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := new(config)
			for _, e := range test.env {
				k, v := e[0], e[1]
				t.Setenv(k, v)
			}
			args := append([]string{"name"}, test.args...)
			err := cfg.parseArgsAndEnv(io.Discard, args...)
			got := *cfg
			switch {
			case !test.wantOk:
				if err == nil {
					t.Errorf("wanted error")
				}
			case err != nil:
				t.Errorf("unwanted error: %v", err)
			case test.want != got:
				t.Errorf("not equal: \n wanted: %#v \n got:    %#v",
					test.want, got)
			}
		})
	}
}
