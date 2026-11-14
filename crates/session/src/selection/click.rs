// SPDX-License-Identifier: GPL-2.0-only
use super::{Selectable, SelectionSet};

/// Normalized desktop selection intent; native key mapping belongs to the platform layer.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SelectionClickMode {
    Replace,
    Toggle,
}
impl SelectionClickMode {
    /// Shift or Ctrl toggles membership; example: `SelectionClickMode::from_modifiers(true, false)`.
    pub fn from_modifiers(shift: bool, control: bool) -> Self {
        if shift || control {
            return Self::Toggle;
        }
        Self::Replace
    }
}
impl<T: Selectable> SelectionSet<T> {
    /// Apply a resolved click; return the count only when the caller must refresh selection.
    /// Tools own their clicks; example: `set.apply_click(Some(edge), mode, false)`.
    pub fn apply_click(
        &mut self,
        picked: Option<T>,
        mode: SelectionClickMode,
        tool_active: bool,
    ) -> Option<usize> {
        if tool_active {
            return None;
        }
        let changed = match picked {
            None => self.clear_on_empty_click(mode),
            Some(item) if mode == SelectionClickMode::Toggle => self.toggle(item),
            Some(item) => self.replace_click(item),
        };
        changed.then_some(self.items().len())
    }
    fn clear_on_empty_click(&mut self, mode: SelectionClickMode) -> bool {
        if mode == SelectionClickMode::Toggle || self.items().is_empty() {
            return false;
        }
        self.clear();
        true
    }
    fn replace_click(&mut self, item: T) -> bool {
        let cleared = !self.items().is_empty();
        self.clear();
        // A rejected addition can still retire old picks; the observer must see the empty set.
        self.add(item) || cleared
    }
}
