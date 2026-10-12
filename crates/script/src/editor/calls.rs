// SPDX-License-Identifier: GPL-2.0-only
use super::{EditorPosition, ScriptEditor, words::is_identifier};
impl ScriptEditor {
    /// Read the nearest lexical call name on the caret's line; example: `editor.enclosing_call()`.
    /// Like Go, this is not a Lua parser and does not skip strings or comments.
    pub fn enclosing_call(&self) -> Option<Vec<String>> {
        let caret = self.selection.caret;
        let line = &self.buffer.lines[caret.line];
        let open = enclosing_open(line, caret.column)?;
        dotted_chain(line, open)
    }
    /// Read the hovered ASCII identifier's dotted prefix; example: `editor.method_chain_at(position)`.
    /// Out-of-range hits return `None`; the position can be just after an identifier.
    pub fn method_chain_at(&self, position: EditorPosition) -> Option<Vec<String>> {
        let line = self.buffer.lines.get(position.line)?;
        let end = hover_word_end(line, position.column)?;
        dotted_chain(line, end)
    }
}
fn hover_word_end(line: &[char], column: usize) -> Option<usize> {
    if column > line.len() {
        return None;
    }
    let mut end = column;
    while end < line.len() && is_identifier(line[end]) {
        end += 1;
    }
    (end > 0 && is_identifier(line[end - 1])).then_some(end)
}
fn enclosing_open(line: &[char], column: usize) -> Option<usize> {
    let mut depth: usize = 0;
    for index in (0..column.min(line.len())).rev() {
        match line[index] {
            ')' | '}' => depth += 1,
            '(' | '{' if depth == 0 => return Some(index),
            '(' | '{' => depth -= 1,
            _ => (),
        }
    }
    None
}
fn dotted_chain(line: &[char], mut end: usize) -> Option<Vec<String>> {
    let mut chain: Vec<String> = Vec::new();
    loop {
        let start = identifier_start(line, end);
        if start == end {
            break;
        }
        chain.push(line[start..end].iter().collect());
        if start == 0 || line[start - 1] != '.' {
            break;
        }
        end = start - 1;
    }
    chain.reverse();
    (!chain.is_empty()).then_some(chain)
}
pub(super) fn identifier_start(line: &[char], mut end: usize) -> usize {
    while end > 0 && is_identifier(line[end - 1]) {
        end -= 1;
    }
    end
}
