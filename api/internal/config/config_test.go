package config

import "testing"

func TestLoadStaticDir(t *testing.T) {
	t.Setenv("STATIC_DIR", "")
	if got := Load().StaticDir; got != "" {
		t.Errorf("StaticDir default = %q, want empty (API-only)", got)
	}

	t.Setenv("STATIC_DIR", "  /srv/web  ")
	if got := Load().StaticDir; got != "/srv/web" {
		t.Errorf("StaticDir = %q, want the trimmed value", got)
	}
}

func TestLoadDevLoginIsOffUnlessExplicitlyEnabled(t *testing.T) {
	t.Setenv("DEV_LOGIN", "")
	t.Setenv("APP_ENV", "development")
	if Load().DevLoginEnabled {
		t.Error("DEV_LOGIN must default to off")
	}
	t.Setenv("DEV_LOGIN", "true")
	if !Load().DevLoginEnabled {
		t.Error("DEV_LOGIN=true should enable it outside production")
	}
}

func TestEnvBool(t *testing.T) {
	cases := map[string]bool{"1": true, "true": true, "YES": true, " on ": true, "0": false, "false": false, "No": false, "off": false}
	for in, want := range cases {
		t.Setenv("X_FLAG", in)
		if got := envBool("X_FLAG", !want); got != want {
			t.Errorf("envBool(%q) = %v, want %v", in, got, want)
		}
	}
	t.Setenv("X_FLAG", "maybe")
	if !envBool("X_FLAG", true) || envBool("X_FLAG", false) {
		t.Error("unrecognised values must fall back to the default")
	}
}
