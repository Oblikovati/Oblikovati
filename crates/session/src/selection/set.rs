// SPDX-License-Identifier: GPL-2.0-only
use super::{SelectionFilter, SelectionKind};

/// A typed handle supplies semantic identity through equality, independent of hit order.
pub trait Selectable: Eq {
    /// Classify a handle for selection filtering; example: return `SelectionKind::Edge`.
    fn selection_kind(&self) -> SelectionKind;
}
/// One ordered, unique selection, independent of model history and GPU handles.
#[derive(Clone, Debug)]
pub struct SelectionSet<T> {
    items: Vec<T>,
    filter: SelectionFilter,
}
impl<T> Default for SelectionSet<T> {
    fn default() -> Self {
        Self {
            items: Vec::new(),
            filter: SelectionFilter::All,
        }
    }
}
impl<T: Selectable> SelectionSet<T> {
    /// Change future additions without pruning current picks; example: `set.set_filter(filter)`.
    pub fn set_filter(&mut self, filter: SelectionFilter) {
        self.filter = filter;
    }
    /// Read the declared tool/context restriction; example: `set.filter().is_restricted()`.
    pub fn filter(&self) -> &SelectionFilter {
        &self.filter
    }
    /// Append one accepted identity once; example: `let changed = set.add(edge)`.
    pub fn add(&mut self, item: T) -> bool {
        if !self.filter.accepts(item.selection_kind()) || self.contains(&item) {
            return false;
        }
        self.items.push(item);
        true
    }
    /// Test handle identity, including its owner; example: `set.contains(&edge)`.
    pub fn contains(&self, item: &T) -> bool {
        self.items.contains(item)
    }
    /// Remove one identity without reordering survivors; example: `set.remove(&edge)`.
    pub fn remove(&mut self, item: &T) -> bool {
        let Some(index) = self.items.iter().position(|picked| picked == item) else {
            return false;
        };
        self.items.remove(index);
        true
    }
    /// Shift/Ctrl membership rule: removal bypasses the current filter (#1222).
    /// Example: `set.toggle(edge)`.
    pub fn toggle(&mut self, item: T) -> bool {
        if self.remove(&item) {
            return true;
        }
        self.add(item)
    }
    /// Clear transient picks, retaining the tool's declared filter; example: `set.clear()`.
    pub fn clear(&mut self) {
        self.items.clear();
    }
    /// Borrow ordered picks without granting mutation; example: `set.items().len()`.
    pub fn items(&self) -> &[T] {
        &self.items
    }
    /// Read the oldest surviving pick; example: `set.first()`.
    pub fn first(&self) -> Option<&T> {
        self.items.first()
    }
}
