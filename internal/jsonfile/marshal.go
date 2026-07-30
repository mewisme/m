package jsonfile

import "encoding/json"

// Marshal formats v as 2-space-indented JSON with a trailing newline.
// Use for standalone JSON files on disk only.
func Marshal(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
