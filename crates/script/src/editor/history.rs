// SPDX-License-Identifier: GPL-2.0-only
use super::{EditorPosition, SourceBuffer, advance};
pub(super) struct SourceChange {
    pub at: EditorPosition,
    pub removed: String,
    pub inserted: String,
}
#[derive(Default)]
pub(super) struct SourceHistory {
    pub done: Vec<SourceChange>,
    pub undone: Vec<SourceChange>,
}
impl SourceChange {
    fn apply(&self, buffer: &mut SourceBuffer, inverse: bool) -> EditorPosition {
        let (removed, inserted) = if inverse {
            (&self.inserted, &self.removed)
        } else {
            (&self.removed, &self.inserted)
        };
        buffer.replace(self.at, advance(self.at, removed), inserted)
    }
    fn coalesce(&mut self, next: &Self) -> bool {
        if self.removed.is_empty() && next.removed.is_empty() {
            return self.merge_insert(next);
        }
        if self.inserted.is_empty() && next.inserted.is_empty() {
            return self.merge_delete(next);
        }
        false
    }
    fn merge_insert(&mut self, next: &Self) -> bool {
        if self.inserted.is_empty()
            || !single_character(&next.inserted)
            || ends_break(&next.inserted)
            || ends_break(&self.inserted)
        {
            return false;
        }
        if next.at != advance(self.at, &self.inserted) {
            return false;
        }
        self.inserted.push_str(&next.inserted);
        true
    }
    fn merge_delete(&mut self, next: &Self) -> bool {
        if self.removed.is_empty() || !single_character(&next.removed) || next.removed == "\n" {
            return false;
        }
        if advance(next.at, &next.removed) == self.at {
            self.at = next.at;
            self.removed = format!("{}{}", next.removed, self.removed);
            return true;
        }
        if next.at != self.at {
            return false;
        }
        self.removed.push_str(&next.removed);
        true
    }
}
impl SourceHistory {
    pub fn record(&mut self, change: SourceChange) {
        self.undone.clear();
        if self
            .done
            .last_mut()
            .is_some_and(|previous| previous.coalesce(&change))
        {
            return;
        }
        self.done.push(change);
    }
    pub fn undo(&mut self, buffer: &mut SourceBuffer) -> Option<EditorPosition> {
        let change = self.done.pop()?;
        let caret = change.apply(buffer, true);
        self.undone.push(change);
        Some(caret)
    }
    pub fn redo(&mut self, buffer: &mut SourceBuffer) -> Option<EditorPosition> {
        let change = self.undone.pop()?;
        let caret = change.apply(buffer, false);
        self.done.push(change);
        Some(caret)
    }
}
fn single_character(source: &str) -> bool {
    source.chars().count() == 1
}
fn ends_break(source: &str) -> bool {
    source.ends_with([' ', '\t', '\n'])
}
