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

func TestPortDefaultsAndTrims(t *testing.T) {
	t.Setenv("PORT", "")
	if got := Port(); got != "8080" {
		t.Errorf("Port default = %q, want 8080", got)
	}
	t.Setenv("PORT", " 9090 ")
	if got := Port(); got != "9090" {
		t.Errorf("Port = %q, want the trimmed value", got)
	}
	if got := Load().Port; got != "9090" {
		t.Errorf("Load().Port = %q: Load and Port() must agree", got)
	}
}

func TestLoadCopiesTheStampedVersion(t *testing.T) { //nolint:paralleltest // mutates the package-level Version
	if Version != "dev" {
		t.Fatalf("an unstamped build should report dev, got %q", Version)
	}
	if got := Load().Version; got != "dev" {
		t.Errorf("Load().Version = %q, want dev", got)
	}
	old := Version
	Version = "abc1234"
	t.Cleanup(func() { Version = old })
	if got := Load().Version; got != "abc1234" {
		t.Errorf("Load().Version = %q, want the stamped value", got)
	}
}
