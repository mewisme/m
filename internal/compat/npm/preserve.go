package npm

import "encoding/json"

func preservePriorPackageFields(doc, prior *Document) {
	if doc == nil || prior == nil {
		return
	}
	for path, priorEntry := range prior.Packages {
		entry, ok := doc.Packages[path]
		if !ok {
			doc.Packages[path] = priorEntry.clone()
			continue
		}
		for k, v := range priorEntry.Extra {
			if _, exists := entry.Extra[k]; !exists {
				if entry.Extra == nil {
					entry.Extra = map[string]json.RawMessage{}
				}
				entry.Extra[k] = append(json.RawMessage(nil), v...)
			}
		}
		if len(priorEntry.BundledDependencies) > 0 && len(entry.BundledDependencies) == 0 {
			entry.BundledDependencies = append([]string(nil), priorEntry.BundledDependencies...)
		}
		doc.Packages[path] = entry
	}
}
