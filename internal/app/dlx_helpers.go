package app

import (
	"github.com/mewisme/mew/internal/runner/dlx"
)

func selectDLXCommand(opts DLXOptions, resolved *dlxResolveResult) (command, owner string, err error) {
	if opts.ModeA {
		bins := dlx.BinNames(resolved.DirectBins[opts.PackageSpecs[0].Name])
		command, err = dlx.InferModeABin(opts.PackageSpecs[0].Name, bins)
		return command, opts.PackageSpecs[0].Name, err
	}
	command = opts.Command
	owner, err = dlx.ResolveModeBCommand(command, resolved.DirectBins)
	return command, owner, err
}
