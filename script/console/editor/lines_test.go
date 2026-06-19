// SPDX-License-Identifier: GPL-2.0-only

package editor

import "testing"

func TestToggleLineCommentAddsAtSharedIndent(t *testing.T) {
	m := New("    foo\n    bar")
	m.SelectAll()
	m.ToggleLineComment()
	if m.Text() != "    -- foo\n    -- bar" {
		t.Fatalf("comment add = %q, want shared-indent comments", m.Text())
	}
	// Toggling again removes them (all lines are commented).
	m.ToggleLineComment()
	if m.Text() != "    foo\n    bar" {
		t.Fatalf("comment remove = %q, want original", m.Text())
	}
}

func TestToggleLineCommentSingleLineNoSelection(t *testing.T) {
	m := New("x = 1")
	m.ToggleLineComment()
	if m.Text() != "-- x = 1" {
		t.Fatalf("single-line comment = %q", m.Text())
	}
}

func TestToggleLineCommentMixedRecomments(t *testing.T) {
	// One line already commented, one not: the toggle should comment the uncommented one
	// (not uncomment), because not all lines are commented.
	m := New("-- a\nb")
	m.SelectAll()
	m.ToggleLineComment()
	if m.Text() != "-- -- a\n-- b" {
		t.Fatalf("mixed toggle = %q, want both commented", m.Text())
	}
}

func TestIndentAndOutdentSelection(t *testing.T) {
	m := New("foo\nbar")
	m.SelectAll()
	m.IndentSelection()
	if m.Text() != "    foo\n    bar" {
		t.Fatalf("indent = %q", m.Text())
	}
	m.OutdentSelection()
	if m.Text() != "foo\nbar" {
		t.Fatalf("outdent = %q", m.Text())
	}
}

func TestOutdentPartialAndTab(t *testing.T) {
	m := New("  two\n\tone")
	m.SelectAll()
	m.OutdentSelection()
	if m.Text() != "two\none" {
		t.Fatalf("outdent partial/tab = %q, want both leading indent removed", m.Text())
	}
}

func TestToggleCommentIsOneUndoStep(t *testing.T) {
	m := New("a\nb")
	m.SelectAll()
	m.ToggleLineComment()
	m.Undo()
	if m.Text() != "a\nb" {
		t.Fatalf("after undo = %q, want a single-step revert", m.Text())
	}
}

func TestBlankLinesNotCommented(t *testing.T) {
	m := New("foo\n\nbar")
	m.SelectAll()
	m.ToggleLineComment()
	if m.Text() != "-- foo\n\n-- bar" {
		t.Fatalf("blank line got a comment: %q", m.Text())
	}
}
