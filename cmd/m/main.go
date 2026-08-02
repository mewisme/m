package main

import (
	"os"

	"github.com/mewisme/mew/internal/cli"
)

// Overridden via -ldflags "-X main.version=… -X main.commit=… -X main.shortCommit=… -X main.dirty=… -X main.buildDate=… -X main.targetOS=… -X main.targetArch=…".
var (
	version     = "dev"
	commit      = ""
	shortCommit = ""
	dirty       = "false"
	buildDate   = ""
	targetOS    = ""
	targetArch  = ""
)

func main() {
	os.Exit(cli.ExecuteM(cli.BuildInfo{
		Version:     version,
		Commit:      commit,
		ShortCommit: shortCommit,
		Dirty:       dirty == "true",
		BuildDate:   buildDate,
		TargetOS:    targetOS,
		TargetArch:  targetArch,
	}))
}
