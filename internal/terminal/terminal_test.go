package terminal

import (
	"os"
	"runtime"
	"testing"
)

func TestInit(t *testing.T) {
	info := Init()
	if info.OS != runtime.GOOS {
		t.Errorf("expected OS %q, got %q", runtime.GOOS, info.OS)
	}
	if info.IsWindows != (runtime.GOOS == "windows") {
		t.Errorf("expected IsWindows to match runtime.GOOS")
	}
}

func TestNoColorEnv(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	info := Init()
	if info.ColorSupported {
		t.Errorf("expected ColorSupported to be false when NO_COLOR is set")
	}
}

func TestDumbTerm(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	os.Setenv("TERM", "dumb")
	defer os.Unsetenv("TERM")

	info := Init()
	if info.ColorSupported {
		t.Errorf("expected ColorSupported to be false when TERM=dumb")
	}
}

func TestIsTerminal(t *testing.T) {
	// Nil file should safely return false
	if IsTerminal(nil) {
		t.Errorf("expected nil file to return false")
	}
}
