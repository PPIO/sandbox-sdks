package core

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Constants mirroring the JS/Python SDKs.
const (
	// RequestTimeoutMS is the default per-request timeout in milliseconds.
	RequestTimeoutMS = 60_000
	// LegacyRequestTimeoutMS is the request timeout used on legacy domains.
	LegacyRequestTimeoutMS = 1_800_000
	// DefaultSandboxTimeoutMS is the default sandbox lifetime.
	DefaultSandboxTimeoutMS = 300_000
	// KeepAlivePingIntervalSec is sent as a header on streaming requests.
	KeepAlivePingIntervalSec = 50
	// KeepAlivePingHeader keeps long-lived streams alive.
	KeepAlivePingHeader = "Keepalive-Ping-Interval"

	// DefaultPPIODomain is the default sandbox domain.
	DefaultPPIODomain = "sandbox.ppio.cn"

	// SDKName is reported to the API for telemetry/version gating.
	SDKName = "sdk_go"
	// SDKVersion is the version of this Go SDK.
	SDKVersion = "0.0.1"

	// EnvdPort is the port envd listens on inside a sandbox.
	EnvdPort = 49983

	// McpPort is the port the MCP gateway listens on inside a sandbox.
	McpPort = 50005

	// DefaultUsername is the user assumed on envd versions older than 0.4.0.
	DefaultUsername = "user"
)

// legacyPPIODomains lists production clusters that use legacy behavior.
var legacyPPIODomains = map[string]bool{
	"sandbox.ppio.cn": true,
}

// EnvdVersion constants gate features that require a minimum envd version.
const (
	EnvdVersionRecursiveWatch = "0.1.4"
	EnvdDebugFallback         = "99.99.99"
	EnvdCommandsStdin         = "0.3.0"
	EnvdDefaultUser           = "0.4.0"
	EnvdEnvdClose             = "0.5.2"
	EnvdOctetStreamUpload     = "0.5.7"
)

// ConnectionConfig holds everything needed to talk to the control plane and envd.
type ConnectionConfig struct {
	APIKey      string
	AccessToken string
	Domain      string
	APIURL      string
	SandboxURL  string
	Debug       bool
	Logger      Logger
	Headers     map[string]string

	RequestTimeoutMs int

	IsLegacyDomain bool
}

// Logger is a minimal logging interface; any logger with these methods works.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// isLegacyDomain reports whether the domain predates modern sandbox features.
func isLegacyDomain(domain string) bool {
	switch strings.ToLower(getEnvVar("PPIO_IS_LEGACY")) {
	case "true":
		return true
	case "false":
		return false
	default:
		return legacyPPIODomains[domain]
	}
}

