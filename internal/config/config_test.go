package config

import "testing"

func TestMaskToken(t *testing.T) {
	if got := MaskToken("short"); got != "***" {
		t.Errorf("short token: got %q want ***", got)
	}
	tok := "ghp_ABCDEFGHIJKLMNOP"
	got := MaskToken(tok)
	if got != tok[:8]+"***"+tok[len(tok)-4:] {
		t.Errorf("MaskToken=%q", got)
	}
	if len(got) >= len(tok) {
		t.Error("masked token should be shorter / redacted")
	}
}

func TestResolveProject(t *testing.T) {
	cfg := &Config{
		DefaultProject: "prod",
		Projects: map[string]*Project{
			"prod":    {Owner: "o", Repo: "r"},
			"sandbox": {Owner: "o", Repo: "sandbox"},
		},
	}
	if p := resolveProject(cfg, ""); p == nil || p.Repo != "r" {
		t.Errorf("default project not resolved: %+v", p)
	}
	if p := resolveProject(cfg, "sandbox"); p == nil || p.Repo != "sandbox" {
		t.Errorf("named project not resolved: %+v", p)
	}
	if p := resolveProject(cfg, "missing"); p != nil {
		t.Errorf("missing project should be nil, got %+v", p)
	}
	if p := resolveProject(nil, ""); p != nil {
		t.Error("nil config should resolve to nil")
	}
}

func TestLoadCredentialsFlagsOverride(t *testing.T) {
	// Flags override config/env/gh entirely, so this is deterministic.
	creds, err := LoadCredentials("TOK", "OWN", "REPO", "")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if creds.Token != "TOK" || creds.Owner != "OWN" || creds.Repo != "REPO" {
		t.Errorf("flags not honored: %+v", creds)
	}
}

func TestLoadCredentialsEnvOverride(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "envtok")
	t.Setenv("GX_OWNER", "envowner")
	t.Setenv("GX_REPO", "envrepo")
	creds, err := LoadCredentials("", "", "", "")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if creds.Token != "envtok" || creds.Owner != "envowner" || creds.Repo != "envrepo" {
		t.Errorf("env not honored: %+v", creds)
	}
}
