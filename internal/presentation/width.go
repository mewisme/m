package presentation

import (
	"strings"

	"github.com/clipperhouse/displaywidth"
)

// ClampWidth bounds a terminal width to the safe layout range.
func ClampWidth(w int) int {
	if w <= 0 {
		return defaultTermWidth
	}
	if w < minTermWidth {
		return minTermWidth
	}
	if w > maxTermWidth {
		return maxTermWidth
	}
	return w
}

// CellWidth returns the visible terminal cell width of s.
func CellWidth(s string) int {
	return displaywidth.String(s)
}

// MiddleTruncate shortens s to at most maxCells, preserving head and tail.
// ellipsis must already match the active Unicode/ASCII policy.
func MiddleTruncate(s string, maxCells int, ellipsis string) string {
	if maxCells <= 0 {
		return ""
	}
	if CellWidth(s) <= maxCells {
		return s
	}
	ew := CellWidth(ellipsis)
	if ew >= maxCells {
		return trimToWidth(ellipsis, maxCells)
	}
	remain := maxCells - ew
	head := remain / 2
	tail := remain - head
	return trimToWidth(s, head) + ellipsis + trimFromEnd(s, tail)
}

// WrapWords wraps s to width cells using spaces; long tokens are hard-split.
func WrapWords(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		if strings.TrimSpace(s) == "" {
			return []string{""}
		}
		return hardWrap(s, width)
	}
	var lines []string
	var cur strings.Builder
	curWidth := 0
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		lines = append(lines, cur.String())
		cur.Reset()
		curWidth = 0
	}
	for _, word := range fields {
		ww := CellWidth(word)
		if ww > width {
			flush()
			lines = append(lines, hardWrap(word, width)...)
			continue
		}
		need := ww
		if curWidth > 0 {
			need++
		}
		if curWidth > 0 && curWidth+need > width {
			flush()
		}
		if curWidth > 0 {
			cur.WriteByte(' ')
			curWidth++
		}
		cur.WriteString(word)
		curWidth += ww
	}
	flush()
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func hardWrap(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var lines []string
	var cur strings.Builder
	curWidth := 0
	for _, r := range s {
		rw := runeWidth(r)
		if curWidth+rw > width && cur.Len() > 0 {
			lines = append(lines, cur.String())
			cur.Reset()
			curWidth = 0
		}
		cur.WriteRune(r)
		curWidth += rw
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func runeWidth(r rune) int {
	rw := displaywidth.String(string(r))
	if rw <= 0 {
		return 1
	}
	return rw
}

func trimToWidth(s string, maxCells int) string {
	if maxCells <= 0 {
		return ""
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := runeWidth(r)
		if w+rw > maxCells {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String()
}

func trimFromEnd(s string, maxCells int) string {
	if maxCells <= 0 {
		return ""
	}
	runes := []rune(s)
	var out []rune
	w := 0
	for i := len(runes) - 1; i >= 0; i-- {
		rw := runeWidth(runes[i])
		if w+rw > maxCells {
			break
		}
		out = append(out, runes[i])
		w += rw
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}
