package core

import (
	"os"
	"testing"
)

func clearPPIOEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"PPIO_API_KEY", "PPIO_ACCESS_TOKEN", "PPIO_DOMAIN",
		"PPIO_API_URL", "PPIO_SANDBOX_URL", "PPIO_DEBUG",
		"PPIO_REQUEST_TIMEOUT", "PPIO_IS_LEGACY",
	} {
		t.Setenv(key, "")
		_ = os.Unsetenv(key)
	}
}

func TestNewConnectionConfigDefaults(t *testing.T) {
	clearPPIOEnv(t)

	opts := &SandboxOpts{APIKey: "test-key"}
	cfg, err := NewConnectionConfig(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Domain != DefaultPPIODomain {
		t.Errorf("Domain = %q, want %q", cfg.Domain, DefaultPPIODomain)
	}
	if cfg.APIURL != "https://api."+DefaultPPIODomain {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, "https://api."+DefaultPPIODomain)
	}
	if !cfg.IsLegacyDomain {
		t.Error("expected legacy domain")
	}
	if cfg.RequestTimeoutMs != LegacyRequestTimeoutMS {
		t.Errorf("RequestTimeoutMs = %d, want %d", cfg.RequestTimeoutMs, LegacyRequestTimeoutMS)
	}
	if cfg.Headers["X-PPIO-SDK-Name"] != SDKName {
		t.Errorf("X-PPIO-SDK-Name = %q, want %q", cfg.Headers["X-PPIO-SDK-Name"], SDKName)
	}
}

func TestNewConnectionConfigLegacyDomainTimeout(t *testing.T) {
	clearPPIOEnv(t)

	cfg, err := NewConnectionConfig(&SandboxOpts{
		APIKey: "key",
		Domain: "sandbox.ppio.cn",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.IsLegacyDomain {
		t.Error("expected legacy domain to be detected")
	}
	if cfg.RequestTimeoutMs != LegacyRequestTimeoutMS {
		t.Errorf("RequestTimeoutMs = %d, want %d", cfg.RequestTimeoutMs, LegacyRequestTimeoutMS)
	}
}

func TestNewConnectionConfigCustomHeaders(t *testing.T) {
	clearPPIOEnv(t)

	cfg, err := NewConnectionConfig(&SandboxOpts{
		APIKey:  "key",
		Headers: map[string]string{"X-Custom": "value"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Headers["X-Custom"] != "value" {
		t.Error("custom header was not preserved")
	}
	if cfg.Headers["X-PPIO-SDK-Version"] != SDKVersion {
		t.Error("SDK version header missing")
	}
}

func TestGetSandboxURL(t *testing.T) {
	clearPPIOEnv(t)

	cfg, err := NewConnectionConfig(&SandboxOpts{APIKey: "key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := cfg.GetSandboxURL("abc123", "", EnvdPort)
	want := "https://49983-abc123." + DefaultPPIODomain
	if got != want {
		t.Errorf("GetSandboxURL = %q, want %q", got, want)
	}

	// An explicit sandbox URL wins.
	cfg2, err := NewConnectionConfig(&SandboxOpts{
		APIKey:     "key",
		SandboxURL: "https://custom.example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := cfg2.GetSandboxURL("abc123", "", EnvdPort); got != "https://custom.example.com" {
		t.Errorf("GetSandboxURL = %q, want override", got)
	}
}

func TestGetHostDebug(t *testing.T) {
	clearPPIOEnv(t)

	debug := true
	cfg, err := NewConnectionConfig(&SandboxOpts{APIKey: "key", Debug: &debug})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := cfg.GetHost("abc", 8080, ""); got != "localhost:8080" {
		t.Errorf("GetHost in debug = %q, want localhost:8080", got)
	}
}

func TestNewClientRequiresAPIKey(t *testing.T) {
	clearPPIOEnv(t)

	if _, err := NewClient(&SandboxOpts{}); err == nil {
		t.Fatal("expected an error when no API key is configured")
	}
}

func TestNewClientDebugDoesNotRequireAPIKey(t *testing.T) {
	clearPPIOEnv(t)
	debug := true
	if _, err := NewClient(&SandboxOpts{Debug: &debug}); err != nil {
		t.Fatalf("NewClient debug failed: %v", err)
	}
}

func TestNewClientReadsAPIKeyFromEnv(t *testing.T) {
	clearPPIOEnv(t)
	t.Setenv("PPIO_API_KEY", "env-key")

	client, err := NewClient(&SandboxOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.Connection().APIKey != "env-key" {
		t.Errorf("APIKey = %q, want env-key", client.Connection().APIKey)
	}
}

func TestLegacyModePolicy(t *testing.T) {
	cases := []struct {
		name, domain, mode string
		legacy             bool
		timeout            int
	}{
		{"automatic sandbox.ppio.cn", "sandbox.ppio.cn", "", true, 1800000},
		{"automatic us-ga-1.sandbox.ppio.cn", "us-ga-1.sandbox.ppio.cn", "", false, 60000},
		{"automatic us-01.sandbox.ppio.cn", "us-01.sandbox.ppio.cn", "", false, 60000},
		{"automatic us-ga-01.sandbox.ppio.cn", "us-ga-01.sandbox.ppio.cn", "", false, 60000},
		{"automatic us-phx-1.sandbox.ppio.cn", "us-phx-1.sandbox.ppio.cn", "", false, 60000},
		{"automatic preview.example.invalid", "preview.example.invalid", "", false, 60000},
		{"automatic dev.example.invalid", "dev.example.invalid", "", false, 60000},
		{"override true private.example.invalid", "private.example.invalid", "true", true, 1800000},
		{"override  TRUE  private.example.invalid", "private.example.invalid", " TRUE ", true, 1800000},
		{"override false sandbox.ppio.cn", "sandbox.ppio.cn", "false", false, 60000},
		{"override  FALSE  sandbox.ppio.cn", "sandbox.ppio.cn", " FALSE ", false, 60000},
		{"override auto sandbox.ppio.cn", "sandbox.ppio.cn", "auto", true, 1800000},
		{"override auto private.example.invalid", "private.example.invalid", "auto", false, 60000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearPPIOEnv(t)
			t.Setenv("PPIO_IS_LEGACY", tc.mode)
			cfg, err := NewConnectionConfig(&SandboxOpts{Domain: tc.domain, APIURL: "https://api.modern.example.invalid"})
			if err != nil {
				t.Fatal(err)
			}
			if cfg.IsLegacyDomain != tc.legacy || cfg.RequestTimeoutMs != tc.timeout {
				t.Fatalf("legacy=%v timeout=%d, want legacy=%v timeout=%d", cfg.IsLegacyDomain, cfg.RequestTimeoutMs, tc.legacy, tc.timeout)
			}
			if cfg.APIURL != "https://api.modern.example.invalid" {
				t.Fatal("legacy mode changed the API URL")
			}
			explicit := 42000
			cfg, err = NewConnectionConfig(&SandboxOpts{Domain: tc.domain, RequestTimeoutMs: &explicit})
			if err != nil {
				t.Fatal(err)
			}
			if cfg.RequestTimeoutMs != explicit {
				t.Fatal("legacy mode replaced the explicit request timeout")
			}
		})
	}
}
