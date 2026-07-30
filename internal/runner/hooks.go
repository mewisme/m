package runner

// ExpandHooks returns pre/main/post lifecycle stages for one script name.
// Missing hook scripts are skipped; only defined stages are returned.
func ExpandHooks(scripts map[string]string, name string) []HookStage {
	events := []string{"pre" + name, name, "post" + name}
	out := make([]HookStage, 0, len(events))
	for _, event := range events {
		body, ok := scripts[event]
		if !ok {
			continue
		}
		out = append(out, HookStage{Event: event, Script: body})
	}
	return out
}

// ExpandPlans resolves hook stages for each script name in order.
// Execution layers stop on the first failing stage; this function only expands plans.
func ExpandPlans(scripts map[string]string, names []string) []ScriptPlan {
	if len(names) == 0 {
		return nil
	}
	out := make([]ScriptPlan, 0, len(names))
	for _, name := range names {
		stages := ExpandHooks(scripts, name)
		if len(stages) == 0 {
			continue
		}
		out = append(out, ScriptPlan{Name: name, Stages: stages})
	}
	return out
}
