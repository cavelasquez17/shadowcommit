package handlers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/cavelasquez17/shadowcommit/api/internal/config"
	"github.com/cavelasquez17/shadowcommit/api/internal/models"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
)

// TestDevLoginRefusesToImpersonateGitHubUser guards the account-takeover fix: dev-login
// must never authenticate into a username that belongs to a real GitHub-linked account.
func TestDevLoginRefusesToImpersonateGitHubUser(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, nil)

	ghID := int64(42)
	victim := models.User{
		PublicID: store.NewPublicID("usr"), Username: "realdev", GitHubID: &ghID,
		Seniority: models.SenioritySenior, CauseOfDeath: models.CauseStillAlive,
		IsEmployed: true, Role: "member",
	}
	if err := db.Create(&victim).Error; err != nil {
		t.Fatalf("seed victim: %v", err)
	}

	resp := postJSON(t, jsonClient(t), ts.URL+"/api/auth/dev-login", map[string]string{"username": "realdev"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403 impersonating a GitHub-linked user, got %d", resp.StatusCode)
	}
}

// TestDevLoginCreatesNewUserAndNeverLeaksToken guards two things: a fresh username still
// works, and the JWT is never echoed in the JSON body (it must live only in the cookie).
func TestDevLoginCreatesNewUserAndNeverLeaksToken(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil)

	resp := postJSON(t, jsonClient(t), ts.URL+"/api/auth/dev-login", map[string]string{"username": "freshdev"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	body := decodeJSON[map[string]any](t, resp)
	if _, leaked := body["token"]; leaked {
		t.Fatal("dev-login response leaked the raw session token in the JSON body")
	}
	if isNew, _ := body["is_new"].(bool); !isNew {
		t.Fatal("want is_new=true for a brand-new username")
	}
}

// TestDevLoginReenteringOwnDevAccountStillWorks guards that the takeover fix did not
// break the feature's actual purpose: switching back into a dev-created persona.
func TestDevLoginReenteringOwnDevAccountStillWorks(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, nil)
	client := jsonClient(t)

	first := postJSON(t, client, ts.URL+"/api/auth/dev-login", map[string]string{"username": "returning"})
	if first.StatusCode != http.StatusOK {
		t.Fatalf("first login: want 200, got %d", first.StatusCode)
	}
	first.Body.Close()

	second := postJSON(t, client, ts.URL+"/api/auth/dev-login", map[string]string{"username": "returning"})
	defer second.Body.Close()
	if second.StatusCode != http.StatusOK {
		t.Fatalf("second login as same dev persona: want 200, got %d", second.StatusCode)
	}
	body := decodeJSON[map[string]any](t, second)
	if isNew, _ := body["is_new"].(bool); isNew {
		t.Fatal("want is_new=false when re-entering an existing dev-created persona")
	}
}

// TestDevLoginRequiresInviteWhenInviteOnly guards INVITE_ONLY actually gating signup:
// previously the flag was only surfaced in /api/meta and never enforced.
func TestDevLoginRequiresInviteWhenInviteOnly(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, func(c *config.Config) { c.InviteOnly = true })

	resp := postJSON(t, jsonClient(t), ts.URL+"/api/auth/dev-login", map[string]string{"username": "gatecrasher"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403 without an invite code when INVITE_ONLY is set, got %d", resp.StatusCode)
	}

	var count int64
	db.Model(&models.User{}).Where("username = ?", "gatecrasher").Count(&count)
	if count != 0 {
		t.Fatal("user must not be created when invite validation fails")
	}
}

// TestDevLoginWithValidInviteConsumesIt guards the invite is atomically marked used
// alongside account creation.
func TestDevLoginWithValidInviteConsumesIt(t *testing.T) {
	t.Parallel()
	ts, db := newTestServer(t, func(c *config.Config) { c.InviteOnly = true })

	inviter := models.User{PublicID: store.NewPublicID("usr"), Username: "inviter", Role: "member"}
	if err := db.Create(&inviter).Error; err != nil {
		t.Fatalf("seed inviter: %v", err)
	}
	inv := models.Invite{
		Code: "WELCOME1", CreatedByID: inviter.ID, ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := db.Create(&inv).Error; err != nil {
		t.Fatalf("seed invite: %v", err)
	}

	resp := postJSON(t, jsonClient(t), ts.URL+"/api/auth/dev-login",
		map[string]string{"username": "invitee", "invite_code": "WELCOME1"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 with a valid invite, got %d", resp.StatusCode)
	}

	var got models.Invite
	if err := db.Where("code = ?", "WELCOME1").First(&got).Error; err != nil {
		t.Fatalf("reload invite: %v", err)
	}
	if got.UsedByID == nil {
		t.Fatal("invite was not marked used after a successful signup")
	}
}

// TestDevLoginDisabledReturnsForbidden guards the existing DEV_LOGIN gate still works
// after the config default changed.
func TestDevLoginDisabledReturnsForbidden(t *testing.T) {
	t.Parallel()
	ts, _ := newTestServer(t, func(c *config.Config) { c.DevLoginEnabled = false })

	resp := postJSON(t, jsonClient(t), ts.URL+"/api/auth/dev-login", map[string]string{"username": "nope"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403 when DEV_LOGIN is off, got %d", resp.StatusCode)
	}
}
