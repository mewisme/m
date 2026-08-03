package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Comment-preserving edits for JSONC config files.
//
// The trick is blankJSONC: it replaces comment bytes with spaces without
// changing length, so encoding/json decoder offsets taken over the blanked
// buffer are exact byte spans in the original source. Edits are byte splices
// into the original, so every comment, blank line, and indentation choice
// outside the edited span survives untouched.

// member is the located span of one object member within a source buffer.
type member struct {
	keyStart   int // first byte of the quoted key
	valueStart int
	valueEnd   int
}

// findMember locates key inside the object whose body spans [objStart, objEnd)
// in the blanked buffer. objStart must point at the opening '{'.
func findMember(blank []byte, objStart, objEnd int, key string) (member, bool, error) {
	dec := json.NewDecoder(bytes.NewReader(blank[objStart:objEnd]))
	tok, err := dec.Token()
	if err != nil {
		return member{}, false, err
	}
	if tok != json.Delim('{') {
		return member{}, false, fmt.Errorf("not an object")
	}
	for dec.More() {
		// Offset before the key token so we can find where the member starts.
		keyOff := objStart + int(dec.InputOffset())
		kt, err := dec.Token()
		if err != nil {
			return member{}, false, err
		}
		k, ok := kt.(string)
		if !ok {
			return member{}, false, fmt.Errorf("object key must be string")
		}
		// Advance keyOff to the opening quote of the key.
		for keyOff < objEnd && blank[keyOff] != '"' {
			keyOff++
		}

		valStart := objStart + int(dec.InputOffset())
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return member{}, false, err
		}
		valEnd := objStart + int(dec.InputOffset())
		// InputOffset after the key token sits before the ':' separator, so
		// skip whitespace, then the colon, then whitespace again.
		for valStart < valEnd && unicode.IsSpace(rune(blank[valStart])) {
			valStart++
		}
		if valStart < valEnd && blank[valStart] == ':' {
			valStart++
		}
		for valStart < valEnd && unicode.IsSpace(rune(blank[valStart])) {
			valStart++
		}
		for valEnd > valStart && unicode.IsSpace(rune(blank[valEnd-1])) {
			valEnd--
		}
		if k == key {
			return member{keyStart: keyOff, valueStart: valStart, valueEnd: valEnd}, true, nil
		}
	}
	return member{}, false, nil
}

// ErrScalarParent reports that a dotted path tried to descend through a value
// that is not an object. Creating the missing child would silently shadow the
// existing scalar with a duplicate key, so the edit is refused instead.
var ErrScalarParent = errors.New("intermediate key is not an object")

// findPath walks a dotted path and returns the span of the final value.
// Returns found=false with the deepest existing object span in parent when
// some segment is missing. Returns ErrScalarParent when a segment exists but
// holds a scalar, because descending further is structurally ambiguous.
func findPath(blank []byte, path []string) (m member, found bool, parentStart, parentEnd int, depth int, err error) {
	objStart, objEnd := 0, len(blank)
	// Position objStart at the root '{'.
	for objStart < objEnd && unicode.IsSpace(rune(blank[objStart])) {
		objStart++
	}
	if objStart >= objEnd || blank[objStart] != '{' {
		return member{}, false, 0, 0, 0, fmt.Errorf("root must be object")
	}
	for i, seg := range path {
		got, ok, ferr := findMember(blank, objStart, objEnd, seg)
		if ferr != nil {
			return member{}, false, 0, 0, 0, ferr
		}
		if !ok {
			return member{}, false, objStart, objEnd, i, nil
		}
		if i == len(path)-1 {
			return got, true, objStart, objEnd, i, nil
		}
		// Descend: the value must be an object.
		vs := got.valueStart
		if vs >= len(blank) || blank[vs] != '{' {
			return member{}, false, objStart, objEnd, i,
				fmt.Errorf("%w: %q", ErrScalarParent, strings.Join(path[:i+1], "."))
		}
		objStart, objEnd = vs, got.valueEnd
	}
	return member{}, false, objStart, objEnd, 0, nil
}