// getEnvVar reads an environment variable, trimmed.
func getEnvVar(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

// NewConnectionConfig builds a ConnectionConfig from explicit options, falling
// back to the same environment variables the other SDKs use.
func NewConnectionConfig(opts *SandboxOpts) (*ConnectionConfig, error) {
	if opts == nil {
		opts = &SandboxOpts{}
	}

	cfg := &ConnectionConfig{
		APIKey:      firstNonEmpty(opts.APIKey, getEnvVar("PPIO_API_KEY")),
		AccessToken: firstNonEmpty(opts.AccessToken, getEnvVar("PPIO_ACCESS_TOKEN")),
		Domain:      firstNonEmpty(opts.Domain, getEnvVar("PPIO_DOMAIN"), DefaultPPIODomain),
	}

	if cfg.Domain == "" {
		return nil, &InvalidArgumentError{Argument: "domain", Reason: "domain cannot be empty"}
	}

	cfg.APIURL = firstNonEmpty(opts.APIURL, getEnvVar("PPIO_API_URL"))
	cfg.SandboxURL = firstNonEmpty(opts.SandboxURL, getEnvVar("PPIO_SANDBOX_URL"))
	cfg.Logger = opts.Logger

	if opts.Debug != nil {
		cfg.Debug = *opts.Debug
	} else {
		cfg.Debug = strings.EqualFold(getEnvVar("PPIO_DEBUG"), "true")
	}

	cfg.IsLegacyDomain = isLegacyDomain(cfg.Domain)

	// Request timeout: explicit > env > legacy/modern default.
	switch {
	case opts.RequestTimeoutMs != nil:
		cfg.RequestTimeoutMs = *opts.RequestTimeoutMs
	default:
		envTimeout := getEnvVar("PPIO_REQUEST_TIMEOUT")
		if envTimeout != "" {
			if v, err := strconv.Atoi(envTimeout); err == nil {
				cfg.RequestTimeoutMs = v * 1000
			} else {
				cfg.RequestTimeoutMs = RequestTimeoutMS
			}
		} else if cfg.IsLegacyDomain {
			cfg.RequestTimeoutMs = LegacyRequestTimeoutMS
		} else {
			cfg.RequestTimeoutMs = RequestTimeoutMS
		}
	}

	// Base headers, merged with any user-supplied ones.
	cfg.Headers = map[string]string{
		"User-Agent":         fmt.Sprintf("ppio-sandbox-go-sdk/%s", SDKVersion),
		"X-PPIO-SDK-Name":    SDKName,
		"X-PPIO-SDK-Version": SDKVersion,
	}
	for k, v := range opts.Headers {
		cfg.Headers[k] = v
	}

	// Derive the control-plane URL if not given explicitly.
	if cfg.APIURL == "" {
		switch {
		case cfg.Debug:
			cfg.APIURL = "http://localhost:3000"
		case isLegacyDomain(cfg.Domain):
			cfg.APIURL = fmt.Sprintf("https://api.%s", cfg.Domain)
		default:
			cfg.APIURL = fmt.Sprintf("https://api.%s", cfg.Domain)
		}
	}
	cfg.APIURL = normalizeAPIURL(cfg.APIURL)

	return cfg, nil
}

// normalizeAPIURL ensures the URL carries a scheme.
func normalizeAPIURL(u string) string {
	value := strings.TrimSpace(u)
	if value == "" {
		return value
	}
	// Already has a scheme like https:// or http://
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c == ':' {
			return value
		}
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') &&
			c != '+' && c != '.' && c != '-' {
			break
		}
	}
	return "https://" + value
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// GetHost returns the host for a sandbox port.
func (c *ConnectionConfig) GetHost(sandboxID string, port int, sandboxDomain string) string {
	if c.Debug {
		return fmt.Sprintf("localhost:%d", port)
	}
	domain := sandboxDomain
	if domain == "" {
		domain = c.Domain
	}
	return fmt.Sprintf("%d-%s.%s", port, sandboxID, domain)
}

// GetSandboxURL returns the base URL for talking to envd inside the sandbox.
func (c *ConnectionConfig) GetSandboxURL(sandboxID, sandboxDomain string, envdPort int) string {
	if c.SandboxURL != "" {
		return c.SandboxURL
	}
	scheme := "https"
	if c.Debug {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s", scheme, c.GetHost(sandboxID, envdPort, sandboxDomain))
}

// GetSignalTimeout returns the effective request timeout as a duration,
// or zero when no timeout should be applied.
func (c *ConnectionConfig) GetSignalTimeout(requestTimeoutMs *int) time.Duration {
	timeout := c.RequestTimeoutMs
	if requestTimeoutMs != nil {
		timeout = *requestTimeoutMs
	}
	if timeout <= 0 {
		return 0
	}
	return time.Duration(timeout) * time.Millisecond
}

// contextWithTimeout attaches the resolved request timeout to a context.
//
// A nil context is treated as context.Background so callers cannot trigger a
// panic by passing an unset context.
func (c *ConnectionConfig) contextWithTimeout(ctx context.Context, requestTimeoutMs *int) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}

	d := c.GetSignalTimeout(requestTimeoutMs)
	if d <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
