package core

import "testing"

func TestPPIOConnectionDefaults(t *testing.T) {
	t.Setenv("PPIO_API_KEY", "test-key")
	t.Setenv("PPIO_DOMAIN", "")
	t.Setenv("PPIO_API_URL", "")
	t.Setenv("PPIO_DEBUG", "")

	cfg, err := NewConnectionConfig(&SandboxOpts{})
	if err != nil {
		t.Fatalf("NewConnectionConfig() error = %v", err)
	}
	if cfg.Domain != DefaultPPIODomain {
		t.Fatalf("Domain = %q, want %q", cfg.Domain, DefaultPPIODomain)
	}
	if cfg.APIURL != "https://api.sandbox.ppio.cn" {
		t.Fatalf("APIURL = %q, want https://api.sandbox.ppio.cn", cfg.APIURL)
	}
	if cfg.Headers["X-PPIO-SDK-Name"] != SDKName {
		t.Fatalf("missing X-PPIO-SDK-Name header: %#v", cfg.Headers)
	}
}