// setJSONCPath splices value in at the dotted key, creating missing
// intermediate objects. Comments outside the edited span are preserved.
func setJSONCPath(src []byte, dotted string, value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	path := strings.Split(dotted, ".")
	blank := blankJSONC(src)

	m, found, parentStart, parentEnd, depth, err := findPath(blank, path)
	if err != nil {
		return nil, err
	}
	if found {
		return concatBytes(src[:m.valueStart], raw, src[m.valueEnd:]), nil
	}

	// Build the nested remainder that does not exist yet, innermost last.
	remaining := path[depth:]
	nested := raw
	for i := len(remaining) - 1; i >= 1; i-- {
		keyJSON, _ := json.Marshal(remaining[i])
		nested = []byte(fmt.Sprintf("{%s: %s}", keyJSON, nested))
	}
	return insertMember(src, blank, parentStart, parentEnd, remaining[0], nested)
}

// unsetJSONCPath removes the member at the dotted key, along with its key,
// separating comma, and the whitespace of its own line. Parent objects left
// empty by the removal are pruned too. Returns changed=false when the key is
// absent.
func unsetJSONCPath(src []byte, dotted string) ([]byte, bool, error) {
	path := strings.Split(dotted, ".")
	out, changed, err := removeMemberAt(src, path)
	if err != nil || !changed {
		return src, false, err
	}
	// Prune parents that the removal emptied, innermost first.
	for i := len(path) - 1; i >= 1; i-- {
		parent := path[:i]
		empty, perr := isEmptyObjectAt(out, parent)
		if perr != nil || !empty {
			break
		}
		pruned, ok, perr := removeMemberAt(out, parent)
		if perr != nil || !ok {
			break
		}
		out = pruned
	}
	return out, true, nil
}

// isEmptyObjectAt reports whether the value at the dotted path is an object
// with no members.
func isEmptyObjectAt(src []byte, path []string) (bool, error) {
	parsed, err := ParseJSONC(src)
	if err != nil {
		return false, err
	}
	cur, ok := parsed.(map[string]any)
	if !ok {
		return false, nil
	}
	for i, seg := range path {
		v, present := cur[seg]
		if !present {
			return false, nil
		}
		next, isObj := v.(map[string]any)
		if !isObj {
			return false, nil
		}
		if i == len(path)-1 {
			return len(next) == 0, nil
		}
		cur = next
	}
	return false, nil
}

// removeMemberAt deletes exactly the member at the dotted path.
func removeMemberAt(src []byte, path []string) ([]byte, bool, error) {
	blank := blankJSONC(src)

	m, found, _, _, _, err := findPath(blank, path)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return src, false, nil
	}

	start, end := m.keyStart, m.valueEnd
	// Absorb a trailing comma if present.
	j := end
	for j < len(blank) && unicode.IsSpace(rune(blank[j])) {
		j++
	}
	if j < len(blank) && blank[j] == ',' {
		end = j + 1
	} else {
		// Last member: absorb a preceding comma instead.
		k := start - 1
		for k >= 0 && unicode.IsSpace(rune(blank[k])) {
			k--
		}
		if k >= 0 && blank[k] == ',' {
			start = k
		}
	}
	// Absorb the leading indentation of the member's own line.
	for start > 0 && (src[start-1] == ' ' || src[start-1] == '\t') {
		start--
	}
	// Absorb the newline the member occupied so no blank line is left behind.
	if start > 0 && src[start-1] == '\n' {
		start--
		if start > 0 && src[start-1] == '\r' {
			start--
		}
	}
	return concatBytes(src[:start], nil, src[end:]), true, nil
}

