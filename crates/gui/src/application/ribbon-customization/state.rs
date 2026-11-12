// SPDX-License-Identifier: GPL-2.0-only
//! Session-local applied commands and a separate editable draft per ribbon tab.
use crate::ribbon::RibbonTab;
use crate::{GuiSketchConstraint, GuiSketchTool};
use std::collections::{BTreeMap, BTreeSet};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum Command {
    Separator,
    Sketches,
    Annotations,
    Tool(GuiSketchTool),
    Constraint(GuiSketchConstraint),
    Dimension,
    Measure,
}
impl Command {
    pub(super) fn label(self) -> &'static str {
        match self {
            Self::Separator => "<Separator>",
            Self::Sketches => "2D Sketches",
            Self::Annotations => "3D Annotations",
            Self::Tool(GuiSketchTool::Circle) => "Circle (Center Point)",
            Self::Tool(tool) => tool.label(),
            Self::Constraint(constraint) => constraint.label(),
            Self::Dimension => "Dimension",
            Self::Measure => "Measure",
        }
    }
}
pub(super) const COMMANDS: [Command; 26] = [
    Command::Separator,
    Command::Tool(GuiSketchTool::Line),
    Command::Sketches,
    Command::Annotations,
    Command::Tool(GuiSketchTool::Circle),
    Command::Measure,
    Command::Tool(GuiSketchTool::Rectangle),
    Command::Tool(GuiSketchTool::Point),
    Command::Tool(GuiSketchTool::CenterPointArc),
    Command::Tool(GuiSketchTool::Ellipse),
    Command::Tool(GuiSketchTool::CenterRectangle),
    Command::Tool(GuiSketchTool::ThreePointRectangle),
    Command::Tool(GuiSketchTool::ThreePointCenterRectangle),
    Command::Tool(GuiSketchTool::ThreePointCircle),
    Command::Tool(GuiSketchTool::ThreePointArc),
    Command::Constraint(GuiSketchConstraint::Horizontal),
    Command::Constraint(GuiSketchConstraint::Vertical),
    Command::Constraint(GuiSketchConstraint::Coincident),
    Command::Constraint(GuiSketchConstraint::Fix),
    Command::Constraint(GuiSketchConstraint::Parallel),
    Command::Constraint(GuiSketchConstraint::Perpendicular),
    Command::Constraint(GuiSketchConstraint::Collinear),
    Command::Constraint(GuiSketchConstraint::Equal),
    Command::Constraint(GuiSketchConstraint::Concentric),
    Command::Constraint(GuiSketchConstraint::Tangent),
    Command::Dimension,
];
#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct Entry {
    pub command: Command,
    pub large: bool,
    pub text: bool,
}
#[derive(Default)]
pub(super) struct Draft {
    applied: BTreeMap<RibbonTab, Vec<Entry>>,
    staged: BTreeMap<RibbonTab, Vec<Entry>>,
    removed: BTreeMap<RibbonTab, Vec<Entry>>,
    changed_tabs: BTreeSet<RibbonTab>,
    pub target: RibbonTab,
    pub source: Option<Command>,
    pub source_extra: Vec<Command>,
    source_extent: Option<usize>,
    pub selected: Option<usize>,
    pub target_extra: BTreeSet<usize>,
    target_extent: Option<usize>,
    pub sketch_category: bool,
}
impl Draft {
    pub fn open(&mut self, target: RibbonTab) {
        self.staged = self.applied.clone();
        self.removed.clear();
        self.changed_tabs.clear();
        self.target = target;
        self.source = None;
        self.source_extra.clear();
        self.source_extent = None;
        self.selected = None;
        self.target_extra.clear();
        self.target_extent = None;
        self.sketch_category = false;
    }
    pub fn rows(&self) -> &[Entry] {
        self.staged.get(&self.target).map_or(&[], Vec::as_slice)
    }
    pub fn applied(&self, tab: RibbonTab) -> &[Entry] {
        self.applied.get(&tab).map_or(&[], Vec::as_slice)
    }
    pub fn changed(&self) -> bool {
        !self.changed_tabs.is_empty()
    }
    pub fn target_changed(&self) -> bool {
        self.changed_tabs.contains(&self.target)
    }
    pub fn switch(&mut self, target: RibbonTab) {
        self.target = target;
        self.selected = None;
        self.target_extra.clear();
        self.target_extent = None;
        self.source = None;
        self.source_extra.clear();
        self.source_extent = None;
    }
    pub fn available(&self, command: Command) -> bool {
        (command == Command::Separator || !self.rows().iter().any(|row| row.command == command))
            && (!self.sketch_category
                || matches!(
                    command,
                    Command::Separator
                        | Command::Tool(_)
                        | Command::Constraint(_)
                        | Command::Dimension
                ))
    }
    pub fn add(&mut self) {
        let Some(command) = self.source.filter(|command| self.available(*command)) else {
            return;
        };
        let commands: Vec<_> = COMMANDS
            .into_iter()
            .filter(|candidate| {
                (*candidate == command || self.source_extra.contains(candidate))
                    && self.available(*candidate)
            })
            .collect();
        let index = self.selected.map_or(self.rows().len(), |index| {
            (index + 1).min(self.rows().len())
        });
        let entries: Vec<_> = commands
            .iter()
            .map(|command| self.restore_or_default(*command))
            .collect();
        let rows = self.staged.entry(self.target).or_default();
        for (offset, entry) in entries.into_iter().enumerate() {
            rows.insert(index + offset, entry);
        }
        self.selected = Some(index);
        self.target_extra = (index + 1..index + commands.len()).collect();
        self.target_extent = Some(index + commands.len() - 1);
        self.mark_changed();
        self.source_extra.clear();
        self.source_extent = None;
        if command != Command::Separator {
            self.source = None;
        }
    }
    pub fn remove(&mut self) {
        let Some(index) = self.selected.filter(|index| *index < self.rows().len()) else {
            return;
        };
        let Some(rows) = self.staged.get_mut(&self.target) else {
            return;
        };
        let mut indices = self.target_extra.clone();
        indices.insert(index);
        let mut removed = Vec::new();
        for selected in indices.iter().rev() {
            if *selected < rows.len() {
                removed.push(rows.remove(*selected));
            }
        }
        self.selected = (!rows.is_empty()).then_some(index.min(rows.len().saturating_sub(1)));
        self.target_extra.clear();
        self.target_extent = None;
        removed.reverse();
        let first = removed.first().map(|entry| entry.command);
        self.removed.entry(self.target).or_default().extend(removed);
        self.source = first.filter(|command| self.available(*command));
        self.source_extra.clear();
        self.source_extent = None;
        self.mark_changed();
        self.normalize();
    }
    pub fn move_selected(&mut self, delta: isize) {
        if !self.target_extra.is_empty() {
            return;
        }
        let Some(index) = self.selected else {
            return;
        };
        let Some(next) = index
            .checked_add_signed(delta)
            .filter(|next| *next < self.rows().len())
        else {
            return;
        };
        let Some(rows) = self.staged.get_mut(&self.target) else {
            return;
        };
        rows.swap(index, next);
        self.selected = Some(next);
        self.target_extent = None;
        self.mark_changed();
    }
    pub fn set_options(&mut self, index: usize, large: bool, text: bool) {
        let Some(entry) = self
            .staged
            .get_mut(&self.target)
            .and_then(|rows| rows.get_mut(index))
        else {
            return;
        };
        if entry.large != large || entry.text != text {
            entry.large = large;
            entry.text = text;
            self.mark_changed();
        }
    }
    pub fn apply(&mut self) {
        self.normalize();
        self.applied = self.staged.clone();
        self.changed_tabs.clear();
        self.removed.clear();
    }
    pub fn mark_changed(&mut self) {
        self.changed_tabs.insert(self.target);
    }
    pub fn reset_source_selection(&mut self, command: Command) {
        self.source = Some(command);
        self.source_extra.clear();
        self.source_extent = None;
    }
    pub fn reset_target_selection(&mut self, index: usize) {
        self.selected = Some(index);
        self.target_extra.clear();
        self.target_extent = None;
    }
    pub fn extend_source(&mut self, direction: isize) {
        let Some(anchor) = self.source else {
            return;
        };
        let available: Vec<_> = COMMANDS
            .into_iter()
            .filter(|command| self.available(*command))
            .collect();
        let Some(index) = available.iter().position(|command| *command == anchor) else {
            return;
        };
        let Some(next) = self
            .source_extent
            .unwrap_or(index)
            .checked_add_signed(direction)
            .filter(|next| *next < available.len())
        else {
            return;
        };
        self.source_extent = Some(next);
        self.source_extra = (index.min(next)..=index.max(next))
            .filter(|item| *item != index)
            .map(|item| available[item])
            .collect();
    }
    pub fn extend_target(&mut self, direction: isize) {
        let Some(index) = self.selected else {
            return;
        };
        let Some(next) = self
            .target_extent
            .unwrap_or(index)
            .checked_add_signed(direction)
            .filter(|next| *next < self.rows().len())
        else {
            return;
        };
        self.target_extent = Some(next);
        self.target_extra = (index.min(next)..=index.max(next))
            .filter(|item| *item != index)
            .collect();
    }
    fn normalize(&mut self) {
        self.staged.retain(|_, entries| !entries.is_empty());
    }
    fn restore_or_default(&mut self, command: Command) -> Entry {
        if let Some(entries) = self.removed.get_mut(&self.target)
            && let Some(index) = entries.iter().position(|entry| entry.command == command)
        {
            return entries.remove(index);
        }
        Entry {
            command,
            large: false,
            text: false,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn staged_tabs_cancel_together_and_applied_tabs_survive_reopening() {
        let mut draft = Draft::default();
        draft.open(RibbonTab::Model);
        draft.source = Some(Command::Sketches);
        draft.add();
        assert!(draft.changed() && draft.target_changed());
        draft.switch(RibbonTab::Sketch);
        assert!(draft.rows().is_empty() && draft.changed() && !draft.target_changed());
        draft.source = Some(Command::Tool(GuiSketchTool::Circle));
        draft.add();
        draft.open(RibbonTab::Model);
        assert!(!draft.changed() && draft.rows().is_empty());
        draft.source = Some(Command::Sketches);
        draft.add();
        draft.apply();
        draft.open(RibbonTab::Model);
        assert_eq!(draft.rows()[0].command, Command::Sketches);
        assert!(!draft.changed());
    }
    #[test]
    fn transfer_reorder_options_and_reusable_separators_have_one_draft() {
        let mut draft = Draft::default();
        draft.open(RibbonTab::Tools);
        for command in [Command::Sketches, Command::Separator, Command::Annotations] {
            draft.source = Some(command);
            draft.add();
        }
        assert!(draft.available(Command::Separator));
        assert!(!draft.available(Command::Sketches));
        draft.move_selected(-1);
        assert_eq!(draft.rows()[1].command, Command::Annotations);
        draft.set_options(1, true, false);
        draft.apply();
        assert!(draft.applied(RibbonTab::Tools)[1].large);
        assert!(!draft.applied(RibbonTab::Tools)[1].text);
        draft.remove();
        assert!(draft.available(Command::Annotations));
        assert!(draft.changed());
    }
    #[test]
    fn reversed_addition_stays_pending_until_apply_or_cancel() {
        let mut draft = Draft::default();
        draft.open(RibbonTab::Tools);
        draft.source = Some(Command::Sketches);
        draft.add();
        draft.remove();
        assert!(draft.rows().is_empty());
        assert!(draft.changed() && draft.target_changed());
        draft.switch(RibbonTab::Model);
        assert!(draft.changed() && !draft.target_changed());
        draft.apply();
        assert!(!draft.changed());
        draft.switch(RibbonTab::Tools);
        assert!(!draft.target_changed());
    }
    #[test]
    fn sketch_category_contains_supported_sketch_commands_once() {
        let mut draft = Draft::default();
        draft.open(RibbonTab::Sketch);
        draft.sketch_category = true;
        let labels: Vec<_> = COMMANDS
            .iter()
            .copied()
            .filter(|command| draft.available(*command))
            .map(Command::label)
            .collect();
        for expected in [
            "Line",
            "Rectangle",
            "Circle (Center Point)",
            "Horizontal",
            "Dimension",
        ] {
            assert_eq!(labels.iter().filter(|label| **label == expected).count(), 1);
        }
        assert!(!labels.contains(&"Measure"));
    }
    #[test]
    fn multiple_commands_transfer_in_order_and_return_together() {
        let mut draft = Draft::default();
        draft.open(RibbonTab::Tools);
        draft.source = Some(Command::Sketches);
        draft.extend_source(1);
        assert_eq!(draft.source_extra, vec![Command::Annotations]);
        draft.add();
        assert_eq!(
            draft
                .rows()
                .iter()
                .map(|row| row.command)
                .collect::<Vec<_>>(),
            vec![Command::Sketches, Command::Annotations]
        );
        assert_eq!(
            draft.target_extra.iter().copied().collect::<Vec<_>>(),
            vec![1]
        );
        draft.move_selected(1);
        assert_eq!(draft.rows()[0].command, Command::Sketches);
        draft.remove();
        assert!(draft.rows().is_empty());
        assert!(draft.available(Command::Sketches) && draft.available(Command::Annotations));
        assert!(draft.target_changed());
    }
    #[test]
    fn shift_selection_extends_and_shrinks_in_both_directions() {
        let mut draft = Draft::default();
        draft.open(RibbonTab::Tools);
        draft.reset_source_selection(Command::Annotations);
        draft.extend_source(-1);
        assert_eq!(draft.source_extra, vec![Command::Sketches]);
        draft.extend_source(1);
        assert!(draft.source_extra.is_empty());
        draft.extend_source(-1);
        draft.add();
        draft.reset_target_selection(1);
        draft.extend_target(-1);
        assert!(draft.target_extra.contains(&0));
        draft.extend_target(1);
        assert!(draft.target_extra.is_empty());
    }
    #[test]
    fn removed_command_restores_its_options_on_its_own_target() {
        let mut draft = Draft::default();
        draft.open(RibbonTab::Tools);
        draft.reset_source_selection(Command::Sketches);
        draft.add();
        assert!(!draft.rows()[0].large && !draft.rows()[0].text);
        draft.set_options(0, true, true);
        draft.remove();
        draft.switch(RibbonTab::Manage);
        draft.reset_source_selection(Command::Sketches);
        draft.add();
        assert!(!draft.rows()[0].large && !draft.rows()[0].text);
        draft.switch(RibbonTab::Tools);
        draft.reset_source_selection(Command::Sketches);
        draft.add();
        assert!(draft.rows()[0].large && draft.rows()[0].text);
    }
}
