package config

import "testing"

func TestOptionalLeboncoinSessionConfiguration(t *testing.T) {
	for _, value := range []string{"", "   ", "  /private/session.json  "} {
		t.Setenv("LEBONCOIN_SESSION_FILE", value)
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		want := ""
		if value == "  /private/session.json  " {
			want = "/private/session.json"
		}
		if cfg.LeboncoinSessionFile != want {
			t.Fatalf("path=%q want=%q", cfg.LeboncoinSessionFile, want)
		}
	}
}
