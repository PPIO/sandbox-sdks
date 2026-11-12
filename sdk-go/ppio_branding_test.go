package ppio

import (
	"testing"

	"github.com/PPIO/sandbox-sdks/sdk-go/core"
)

func TestPPIOFacadeUsesPPIONames(t *testing.T) {
	client, err := NewPPIO(&core.SandboxOpts{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("NewPPIO() error = %v", err)
	}
	var _ *PPIO = client
}
