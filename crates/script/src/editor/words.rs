// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[derive(Clone, Copy, PartialEq, Eq)]
enum SourceCharacter {
    Blank,
    Identifier,
    Other,
}
fn character_class(character: char) -> SourceCharacter {
    match character {
        ' ' | '\t' => SourceCharacter::Blank,
        character if is_identifier(character) => SourceCharacter::Identifier,
        _ => SourceCharacter::Other,
    }
}
impl SourceBuffer {
    fn word_left(&self, caret: EditorPosition) -> EditorPosition {
        if caret.column == 0 {
            return self.left(caret);
        }
        let line = &self.lines[caret.line];
        let mut column = caret.column;
        while column > 0 && character_class(line[column - 1]) == SourceCharacter::Blank {
            column -= 1;
        }
        if column == 0 {
            return EditorPosition { column, ..caret };
        }
        let class = character_class(line[column - 1]);
        while column > 0 && character_class(line[column - 1]) == class {
            column -= 1;
        }
        EditorPosition { column, ..caret }
    }
    fn word_right(&self, caret: EditorPosition) -> EditorPosition {
        let line = &self.lines[caret.line];
        let class = class_at(line, caret.column);
        let mut column = caret.column;
        while column < line.len()
            && class != SourceCharacter::Blank
            && character_class(line[column]) == class
        {
            column += 1;
        }
        if column == line.len() {
            return self.right(EditorPosition { column, ..caret });
        }
        while column < line.len() && character_class(line[column]) == SourceCharacter::Blank {
            column += 1;
        }
        EditorPosition { column, ..caret }
    }
    fn identifier_bounds(&self, caret: EditorPosition) -> (EditorPosition, EditorPosition) {
        let line = &self.lines[caret.line];
        if class_at(line, caret.column) != SourceCharacter::Identifier {
            return (caret, caret);
        }
        let (first, last) = identifier_span(line, caret.column);
        (
            EditorPosition {
                column: first,
                ..caret
            },
            EditorPosition {
                column: last,
                ..caret
            },
        )
    }
}
impl ScriptEditor {
    /// Jump left across blanks and one identifier/punctuation run; example: `editor.move_word_left(true)`.
    pub fn move_word_left(&mut self, extend: bool) {
        self.set_caret(self.buffer.word_left(self.selection.caret), extend);
    }
    /// Jump right across one run and following blanks; example: `editor.move_word_right(false)`.
    pub fn move_word_right(&mut self, extend: bool) {
        self.set_caret(self.buffer.word_right(self.selection.caret), extend);
    }
    /// Select the ASCII Lua identifier under a clamped hit; example: `editor.select_word(position)`.
    pub fn select_word(&mut self, position: EditorPosition) {
        let caret = self.buffer.clamp(position);
        let (anchor, end) = self.buffer.identifier_bounds(caret);
        self.selection = EditorSelection { anchor, caret: end };
        self.goal_column = end.column;
    }
}

fn class_at(line: &[char], column: usize) -> SourceCharacter {
    line.get(column)
        .map(|character| character_class(*character))
        .unwrap_or(SourceCharacter::Blank)
}
fn identifier_span(line: &[char], column: usize) -> (usize, usize) {
    let mut first = column;
    let mut last = column;
    while first > 0 && character_class(line[first - 1]) == SourceCharacter::Identifier {
        first -= 1;
    }
    while last < line.len() && character_class(line[last]) == SourceCharacter::Identifier {
        last += 1;
    }
    (first, last)
}

pub(super) fn is_identifier(character: char) -> bool {
    character.is_ascii_alphanumeric() || character == '_'
}
