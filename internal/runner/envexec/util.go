package envexec

import "os"

func osEnviron() []string {
	return os.Environ()
}

func hostEnvFrom(deps ProviderDeps) []string {
	if deps.HostEnv != nil {
		return deps.HostEnv()
	}
	return osEnviron()
}
