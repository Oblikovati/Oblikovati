// SPDX-License-Identifier: GPL-2.0-only
use super::{SelectionFilter, SelectionKind};
use SelectionKind::*;
use std::collections::BTreeSet;

// Preserve the Go snap-before-depth category priority, independent of enum ordinals (#1222).
const DEFAULT_PICK_ORDER: [SelectionKind; 12] = [
    WorkPoint,
    WorkAxis,
    PointCloudPoint,
    Vertex,
    Edge,
    Occurrence,
    Body,
    Face,
    MeshFace,
    WorkPlane,
    Profile,
    SketchEntity,
];
/// Ribbon presets write the same state as the Selection Filter window.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SelectionPriority {
    General,
    Part,
    Face,
    Edge,
}
/// User-owned enabled categories and priority order; tools retain their own restriction.
#[derive(Clone, Debug)]
pub struct AmbientSelectionFilter {
    enabled: BTreeSet<SelectionKind>,
    order: Vec<SelectionKind>,
}
impl Default for AmbientSelectionFilter {
    fn default() -> Self {
        Self {
            enabled: DEFAULT_PICK_ORDER.into_iter().collect(),
            order: DEFAULT_PICK_ORDER.to_vec(),
        }
    }
}
impl AmbientSelectionFilter {
    /// Inspect category enablement; example: `ambient.enabled(SelectionKind::Face)`.
    pub fn enabled(&self, kind: SelectionKind) -> bool {
        self.enabled.contains(&kind)
    }
    /// Change one category without changing rank; example: `ambient.set_enabled(SelectionKind::Face, false)`.
    pub fn set_enabled(&mut self, kind: SelectionKind, enabled: bool) {
        if enabled {
            self.enabled.insert(kind);
            return;
        }
        self.enabled.remove(&kind);
    }
    /// Borrow the ordered window categories; example: `ambient.order()[0]`.
    pub fn order(&self) -> &[SelectionKind] {
        &self.order
    }
    /// Reorder a category, ignoring invalid or equal positions; example: `ambient.move_kind(7, 0)`.
    pub fn move_kind(&mut self, from: usize, to: usize) {
        if from >= self.order.len() || to >= self.order.len() || from == to {
            return;
        }
        let kind = self.order.remove(from);
        self.order.insert(to, kind);
    }
    /// Rank unknown categories after every window category; example: `ambient.rank(SelectionKind::Face)`.
    pub fn rank(&self, kind: SelectionKind) -> usize {
        self.order
            .iter()
            .position(|row| *row == kind)
            .unwrap_or(self.order.len())
    }
    /// Enable all window categories; example: `ambient.enable_all()`.
    pub fn enable_all(&mut self) {
        self.enabled.extend(self.order.iter().copied());
    }
    /// Disable all window categories; example: `ambient.disable_all()`.
    pub fn disable_all(&mut self) {
        for kind in &self.order {
            self.enabled.remove(kind);
        }
    }
    /// Build the Go all/none/subset filter, considering only window categories.
    /// Example: `ambient.filter().accepts(SelectionKind::Edge)`.
    pub fn filter(&self) -> SelectionFilter {
        let kinds: Vec<SelectionKind> = self
            .order
            .iter()
            .copied()
            .filter(|kind| self.enabled(*kind))
            .collect();
        match kinds.len() {
            0 => SelectionFilter::None,
            count if count == self.order.len() => SelectionFilter::All,
            _ => SelectionFilter::only(kinds),
        }
    }
    /// Apply one ribbon preset without resetting the user's rank order.
    /// Example: `ambient.apply_priority(SelectionPriority::Part)`.
    pub fn apply_priority(&mut self, priority: SelectionPriority) {
        if priority == SelectionPriority::General {
            self.enable_all();
            return;
        }
        let accepted = match priority {
            SelectionPriority::Part => SelectionFilter::only([Body, Occurrence]),
            SelectionPriority::Face => SelectionFilter::only([Face]),
            SelectionPriority::Edge => SelectionFilter::only([Edge]),
            SelectionPriority::General => SelectionFilter::All,
        };
        for kind in self.order.clone() {
            self.set_enabled(kind, accepted.accepts(kind));
        }
    }
    /// A running tool or explicit restriction takes precedence over ambient settings.
    /// Example: `ambient.pick_filter(false, set.filter())`.
    pub fn pick_filter(&self, tool_active: bool, declared: &SelectionFilter) -> SelectionFilter {
        if tool_active || declared.is_restricted() {
            return declared.clone();
        }
        self.filter()
    }
}
