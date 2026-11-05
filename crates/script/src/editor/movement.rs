// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl ScriptEditor {
    /// Place a clamped caret, optionally extending its anchor; example: `editor.set_caret(position, true)`.
    pub fn set_caret(&mut self, position: EditorPosition, extend: bool) {
        self.move_caret(self.buffer.clamp(position), extend, false);
    }
    /// Move left, or collapse a selection to its near edge; example: `editor.move_left(false)`.
    pub fn move_left(&mut self, extend: bool) {
        if !extend && !self.selection.is_empty() {
            self.collapse(self.selection.ordered().0);
            return;
        }
        self.move_caret(self.buffer.left(self.selection.caret), extend, false);
    }
    /// Move right, or collapse a selection to its far edge; example: `editor.move_right(true)`.
    pub fn move_right(&mut self, extend: bool) {
        if !extend && !self.selection.is_empty() {
            self.collapse(self.selection.ordered().1);
            return;
        }
        self.move_caret(self.buffer.right(self.selection.caret), extend, false);
    }
    /// Select the full source with the caret at its end; example: `editor.select_all()`.
    pub fn select_all(&mut self) {
        self.selection = EditorSelection {
            anchor: EditorPosition::default(),
            caret: self.buffer.end(),
        };
        self.goal_column = self.selection.caret.column;
    }
    /// Move up while retaining the original goal column; example: `editor.move_up(true)`.
    pub fn move_up(&mut self, extend: bool) {
        if self.selection.caret.line == 0 {
            self.move_caret(EditorPosition::default(), extend, true);
            return;
        }
        self.vertical(self.selection.caret.line - 1, extend);
    }
    /// Move down while retaining the original goal column; example: `editor.move_down(false)`.
    pub fn move_down(&mut self, extend: bool) {
        if self.selection.caret.line + 1 == self.line_count() {
            self.move_caret(self.buffer.end(), extend, true);
            return;
        }
        self.vertical(self.selection.caret.line + 1, extend);
    }
    /// Toggle smart Home between indentation and column zero; example: `editor.move_home(false)`.
    pub fn move_home(&mut self, extend: bool) {
        let caret = self.selection.caret;
        let indentation = self.buffer.lines[caret.line]
            .iter()
            .take_while(|character| **character == ' ' || **character == '\t')
            .count();
        let column = if caret.column > indentation {
            indentation
        } else {
            0
        };
        self.move_caret(EditorPosition { column, ..caret }, extend, false);
    }
    /// Go to line end; example: `editor.move_end(true)` selects to its end.
    pub fn move_end(&mut self, extend: bool) {
        let caret = self.selection.caret;
        self.move_caret(
            EditorPosition {
                column: self.buffer.lines[caret.line].len(),
                ..caret
            },
            extend,
            false,
        );
    }
    /// Go to source start; example: `editor.move_document_start(false)`.
    pub fn move_document_start(&mut self, extend: bool) {
        self.move_caret(EditorPosition::default(), extend, false);
    }
    /// Go to source end; example: `editor.move_document_end(true)`.
    pub fn move_document_end(&mut self, extend: bool) {
        self.move_caret(self.buffer.end(), extend, false);
    }
    fn vertical(&mut self, line: usize, extend: bool) {
        let target = EditorPosition {
            line,
            column: self.goal_column.min(self.buffer.lines[line].len()),
        };
        self.move_caret(target, extend, true);
    }
    fn move_caret(&mut self, caret: EditorPosition, extend: bool, vertical: bool) {
        self.selection.caret = caret;
        if !extend {
            self.selection.anchor = caret;
        }
        if !vertical {
            self.goal_column = caret.column;
        }
    }
}
