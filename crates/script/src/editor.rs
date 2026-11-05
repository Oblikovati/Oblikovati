// SPDX-License-Identifier: GPL-2.0-only
//! Unicode editor state; native input and drawing do not own the source or edit history.
mod blocks;
mod brackets;
mod buffer;
mod calls;
mod completion;
pub use completion::{
    CompletionCandidate, CompletionEngine, CompletionKey, CompletionKind, CompletionPopup,
    CompletionView, EditorCompletion,
};
mod highlighting;
mod history;
mod movement;
mod search;
mod words;
use buffer::{SourceBuffer, advance};
use history::{SourceChange, SourceHistory};
pub use search::{EditorMatch, EditorSearch};
use std::rc::Rc;
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, PartialOrd, Ord)]
pub struct EditorPosition {
    pub line: usize,
    pub column: usize,
}
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct EditorSelection {
    pub anchor: EditorPosition,
    pub caret: EditorPosition,
}
impl EditorSelection {
    /// Return document-ordered bounds; example: `let (first, last) = selection.ordered();`.
    pub fn ordered(self) -> (EditorPosition, EditorPosition) {
        (self.anchor.min(self.caret), self.anchor.max(self.caret))
    }
    /// Test whether the caret is unselected; example: `selection.is_empty()`.
    pub fn is_empty(self) -> bool {
        self.anchor == self.caret
    }
}
pub struct ScriptEditor {
    buffer: SourceBuffer,
    selection: EditorSelection,
    history: SourceHistory,
    goal_column: usize,
    source_epoch: Rc<()>,
}
impl ScriptEditor {
    /// Load LF/CRLF text with a start caret; example: `ScriptEditor::new("local x=1")`.
    pub fn new(source: &str) -> Self {
        Self {
            buffer: SourceBuffer::new(source),
            selection: EditorSelection::default(),
            history: SourceHistory::default(),
            goal_column: 0,
            source_epoch: Rc::new(()),
        }
    }
    /// Copy source for Run or persistence; example: `console.run(&editor.text())?;`.
    pub fn text(&self) -> String {
        self.buffer.text()
    }
    /// Replace the loaded script and clear history; example: `editor.set_text(source)`.
    pub fn set_text(&mut self, source: &str) {
        *self = Self::new(source);
    }
    /// Count renderable lines, including an empty document line; example: `editor.line_count()`.
    pub fn line_count(&self) -> usize {
        self.buffer.lines.len()
    }
    /// Read a renderable line or name the invalid index; example: `editor.line(0)?;`.
    pub fn line(&self, index: usize) -> Result<String, String> {
        self.buffer
            .lines
            .get(index)
            .map(|line| line.iter().collect())
            .ok_or_else(|| {
                format!(
                    "editor line {index}: expected index below {}",
                    self.line_count()
                )
            })
    }
    /// Copy the anchored selection; example: `let selection = editor.selection();`.
    pub fn selection(&self) -> EditorSelection {
        self.selection
    }
    /// Copy text for Copy/Cut; example: `clipboard.set(editor.selected_text())`.
    pub fn selected_text(&self) -> String {
        let (first, last) = self.selection.ordered();
        self.buffer.range(first, last)
    }
    /// Query undo availability; example: `editor.can_undo()` enables the toolbar action.
    pub fn can_undo(&self) -> bool {
        !self.history.done.is_empty()
    }
    /// Query redo availability; example: `editor.can_redo()` enables the toolbar action.
    pub fn can_redo(&self) -> bool {
        !self.history.undone.is_empty()
    }
    /// Replace the selection with typed/pasted source; example: `editor.insert("hello")`.
    pub fn insert(&mut self, source: &str) {
        if source.is_empty() {
            return;
        }
        let (first, last) = self.selection.ordered();
        self.replace(first, last, source);
    }
    /// Insert a line break with the starting line's indentation; example: `editor.newline()`.
    pub fn newline(&mut self) {
        let (first, _) = self.selection.ordered();
        let indentation: String = self.buffer.lines[first.line]
            .iter()
            .take_while(|character| **character == ' ' || **character == '\t')
            .collect();
        self.insert(&format!("\n{indentation}"));
    }
    /// Delete selected text or the preceding Unicode character; example: `editor.backspace()`.
    pub fn backspace(&mut self) {
        self.remove(true);
    }
    /// Delete selected text or the following Unicode character; example: `editor.delete()`.
    pub fn delete(&mut self) {
        self.remove(false);
    }
    fn remove(&mut self, backwards: bool) {
        let (mut first, mut last) = self.selection.ordered();
        if self.selection.is_empty() {
            if backwards {
                first = self.buffer.left(first);
            } else {
                last = self.buffer.right(last);
            }
        }
        if first != last {
            self.replace(first, last, "");
        }
    }
    fn replace(&mut self, first: EditorPosition, last: EditorPosition, source: &str) {
        let removed = self.buffer.range(first, last);
        if removed.is_empty() && source.is_empty() {
            return;
        }
        self.source_epoch = Rc::new(());
        let caret = self.buffer.replace(first, last, source);
        self.history.record(SourceChange {
            at: first,
            removed,
            inserted: source.into(),
        });
        self.collapse(caret);
    }
    /// Reverse the latest coalesced edit; example: `editor.undo()`.
    pub fn undo(&mut self) {
        if let Some(caret) = self.history.undo(&mut self.buffer) {
            self.source_epoch = Rc::new(());
            self.collapse(caret);
        }
    }
    /// Reapply the latest undone edit; example: `editor.redo()`.
    pub fn redo(&mut self) {
        if let Some(caret) = self.history.redo(&mut self.buffer) {
            self.source_epoch = Rc::new(());
            self.collapse(caret);
        }
    }
    fn collapse(&mut self, caret: EditorPosition) {
        self.selection = EditorSelection {
            anchor: caret,
            caret,
        };
        self.goal_column = caret.column;
    }
}
