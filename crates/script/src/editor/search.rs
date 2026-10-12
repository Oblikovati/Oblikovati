// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::rc::{Rc, Weak};
/// A source-generation-bound occurrence. Coordinates are readable, but matches cannot be forged.
#[derive(Clone, Debug)]
pub struct EditorMatch {
    start: EditorPosition,
    end: EditorPosition,
    epoch: Weak<()>,
}
impl EditorMatch {
    /// Read the first character position; example: `highlight.start()`.
    pub fn start(&self) -> EditorPosition {
        self.start
    }
    /// Read the half-open end position; example: `highlight.end()`.
    pub fn end(&self) -> EditorPosition {
        self.end
    }
}
impl ScriptEditor {
    /// Find non-overlapping, literal, case-sensitive line occurrences; example: `editor.find("name")`.
    pub fn find(&self, query: &str) -> Vec<EditorMatch> {
        let query: Vec<char> = query.chars().collect();
        if query.is_empty() {
            return Vec::new();
        }
        let mut matches = Vec::new();
        for (line, characters) in self.buffer.lines.iter().enumerate() {
            append_matches(&mut matches, line, characters, &query, &self.source_epoch);
        }
        matches
    }
    /// Select only a match from this unchanged source; example: `editor.select_match(&hit)?;`.
    pub fn select_match(&mut self, matched: &EditorMatch) -> Result<(), String> {
        if !current_epoch(&matched.epoch, &self.source_epoch) {
            return Err(format!(
                "editor match {:?}..{:?}: expected this editor's current source generation",
                matched.start, matched.end
            ));
        }
        self.selection = EditorSelection {
            anchor: matched.start,
            caret: matched.end,
        };
        self.goal_column = matched.end.column;
        Ok(())
    }
    /// Replace whole-source occurrences as one edit; query may include line breaks, matching Go.
    /// Example: `editor.replace_all("old", "new")` returns the replacement count.
    pub fn replace_all(&mut self, query: &str, replacement: &str) -> usize {
        if query.is_empty() {
            return 0;
        }
        let source = self.text();
        let count = source.matches(query).count();
        if count == 0 {
            return 0;
        }
        self.replace(
            EditorPosition::default(),
            self.buffer.end(),
            &source.replace(query, replacement),
        );
        count
    }
}
fn append_matches(
    matches: &mut Vec<EditorMatch>,
    line: usize,
    characters: &[char],
    query: &[char],
    epoch: &Rc<()>,
) {
    let mut column = 0;
    while column + query.len() <= characters.len() {
        if &characters[column..column + query.len()] == query {
            matches.push(EditorMatch {
                start: EditorPosition { line, column },
                end: EditorPosition {
                    line,
                    column: column + query.len(),
                },
                epoch: Rc::downgrade(epoch),
            });
            column += query.len();
        } else {
            column += 1;
        }
    }
}
fn current_epoch(saved: &Weak<()>, current: &Rc<()>) -> bool {
    saved
        .upgrade()
        .is_some_and(|epoch| Rc::ptr_eq(&epoch, current))
}
/// Headless Find-bar state; recomputes invalidated matches before selecting a new hit.
#[derive(Default)]
pub struct EditorSearch {
    query: String,
    matches: Vec<EditorMatch>,
    index: usize,
    epoch: Weak<()>,
}
impl EditorSearch {
    /// Change the query and select its first hit; example: `find.set_query(&mut editor, "name")`.
    pub fn set_query(&mut self, editor: &mut ScriptEditor, query: &str) {
        self.query = query.into();
        self.refresh(editor);
    }
    /// Refresh after opening Find or changing source; example: `find.refresh(&mut editor)`.
    pub fn refresh(&mut self, editor: &mut ScriptEditor) {
        self.matches = editor.find(&self.query);
        self.epoch = Rc::downgrade(&editor.source_epoch);
        self.index = 0;
        self.select_current(editor);
    }
    /// Select the next hit with wrapping; source changes first reset to the new first hit.
    /// Example: `find.next(&mut editor)` returns whether any hit is selected.
    pub fn next(&mut self, editor: &mut ScriptEditor) -> bool {
        self.step(editor, false)
    }
    /// Select the previous hit with wrapping; example: `find.previous(&mut editor)`.
    pub fn previous(&mut self, editor: &mut ScriptEditor) -> bool {
        self.step(editor, true)
    }
    fn step(&mut self, editor: &mut ScriptEditor, backwards: bool) -> bool {
        if !current_epoch(&self.epoch, &editor.source_epoch) {
            self.refresh(editor);
            return !self.matches.is_empty();
        }
        if self.matches.is_empty() {
            return false;
        }
        self.index = if backwards {
            (self.index + self.matches.len() - 1) % self.matches.len()
        } else {
            (self.index + 1) % self.matches.len()
        };
        self.select_current(editor)
    }
    fn select_current(&self, editor: &mut ScriptEditor) -> bool {
        self.matches
            .get(self.index)
            .is_some_and(|matched| editor.select_match(matched).is_ok())
    }
    /// Replace query occurrences and refresh the remaining hits; example: `find.replace_all(&mut editor, text)`.
    pub fn replace_all(&mut self, editor: &mut ScriptEditor, replacement: &str) -> usize {
        let count = editor.replace_all(&self.query, replacement);
        self.refresh(editor);
        count
    }
    /// Copy valid highlight spans; invalidated source yields no stale coordinates.
    /// Example: `find.highlights(&editor)` feeds one native frame.
    pub fn highlights(&self, editor: &ScriptEditor) -> Vec<EditorMatch> {
        if !current_epoch(&self.epoch, &editor.source_epoch) {
            return Vec::new();
        }
        self.matches.clone()
    }
    /// Read the one-based hit index and total, or zero after source changes; example: `find.ordinal(&editor)`.
    pub fn ordinal(&self, editor: &ScriptEditor) -> (usize, usize) {
        if !current_epoch(&self.epoch, &editor.source_epoch) || self.matches.is_empty() {
            return (0, 0);
        }
        (self.index + 1, self.matches.len())
    }
}
