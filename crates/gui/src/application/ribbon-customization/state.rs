// SPDX-License-Identifier: GPL-2.0-only
//! Session-local applied commands and a separate editable draft per ribbon tab.
use crate::{GuiSketchConstraint, GuiSketchTool};
use crate::{application::viewport_orientation::ZoomAction, ribbon::RibbonTab};
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
    CancelSketch,
    FinishSketch,
    Zoom(ZoomAction),
}
impl Command {
    pub(super) fn id(self) -> &'static str {
        match self {
            Self::Separator => "separator",
            Self::Sketches => "sketch-visibility",
            Self::Annotations => "annotations-visibility",
            Self::Measure => "measure",
            Self::Dimension => "dimension",
            Self::CancelSketch => "sketch-cancel",
            Self::FinishSketch => "sketch-finish",
            Self::Zoom(ZoomAction::Zoom) => "view-zoom",
            Self::Zoom(ZoomAction::All) => "view-zoom-all",
            Self::Zoom(ZoomAction::Window) => "view-zoom-window",
            Self::Zoom(ZoomAction::Selected) => "view-zoom-selected",
            Self::Tool(tool) => match tool {
                GuiSketchTool::Line => "sketch-line",
                GuiSketchTool::MidpointLine => "sketch-midpoint-line",
                GuiSketchTool::Rectangle => "sketch-rectangle",
                GuiSketchTool::Circle => "sketch-circle",
                GuiSketchTool::Point => "sketch-point",
                GuiSketchTool::CenterPointArc => "sketch-center-arc",
                GuiSketchTool::Ellipse => "sketch-ellipse",
                GuiSketchTool::Fillet => "sketch-fillet",
                GuiSketchTool::Text => "sketch-text",
                GuiSketchTool::CenterRectangle => "sketch-center-rectangle",
                GuiSketchTool::ThreePointRectangle => "sketch-three-point-rectangle",
                GuiSketchTool::ThreePointCenterRectangle => "sketch-three-point-center-rectangle",
                GuiSketchTool::ThreePointCircle => "sketch-three-point-circle",
                GuiSketchTool::ThreePointArc => "sketch-three-point-arc",
            },
            Self::Constraint(constraint) => match constraint {
                GuiSketchConstraint::Horizontal => "constraint-horizontal",
                GuiSketchConstraint::Vertical => "constraint-vertical",
                GuiSketchConstraint::Coincident => "constraint-coincident",
                GuiSketchConstraint::Fix => "constraint-fix",
                GuiSketchConstraint::Parallel => "constraint-parallel",
                GuiSketchConstraint::Perpendicular => "constraint-perpendicular",
                GuiSketchConstraint::Collinear => "constraint-collinear",
                GuiSketchConstraint::Equal => "constraint-equal",
                GuiSketchConstraint::Concentric => "constraint-concentric",
                GuiSketchConstraint::Tangent => "constraint-tangent",
            },
        }
    }
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
            Self::CancelSketch => "Cancel Sketch",
            Self::FinishSketch => "Finish Sketch",
            Self::Zoom(zoom) => zoom.label(),
        }
    }
}
pub(super) const COMMANDS: [Command; 35] = [
    Command::Separator,
    Command::Tool(GuiSketchTool::Line),
    Command::Tool(GuiSketchTool::MidpointLine),
    Command::Sketches,
    Command::Annotations,
    Command::Tool(GuiSketchTool::Circle),
    Command::Measure,
    Command::Tool(GuiSketchTool::Rectangle),
    Command::Tool(GuiSketchTool::Point),
    Command::Tool(GuiSketchTool::CenterPointArc),
    Command::Tool(GuiSketchTool::Ellipse),
    Command::Tool(GuiSketchTool::Fillet),
    Command::Tool(GuiSketchTool::Text),
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
    Command::CancelSketch,
    Command::FinishSketch,
    Command::Zoom(ZoomAction::Zoom),
    Command::Zoom(ZoomAction::All),
    Command::Zoom(ZoomAction::Window),
    Command::Zoom(ZoomAction::Selected),
];
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(super) enum Category {
    #[default]
    All,
    Macros,
    Sketch,
    View,
    ExitSketch,
}
impl Category {
    pub fn label(self) -> &'static str {
        match self {
            Self::All => "All Commands",
            Self::Macros => "Macros",
            Self::Sketch => "Sketch Tab",
            Self::View => "View Tab",
            Self::ExitSketch => "Exit 2D Sketch Tab",
        }
    }
    fn contains(self, command: Command) -> bool {
        match self {
            Self::All => true,
            Self::Macros => command == Command::Separator,
            Self::Sketch => matches!(
                command,
                Command::Separator | Command::Tool(_) | Command::Constraint(_) | Command::Dimension
            ),
            Self::View => matches!(
                command,
                Command::Separator | Command::Sketches | Command::Zoom(_)
            ),
            Self::ExitSketch => matches!(
                command,
                Command::Separator | Command::CancelSketch | Command::FinishSketch
            ),
        }
    }
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct Entry {
    pub command: Command,
    pub large: bool,
    pub text: bool,
    pub expanded: bool,
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
    pub category: Category,
    pub source_descending: bool,
}
impl Draft {
    pub fn load_applied(&mut self, settings: BTreeMap<RibbonTab, Vec<Entry>>) {
        self.applied = settings;
        self.open(self.target);
    }
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
        self.category = Category::All;
        self.source_descending = false;
    }
    pub fn rows(&self) -> &[Entry] {
        self.staged.get(&self.target).map_or(&[], Vec::as_slice)
    }
    pub fn applied(&self, tab: RibbonTab) -> &[Entry] {
        self.applied.get(&tab).map_or(&[], Vec::as_slice)
    }
    pub fn snapshot(&self) -> &BTreeMap<RibbonTab, Vec<Entry>> {
        &self.staged
    }
    pub fn applied_snapshot(&self) -> &BTreeMap<RibbonTab, Vec<Entry>> {
        &self.applied
    }
    pub fn relocate(&mut self, tab: RibbonTab, command: Command, expanded: bool) {
        for lists in [&mut self.applied, &mut self.staged, &mut self.removed] {
            if let Some(entries) = lists.get_mut(&tab) {
                for entry in entries.iter_mut().filter(|entry| entry.command == command) {
                    entry.expanded = expanded;
                }
            }
        }
    }
    pub fn import(&mut self, staged: BTreeMap<RibbonTab, Vec<Entry>>) {
        self.changed_tabs.extend(self.staged.keys().copied());
        self.changed_tabs.extend(staged.keys().copied());
        self.changed_tabs.extend(self.applied.keys().copied());
        self.changed_tabs.insert(self.target);
        self.staged = staged;
        self.removed.clear();
        self.selected = None;
        self.target_extra.clear();
        self.target_extent = None;
        self.source = None;
        self.source_extra.clear();
        self.source_extent = None;
        self.normalize();
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
        let mut additional = std::mem::take(&mut self.source_extra);
        additional.retain(|command| self.available(*command));
        self.source = self.source.filter(|command| self.available(*command));
        if self.source.is_none() && !additional.is_empty() {
            self.source = Some(additional.remove(0));
        }
        self.source_extra = additional;
        self.source_extent = None;
    }
    pub fn available(&self, command: Command) -> bool {
        (command == Command::Separator || !self.rows().iter().any(|row| row.command == command))
            && self.category.contains(command)
    }
    pub fn set_category(&mut self, category: Category) {
        self.category = category;
        self.source = self.source.map(|command| {
            if self.available(command) {
                command
            } else {
                Command::Separator
            }
        });
        self.source_extra.clear();
        self.source_extent = None;
    }
    pub fn available_commands(&self) -> Vec<Command> {
        let mut commands: Vec<_> = COMMANDS
            .into_iter()
            .filter(|command| self.available(*command))
            .collect();
        commands.sort_by(|left, right| {
            match (*left == Command::Separator, *right == Command::Separator) {
                (true, false) => std::cmp::Ordering::Less,
                (false, true) => std::cmp::Ordering::Greater,
                _ if self.source_descending => sort_name(*right).cmp(&sort_name(*left)),
                _ => sort_name(*left).cmp(&sort_name(*right)),
            }
        });
        commands
    }
    pub fn reverse_source_order(&mut self) {
        self.source_descending = !self.source_descending;
        self.source_extent = None;
    }
    pub fn add(&mut self) {
        let Some(command) = self.source.filter(|command| self.available(*command)) else {
            return;
        };
        let commands: Vec<_> = self
            .available_commands()
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
    pub fn select_visible_source(&mut self, visible: &[Command]) {
        let Some(first) = visible.first().copied() else {
            return;
        };
        let anchor = self
            .source
            .filter(|command| visible.contains(command))
            .unwrap_or(first);
        self.source = Some(anchor);
        self.source_extra = visible
            .iter()
            .copied()
            .filter(|command| *command != anchor)
            .collect();
        self.source_extent = None;
    }
    pub fn select_all_target(&mut self) {
        let count = self.rows().len();
        if count == 0 {
            return;
        }
        let anchor = self.selected.filter(|index| *index < count).unwrap_or(0);
        self.selected = Some(anchor);
        self.target_extra = (0..count).filter(|index| *index != anchor).collect();
        self.target_extent = None;
    }
    pub fn extend_source(&mut self, direction: isize) {
        let Some(anchor) = self.source else {
            return;
        };
        let available = self.available_commands();
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
            expanded: false,
        }
    }
}

fn sort_name(command: Command) -> String {
    command
        .label()
        .chars()
        .filter(|character| character.is_alphanumeric())
        .flat_map(char::to_lowercase)
        .collect()
}

#[cfg(test)]
#[path = "state_tests.rs"]
mod tests;
