package config

// LoadSpec captures immutable config load inputs for one CLI invocation.
// Mutation reload clones this spec and may only change ProjectRoot.
type LoadSpec struct {
	CWD                  string
	ProjectRoot          string
	ProjectPath          string // absolute; empty = default <root>/m.jsonc
	GlobalPath           string // absolute; empty = default global path
	Env                  []string
	EnvSnapshot          EnvSnapshot
	CLI                  map[string]any
	RequireProjectConfig bool
	RequireGlobalConfig  bool
}

// LoadSpecFromOptions builds a defensive copy of load inputs.
func LoadSpecFromOptions(opts LoadOptions) LoadSpec {
	spec := LoadSpec{
		CWD:                  opts.CWD,
		ProjectRoot:          opts.ProjectRoot,
		ProjectPath:          opts.ProjectPath,
		GlobalPath:           opts.GlobalPath,
		RequireProjectConfig: opts.RequireProjectConfig,
		RequireGlobalConfig:  opts.RequireGlobalConfig,
	}
	if opts.Env != nil {
		spec.Env = append([]string(nil), opts.Env...)
	}
	if opts.EnvSnapshot.Initialized() {
		spec.EnvSnapshot = opts.EnvSnapshot.Clone()
	}
	if opts.CLI != nil {
		spec.CLI = make(map[string]any, len(opts.CLI))
		for k, v := range opts.CLI {
			spec.CLI[k] = v
		}
	}
	return spec
}

// Clone returns a deep copy of the load spec.
func (s LoadSpec) Clone() LoadSpec {
	out := s
	if s.Env != nil {
		out.Env = append([]string(nil), s.Env...)
	}
	out.EnvSnapshot = s.EnvSnapshot.Clone()
	if s.CLI != nil {
		out.CLI = make(map[string]any, len(s.CLI))
		for k, v := range s.CLI {
			out.CLI[k] = v
		}
	}
	return out
}

// WithProjectRoot returns a copy with an updated project root (mutation reload only).
func (s LoadSpec) WithProjectRoot(root string) LoadSpec {
	out := s.Clone()
	out.ProjectRoot = root
	return out
}

// LoadOptions maps the spec back to LoadOptions for config.Load.
func (s LoadSpec) LoadOptions() LoadOptions {
	opts := LoadOptions{
		CWD:                  s.CWD,
		ProjectRoot:          s.ProjectRoot,
		ProjectPath:          s.ProjectPath,
		GlobalPath:           s.GlobalPath,
		RequireProjectConfig: s.RequireProjectConfig,
		RequireGlobalConfig:  s.RequireGlobalConfig,
	}
	if s.Env != nil {
		opts.Env = append([]string(nil), s.Env...)
	}
	if s.EnvSnapshot.Initialized() {
		opts.EnvSnapshot = s.EnvSnapshot.Clone()
	}
	if s.CLI != nil {
		opts.CLI = make(map[string]any, len(s.CLI))
		for k, v := range s.CLI {
			opts.CLI[k] = v
		}
	}
	return opts
}
