package main

import "testing"

func TestParseServeConfig(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		port    int
		wantErr bool
	}{
		{name: "default", port: 3000},
		{name: "custom port", args: []string{"--port", "3001"}, port: 3001},
		{name: "invalid low port", args: []string{"--port", "0"}, wantErr: true},
		{name: "invalid high port", args: []string{"--port", "65536"}, wantErr: true},
		{name: "invalid value", args: []string{"--port", "not-a-port"}, wantErr: true},
		{name: "unexpected argument", args: []string{"extra"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseServeConfig(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseServeConfig() error = %v", err)
			}
			if cfg.port != tt.port {
				t.Fatalf("parseServeConfig() port = %d, want %d", cfg.port, tt.port)
			}
		})
	}
}
