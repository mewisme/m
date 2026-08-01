package transform

// stripJSONC removes line comments (//), block comments (/* */), and trailing
// commas from JSONC, handling strings and escapes.
func stripJSONC(b []byte) []byte {
	b = stripComments(b)
	b = stripTrailingCommas(b)
	return b
}

// stripComments removes // and /* */ comments from b.
func stripComments(b []byte) []byte {
	out := make([]byte, 0, len(b))
	i := 0
	n := len(b)
	for i < n {
		c := b[i]
		switch c {
		case '"':
			out = append(out, c)
			i++
			// Scan string literal.
			for i < n {
				ch := b[i]
				out = append(out, ch)
				i++
				if ch == '\\' && i < n {
					out = append(out, b[i])
					i++
				} else if ch == '"' {
					break
				}
			}
		case '/':
			if i+1 < n {
				next := b[i+1]
				if next == '/' {
					// Line comment: skip to end of line.
					i += 2
					for i < n && b[i] != '\n' {
						i++
					}
					// Keep the newline.
					if i < n {
						out = append(out, '\n')
						i++
					}
					continue
				}
				if next == '*' {
					// Block comment: skip to */.
					i += 2
					for i+1 < n {
						if b[i] == '*' && b[i+1] == '/' {
							i += 2
							break
						}
						i++
					}
					// Replace with single space to preserve token separation.
					if len(out) > 0 && out[len(out)-1] != ' ' && out[len(out)-1] != '\n' {
						out = append(out, ' ')
					}
					continue
				}
			}
			out = append(out, c)
			i++
		default:
			out = append(out, c)
			i++
		}
	}
	return out
}

// stripTrailingCommas removes commas immediately before } or ] (with optional whitespace).
func stripTrailingCommas(b []byte) []byte {
	out := make([]byte, 0, len(b))
	i := 0
	n := len(b)
	for i < n {
		c := b[i]

		// Track strings to avoid stripping commas inside string literals.
		if c == '"' {
			out = append(out, c)
			i++
			for i < n {
				ch := b[i]
				out = append(out, ch)
				i++
				if ch == '\\' && i < n {
					out = append(out, b[i])
					i++
				} else if ch == '"' {
					break
				}
			}
			continue
		}

		if c == ',' {
			// Check if next non-whitespace char is } or ].
			j := i + 1
			for j < n && (b[j] == ' ' || b[j] == '\t' || b[j] == '\n' || b[j] == '\r') {
				j++
			}
			if j < n && (b[j] == '}' || b[j] == ']') {
				// Skip the trailing comma.
				i = j
				continue
			}
		}
		out = append(out, c)
		i++
	}
	return out
}
