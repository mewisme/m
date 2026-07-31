package presentation

import (
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
)

// ColorProfile is the terminal color support level for one invocation.
type ColorProfile int

const (
	ColorProfileNone ColorProfile = iota
	ColorProfileASCII
	ColorProfileANSI
	ColorProfileANSI256
	ColorProfileTrueColor
)

// BackgroundMode is the terminal background luminance hint.
type BackgroundMode string

const (
	BackgroundLight   BackgroundMode = "light"
	BackgroundDark    BackgroundMode = "dark"
	BackgroundUnknown BackgroundMode = "unknown"
)

const (
	defaultTermWidth  = 80
	defaultTermHeight = 24
	minTermWidth      = 20
	maxTermWidth      = 500
)

// Capabilities is an immutable terminal/environment snapshot for one invocation.
type Capabilities struct {
	StdinTTY     bool
	StdoutTTY    bool
	StderrTTY    bool
	Width        int
	Height       int
	CI           bool
	DumbTerminal bool
	Unicode      bool
	Interactive  bool
	ScreenReader bool
	ColorProfile ColorProfile
	Background   BackgroundMode
	Hyperlinks   bool
	Term         string
}

// EnvLookup reads one environment variable (typically os.LookupEnv).
type EnvLookup func(key string) (string, bool)

// DetectCapabilities probes readers/writers and environment once per invocation.
func DetectCapabilities(stdin io.Reader, stdout, stderr io.Writer, lookup EnvLookup) Capabilities {
	var env []string
	if lookup == nil {
		lookup = os.LookupEnv
		env = os.Environ()
	} else {
		env = environFromLookup(lookup)
	}

	caps := Capabilities{
		StdinTTY:   isReaderTTY(stdin),
		StdoutTTY:  isTTY(stdout),
		StderrTTY:  isTTY(stderr),
		Width:      defaultTermWidth,
		Height:     defaultTermHeight,
		Background: BackgroundLight,
	}
	if v, ok := lookup("CI"); ok && isTruthy(v) {
		caps.CI = true
	}
	if _, ok := lookup("GITHUB_ACTIONS"); ok {
		caps.CI = true
	}
	if termName, ok := lookup("TERM"); ok {
		caps.Term = termName
		caps.DumbTerminal = strings.EqualFold(termName, "dumb")
	}

	caps.ColorProfile = mapColorProfile(colorprofile.Detect(stdout, env))
	caps.Background = detectBackground(lookup)
	caps.Unicode = unicodeCapable(caps, lookup)
	caps.Interactive = caps.StdinTTY && !caps.CI && !caps.DumbTerminal
	caps.Hyperlinks = hyperlinkCapable(lookup)

	if w, h, ok := terminalSize(stdout); ok {
		caps.Width = ClampWidth(w)
		if h > 0 {
			caps.Height = h
		}
	} else if w, h, ok := terminalSize(stderr); ok {
		caps.Width = ClampWidth(w)
		if h > 0 {
			caps.Height = h
		}
	}

	return caps
}

func environFromLookup(lookup EnvLookup) []string {
	keys := []string{
		"TERM", "COLORTERM", "CLICOLOR", "CLICOLOR_FORCE",
		"CI", "GITHUB_ACTIONS", "WT_SESSION", "TERM_PROGRAM", "COLORFGBG",
	}
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if v, ok := lookup(k); ok {
			out = append(out, k+"="+v)
		}
	}
	return out
}

func mapColorProfile(p colorprofile.Profile) ColorProfile {
	switch p {
	case colorprofile.TrueColor:
		return ColorProfileTrueColor
	case colorprofile.ANSI256:
		return ColorProfileANSI256
	case colorprofile.ANSI:
		return ColorProfileANSI
	case colorprofile.ASCII:
		return ColorProfileASCII
	default:
		return ColorProfileNone
	}
}

func detectBackground(lookup EnvLookup) BackgroundMode {
	fgbg, ok := lookup("COLORFGBG")
	if !ok || strings.TrimSpace(fgbg) == "" {
		return BackgroundLight
	}
	parts := strings.Split(fgbg, ";")
	bg := strings.TrimSpace(parts[len(parts)-1])
	n, err := strconv.Atoi(bg)
	if err != nil {
		return BackgroundLight
	}
	if n <= 6 || n == 8 {
		return BackgroundDark
	}
	return BackgroundLight
}

func unicodeCapable(caps Capabilities, lookup EnvLookup) bool {
	if caps.DumbTerminal || caps.ScreenReader {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return true
}

func hyperlinkCapable(lookup EnvLookup) bool {
	if prog, ok := lookup("TERM_PROGRAM"); ok {
		switch strings.ToLower(prog) {
		case "iterm.app", "wezterm", "ghostty", "vscode":
			return true
		}
	}
	if _, ok := lookup("WT_SESSION"); ok {
		return true
	}
	return false
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(f.Fd())
}

func isReaderTTY(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(f.Fd())
}

func terminalSize(w io.Writer) (width, height int, ok bool) {
	f, isFile := w.(*os.File)
	if !isFile {
		return 0, 0, false
	}
	width, height, err := term.GetSize(f.Fd())
	if err != nil || width <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// SupportsColor reports whether the profile can emit ANSI colors.
func (c Capabilities) SupportsColor() bool {
	return c.ColorProfile >= ColorProfileANSI
}
