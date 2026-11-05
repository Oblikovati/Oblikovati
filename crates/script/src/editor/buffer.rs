// SPDX-License-Identifier: GPL-2.0-only
use super::EditorPosition;
pub(super) struct SourceBuffer {
    pub lines: Vec<Vec<char>>,
}
impl SourceBuffer {
    pub fn new(source: &str) -> Self {
        Self {
            lines: source
                .split('\n')
                .map(|line| line.strip_suffix('\r').unwrap_or(line).chars().collect())
                .collect(),
        }
    }
    pub fn text(&self) -> String {
        self.lines
            .iter()
            .map(|line| line.iter().collect::<String>())
            .collect::<Vec<_>>()
            .join("\n")
    }
    pub fn end(&self) -> EditorPosition {
        EditorPosition {
            line: self.lines.len() - 1,
            column: self.lines.last().expect("one editor line").len(),
        }
    }
    pub fn clamp(&self, position: EditorPosition) -> EditorPosition {
        if position.line >= self.lines.len() {
            return self.end();
        }
        EditorPosition {
            line: position.line,
            column: position.column.min(self.lines[position.line].len()),
        }
    }
    pub fn range(&self, first: EditorPosition, last: EditorPosition) -> String {
        if first.line == last.line {
            return self.lines[first.line][first.column..last.column]
                .iter()
                .collect();
        }
        let mut segments = vec![
            self.lines[first.line][first.column..]
                .iter()
                .collect::<String>(),
        ];
        for line in &self.lines[first.line + 1..last.line] {
            segments.push(line.iter().collect());
        }
        segments.push(self.lines[last.line][..last.column].iter().collect());
        segments.join("\n")
    }
    pub fn replace(
        &mut self,
        first: EditorPosition,
        last: EditorPosition,
        source: &str,
    ) -> EditorPosition {
        let prefix = self.lines[first.line][..first.column].to_vec();
        let tail = self.lines[last.line][last.column..].to_vec();
        let mut replacement: Vec<Vec<char>> = source
            .split('\n')
            .map(|line| line.chars().collect())
            .collect();
        replacement[0].splice(..0, prefix);
        replacement
            .last_mut()
            .expect("one replacement line")
            .extend(tail);
        self.lines.splice(first.line..=last.line, replacement);
        advance(first, source)
    }
    pub fn left(&self, caret: EditorPosition) -> EditorPosition {
        if caret.column > 0 {
            return EditorPosition {
                column: caret.column - 1,
                ..caret
            };
        }
        if caret.line == 0 {
            return caret;
        }
        EditorPosition {
            line: caret.line - 1,
            column: self.lines[caret.line - 1].len(),
        }
    }
    pub fn right(&self, caret: EditorPosition) -> EditorPosition {
        if caret.column < self.lines[caret.line].len() {
            return EditorPosition {
                column: caret.column + 1,
                ..caret
            };
        }
        if caret.line + 1 == self.lines.len() {
            return caret;
        }
        EditorPosition {
            line: caret.line + 1,
            column: 0,
        }
    }
}
pub(super) fn advance(start: EditorPosition, source: &str) -> EditorPosition {
    let mut caret = start;
    for character in source.chars() {
        if character == '\n' {
            caret.line += 1;
            caret.column = 0;
        } else {
            caret.column += 1;
        }
    }
    caret
}
