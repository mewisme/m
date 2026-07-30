package presentation

import "bytes"

// ContainsCSI reports whether b contains a CSI introducer (ESC [).
func ContainsCSI(b []byte) bool {
	return bytes.Contains(b, []byte{0x1b, '['})
}

// ContainsCursorControl reports common cursor/erase CSI sequences used by live UIs.
func ContainsCursorControl(b []byte) bool {
	if !ContainsCSI(b) {
		return false
	}
	for _, marker := range cursorControlMarkers {
		if bytes.Contains(b, marker) {
			return true
		}
	}
	return false
}

var cursorControlMarkers = [][]byte{
	[]byte("\x1b[?25"), // show/hide cursor
	[]byte("\x1b[H"),   // cursor home
	[]byte("\x1b[2J"),  // erase display
	[]byte("\x1b[J"),   // erase in display
	[]byte("\x1b[K"),   // erase in line
	[]byte("\x1b[A"),   // cursor up
	[]byte("\x1b[B"),   // cursor down
	[]byte("\x1b[C"),   // cursor forward
	[]byte("\x1b[D"),   // cursor back
	[]byte("\x1b[s"),   // save cursor
	[]byte("\x1b[u"),   // restore cursor
}
