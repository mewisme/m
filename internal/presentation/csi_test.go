package presentation

import (
	"testing"
)

func TestCLIUXContainsCSI(t *testing.T) {
	if ContainsCSI([]byte("plain text")) {
		t.Fatal("false positive on plain text")
	}
	if !ContainsCSI([]byte("x\x1b[31mred")) {
		t.Fatal("missed CSI introducer")
	}
	if ContainsCursorControl([]byte("x\x1b[31mred")) {
		t.Fatal("SGR color is not cursor control")
	}
	if !ContainsCursorControl([]byte("\x1b[2Jclear")) {
		t.Fatal("missed erase display")
	}
	if !ContainsCursorControl([]byte("\x1b[?25lhide")) {
		t.Fatal("missed hide cursor")
	}
	if !ContainsCursorControl([]byte("\x1b[Aup")) {
		t.Fatal("missed cursor up")
	}
}