// insertMember adds "key": value into the object spanning [objStart, objEnd),
// matching the surrounding indentation.
func insertMember(src, blank []byte, objStart, objEnd int, key string, value []byte) ([]byte, error) {
	// Locate the object's closing brace in the blanked buffer.
	closeIdx := objEnd - 1
	for closeIdx > objStart && blank[closeIdx] != '}' {
		closeIdx--
	}
	if closeIdx <= objStart {
		return nil, fmt.Errorf("cannot find closing brace")
	}
	// Is the object empty (ignoring comments, which are blanks here)?
	j := closeIdx - 1
	for j > objStart && unicode.IsSpace(rune(blank[j])) {
		j--
	}
	empty := j == objStart && blank[j] == '{'

	keyJSON, err := json.Marshal(key)
	if err != nil {
		return nil, err
	}
	closeIndent := lineIndentAt(src, closeIdx)

	if empty {
		insert := fmt.Sprintf("\n%s%s: %s\n%s", closeIndent+"  ", keyJSON, value, closeIndent)
		return concatBytes(src[:objStart+1], []byte(insert), src[closeIdx:]), nil
	}

	// Match the indentation existing members already use.
	indent := memberIndent(src, blank, objStart, closeIdx)
	if indent == "" {
		indent = closeIndent + "  "
	}
	// Insert after the last member: the last non-space byte before '}'.
	tail := closeIdx
	for tail > objStart && unicode.IsSpace(rune(blank[tail-1])) {
		tail--
	}
	insert := fmt.Sprintf(",\n%s%s: %s", indent, keyJSON, value)
	return concatBytes(src[:tail], []byte(insert), src[tail:]), nil
}

// memberIndent returns the leading whitespace used by members of the object
// spanning [objStart, objEnd), or "" when it cannot be determined.
func memberIndent(src, blank []byte, objStart, objEnd int) string {
	if objEnd > len(src) {
		objEnd = len(src)
	}
	// Scan the blanked buffer so a quote inside a comment is not mistaken for
	// a member, but slice the indent out of the real source.
	for i := objStart; i < objEnd; i++ {
		if blank[i] != '"' {
			continue
		}
		return lineIndentAt(src, i)
	}
	return ""
}

// lineIndentAt returns the leading whitespace on the line containing pos.
func lineIndentAt(src []byte, pos int) string {
	if pos > len(src) {
		pos = len(src)
	}
	start := pos
	for start > 0 && src[start-1] != '\n' {
		start--
	}
	end := start
	for end < pos && (src[end] == ' ' || src[end] == '\t') {
		end++
	}
	return string(src[start:end])
}

func concatBytes(a, b, c []byte) []byte {
	out := make([]byte, 0, len(a)+len(b)+len(c))
	out = append(out, a...)
	out = append(out, b...)
	out = append(out, c...)
	return out
}

// DuplicateKeyError names one object key that appears twice in the same
// object. encoding/json silently keeps the last occurrence, which makes a
// duplicate look like a working setting that does nothing.
type DuplicateKeyError struct {
	Path string
}

func (e *DuplicateKeyError) Error() string {
	return "duplicate key " + strconv.Quote(e.Path)
}

// DetectDuplicateKeys reports the first duplicated object key in a JSONC
// document, walking nested objects and arrays. Comments are blanked first so
// a key mentioned in a comment is never counted.
func DetectDuplicateKeys(src []byte) error {
	dec := json.NewDecoder(bytes.NewReader(blankJSONC(src)))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		// Malformed input is the parser's error to report, not ours.
		return nil //nolint:nilerr // parse errors surface through ParseJSONC
	}
	if tok != json.Delim('{') {
		return nil
	}
	return walkDuplicates(dec, "")
}

// walkDuplicates consumes members of the object the decoder just opened.
func walkDuplicates(dec *json.Decoder, prefix string) error {
	seen := map[string]bool{}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil //nolint:nilerr // parse errors surface through ParseJSONC
		}
		k, ok := kt.(string)
		if !ok {
			return nil
		}
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		if seen[k] {
			return &DuplicateKeyError{Path: path}
		}
		seen[k] = true
		if err := walkDuplicateValue(dec, path); err != nil {
			return err
		}
	}
	// Consume the closing '}'.
	if _, err := dec.Token(); err != nil {
		return nil //nolint:nilerr // parse errors surface through ParseJSONC
	}
	return nil
}

// walkDuplicateValue consumes exactly one value, descending into containers.
func walkDuplicateValue(dec *json.Decoder, path string) error {
	tok, err := dec.Token()
	if err != nil {
		return nil //nolint:nilerr // parse errors surface through ParseJSONC
	}
	switch tok {
	case json.Delim('{'):
		return walkDuplicates(dec, path)
	case json.Delim('['):
		for dec.More() {
			if err := walkDuplicateValue(dec, path); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil {
			return nil //nolint:nilerr // parse errors surface through ParseJSONC
		}
	}
	return nil
}
