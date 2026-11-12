// SPDX-License-Identifier: GPL-2.0-only
//! Session-local applied commands and a separate editable draft per ribbon tab.
use crate::ribbon::RibbonTab;
use std::collections::BTreeMap;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum Command {
    Separator,
    Sketches,
    Annotations,
    Circle,
    Measure,
}
impl Command {
    pub(super) fn label(self) -> &'static str {
        match self {
            Self::Separator => "<Separator>",
            Self::Sketches => "2D Sketches",
            Self::Annotations => "3D Annotations",
            Self::Circle => "Circle (Center Point)",
            Self::Measure => "Measure",
        }
    }
}
pub(super) const COMMANDS: [Command; 5] = [
    Command::Separator,
    Command::Sketches,
    Command::Annotations,
    Command::Circle,
    Command::Measure,
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
    pub target: RibbonTab,
    pub source: Option<Command>,
    pub selected: Option<usize>,
    pub sketch_category: bool,
}
impl Draft {
    pub fn open(&mut self, target: RibbonTab) {
        self.staged = self.applied.clone();
        self.target = target;
        self.source = None;
        self.selected = None;
        self.sketch_category = false;
    }
    pub fn rows(&self) -> &[Entry] {
        self.staged.get(&self.target).map_or(&[], Vec::as_slice)
    }
    pub fn applied(&self, tab: RibbonTab) -> &[Entry] {
        self.applied.get(&tab).map_or(&[], Vec::as_slice)
    }
    pub fn changed(&self) -> bool {
        self.staged != self.applied
    }
    pub fn target_changed(&self) -> bool {
        self.rows() != self.applied(self.target)
    }
    pub fn switch(&mut self, target: RibbonTab) {
        self.target = target;
        self.selected = None;
        self.source = None;
    }
    pub fn available(&self, command: Command) -> bool {
        (command == Command::Separator || !self.rows().iter().any(|row| row.command == command))
            && (!self.sketch_category || matches!(command, Command::Separator | Command::Circle))
    }
    pub fn add(&mut self) {
        let Some(command) = self.source.filter(|command| self.available(*command)) else {
            return;
        };
        let index = self.selected.map_or(self.rows().len(), |index| {
            (index + 1).min(self.rows().len())
        });
        self.staged.entry(self.target).or_default().insert(
            index,
            Entry {
                command,
                large: false,
                text: true,
            },
        );
        self.selected = Some(index);
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
        rows.remove(index);
        self.selected = (!rows.is_empty()).then_some(index.min(rows.len().saturating_sub(1)));
        self.normalize();
    }
    pub fn move_selected(&mut self, delta: isize) {
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
    }
    pub fn selected_mut(&mut self) -> Option<&mut Entry> {
        let index = self.selected?;
        self.staged.get_mut(&self.target)?.get_mut(index)
    }
    pub fn apply(&mut self) {
        self.normalize();
        self.applied = self.staged.clone();
    }
    fn normalize(&mut self) {
        self.staged.retain(|_, entries| !entries.is_empty());
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
        draft.source = Some(Command::Circle);
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
        draft.selected_mut().unwrap().large = true;
        draft.selected_mut().unwrap().text = false;
        draft.apply();
        assert!(draft.applied(RibbonTab::Tools)[1].large);
        assert!(!draft.applied(RibbonTab::Tools)[1].text);
        draft.remove();
        assert!(draft.available(Command::Annotations));
        assert!(draft.changed());
    }
}
