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
		if len(priorEntry.PeerDependenciesMeta) > 0 {
			if entry.PeerDependenciesMeta == nil {
				entry.PeerDependenciesMeta = make(map[string]PeerMeta, len(priorEntry.PeerDependenciesMeta))
			}
			for k, v := range priorEntry.PeerDependenciesMeta {
				if _, ok := entry.PeerDependenciesMeta[k]; !ok {
					entry.PeerDependenciesMeta[k] = v
				}
			}
		}
		doc.Packages[path] = entry
	}
}
