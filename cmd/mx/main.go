package main

import (
	"os"

	"github.com/mewisme/m/internal/cli"
)

// Overridden via -ldflags "-X main.version=… -X main.commit=… -X main.buildDate=…".
var (
	version   = "0.0.0-dev"
	commit    = ""
	buildDate = ""
)

func main() {
	os.Exit(cli.ExecuteMX(cli.BuildInfo{
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
	}))
}
