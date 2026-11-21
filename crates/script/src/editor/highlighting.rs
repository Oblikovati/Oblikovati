// SPDX-License-Identifier: GPL-2.0-only
use super::ScriptEditor;
use crate::syntax::{LuaLineState, LuaToken, tokenize_line};
impl ScriptEditor {
    /// Highlight a half-open visible line range using preceding source context.
    /// Example: `editor.highlight_lines(10, 20)?` accounts for long comments opened above line 10.
    pub fn highlight_lines(&self, first: usize, last: usize) -> Result<Vec<Vec<LuaToken>>, String> {
        if first > last || last > self.line_count() {
            return Err(format!(
                "highlight lines {first}..{last}: expected ordered bounds within 0..={}",
                self.line_count()
            ));
        }
        let mut state = LuaLineState::Plain;
        let mut highlighted = Vec::with_capacity(last - first);
        for (index, characters) in self.buffer.lines[..last].iter().enumerate() {
            let line: String = characters.iter().collect();
            let (tokens, next) = tokenize_line(&line, state);
            state = next;
            if index >= first {
                highlighted.push(tokens);
            }
        }
        Ok(highlighted)
    }
}
