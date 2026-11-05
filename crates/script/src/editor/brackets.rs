// SPDX-License-Identifier: GPL-2.0-only
use super::{EditorPosition, ScriptEditor};
impl ScriptEditor {
    /// Match the bracket at or before the caret, across lines; example: `editor.matching_bracket()`.
    /// This lexical Go behavior counts brackets inside strings/comments too.
    pub fn matching_bracket(&self) -> Option<(EditorPosition, EditorPosition)> {
        for position in [self.selection.caret, self.buffer.left(self.selection.caret)] {
            let partner = self.character_at(position).and_then(bracket_kind).and_then(
                |(open, close, forward)| self.scan_bracket(position, open, close, forward),
            );
            if let Some(partner) = partner {
                return Some((position, partner));
            }
        }
        None
    }
    fn character_at(&self, position: EditorPosition) -> Option<char> {
        self.buffer.lines[position.line]
            .get(position.column)
            .copied()
    }
    fn scan_bracket(
        &self,
        mut position: EditorPosition,
        open: char,
        close: char,
        forward: bool,
    ) -> Option<EditorPosition> {
        let mut depth: i64 = 0;
        loop {
            depth += bracket_delta(self.character_at(position), open, close);
            if depth == 0 {
                return Some(position);
            }
            let next = self.bracket_step(position, forward);
            if next == position {
                return None;
            }
            position = next;
        }
    }
    fn bracket_step(&self, position: EditorPosition, forward: bool) -> EditorPosition {
        if forward {
            self.buffer.right(position)
        } else {
            self.buffer.left(position)
        }
    }
}
fn bracket_kind(character: char) -> Option<(char, char, bool)> {
    match character {
        '(' | ')' => Some(('(', ')', character == '(')),
        '[' | ']' => Some(('[', ']', character == '[')),
        '{' | '}' => Some(('{', '}', character == '{')),
        _ => None,
    }
}
fn bracket_delta(character: Option<char>, open: char, close: char) -> i64 {
    match character {
        Some(character) if character == open => 1,
        Some(character) if character == close => -1,
        _ => 0,
    }
}
