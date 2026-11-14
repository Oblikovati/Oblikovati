// SPDX-License-Identifier: GPL-2.0-only
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;

/// A selection category presented by the desktop filter.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiSelectionKind {
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
}
impl GuiSelectionKind {
    /// Return the visible category name. Example: `GuiSelectionKind::Face.label()`.
    pub fn label(self) -> &'static str {
        match self {
            Self::WorkPoint => "Work points",
            Self::WorkAxis => "Work axes",
            Self::PointCloudPoint => "Point cloud points",
            Self::Vertex => "Vertices",
            Self::Edge => "Edges",
            Self::Occurrence => "Occurrences",
            Self::Body => "Bodies",
            Self::Face => "Faces",
            Self::MeshFace => "Mesh faces",
            Self::WorkPlane => "Work planes",
            Self::Profile => "Profiles",
            Self::SketchEntity => "Sketch entities",
        }
    }
}
/// One category and its current pick state.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct GuiSelectionFilterEntry {
    pub kind: GuiSelectionKind,
    pub enabled: bool,
}
/// Ordered pick categories supplied by the document owner.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiSelectionFilter {
    pub entries: Vec<GuiSelectionFilterEntry>,
}
#[derive(Default)]
pub(crate) struct SelectionFilterPanel {
    open: bool,
}
impl DockableWindow for SelectionFilterPanel {}
impl SelectionFilterPanel {
    pub(crate) fn open(&mut self) {
        self.open = true;
    }
    pub(crate) fn draw(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !self.open {
            return;
        }
        let mut open = self.open;
        Self::dockable_window(ui, "Selection Filter", &mut open)
            .size([300., 360.], imgui::Condition::FirstUseEver)
            .build(|| draw_filter(ui, state.selection_filter.as_ref(), actions));
        self.open = open;
    }
}
fn draw_filter(ui: &Ui, filter: Option<&GuiSelectionFilter>, actions: &mut Vec<GuiShellAction>) {
    ui.text("Priority applies to part faces and bodies.");
    ui.text("Other pick categories are not yet complete.");
    let Some(filter) = filter else {
        ui.text("No active document");
        return;
    };
    if ui.button("Reset") {
        actions.push(GuiShellAction::ResetSelectionFilter);
    }
    for (index, entry) in filter.entries.iter().enumerate() {
        draw_entry(ui, entry, index, filter.entries.len(), actions);
    }
    if ui.button("Select All") {
        set_all(filter, true, actions);
    }
    ui.same_line();
    if ui.button("Deselect All") {
        set_all(filter, false, actions);
    }
}
fn draw_entry(
    ui: &Ui,
    entry: &GuiSelectionFilterEntry,
    index: usize,
    count: usize,
    actions: &mut Vec<GuiShellAction>,
) {
    let mut enabled = entry.enabled;
    let clicked = ui.checkbox(entry.kind.label(), &mut enabled);
    if clicked {
        actions.push(GuiShellAction::SetSelectionEnabled {
            kind: entry.kind,
            enabled,
        });
    }
    ui.same_line();
    let up = ui.small_button(format!("Up##selection-{index}"));
    ui.same_line();
    let down = ui.small_button(format!("Down##selection-{index}"));
    if let Some(to) = moved_index(index, count, up, down) {
        actions.push(GuiShellAction::MoveSelectionKind { from: index, to });
    }
}
fn moved_index(index: usize, count: usize, up: bool, down: bool) -> Option<usize> {
    if up {
        return index.checked_sub(1);
    }
    if down && index.checked_add(1).is_some_and(|next| next < count) {
        return Some(index + 1);
    }
    None
}
fn set_all(filter: &GuiSelectionFilter, enabled: bool, actions: &mut Vec<GuiShellAction>) {
    actions.extend(
        filter
            .entries
            .iter()
            .filter(|entry| entry.enabled != enabled)
            .map(|entry| GuiShellAction::SetSelectionEnabled {
                kind: entry.kind,
                enabled,
            }),
    );
}
#[cfg(test)]
mod tests {
    use super::*;
    struct SelectionPanelFixture {
        context: imgui::Context,
        panel: SelectionFilterPanel,
        state: GuiCommandState,
    }
    impl SelectionPanelFixture {
        fn new() -> Self {
            let mut context = imgui::Context::create();
            context.set_ini_filename(None);
            context.io_mut().display_size = [800., 600.];
            context.io_mut().delta_time = 1. / 60.;
            context.fonts().build_rgba32_texture();
            let mut panel = SelectionFilterPanel::default();
            panel.open();
            let state = GuiCommandState {
                selection_filter: Some(GuiSelectionFilter {
                    entries: vec![GuiSelectionFilterEntry {
                        kind: GuiSelectionKind::Face,
                        enabled: true,
                    }],
                }),
                ..Default::default()
            };
            Self {
                context,
                panel,
                state,
            }
        }
        fn frame(&mut self, events: &[crate::GuiInputEvent]) -> Vec<GuiShellAction> {
            for event in events {
                crate::input::apply_input(self.context.io_mut(), event, [1.; 2]);
            }
            let ui = self.context.frame();
            let mut actions = Vec::new();
            self.panel.draw(ui, &self.state, &mut actions);
            self.context.render();
            actions
        }
    }
    #[test]
    fn a_filter_checkbox_emits_the_requested_category_change() {
        let mut fixture = SelectionPanelFixture::new();
        fixture.frame(&[]);
        use crate::{GuiInputEvent, GuiPointerButton};
        fixture.frame(&[GuiInputEvent::PointerMoved([80., 150.])]);
        fixture.frame(&[GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed: true,
        }]);
        let actions = fixture.frame(&[GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed: false,
        }]);
        assert_eq!(
            actions,
            vec![GuiShellAction::SetSelectionEnabled {
                kind: GuiSelectionKind::Face,
                enabled: false
            }]
        );
    }
    #[test]
    fn reset_button_emits_a_single_reset_action() {
        let mut fixture = SelectionPanelFixture::new();
        fixture.frame(&[]);
        use crate::{GuiInputEvent, GuiPointerButton};
        fixture.frame(&[GuiInputEvent::PointerMoved([80., 130.])]);
        fixture.frame(&[GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed: true,
        }]);
        let actions = fixture.frame(&[GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed: false,
        }]);
        assert_eq!(actions, vec![GuiShellAction::ResetSelectionFilter]);
    }
    #[test]
    fn the_title_bar_close_button_closes_the_filter() {
        let mut fixture = SelectionPanelFixture::new();
        fixture.frame(&[]);
        use crate::{GuiInputEvent, GuiPointerButton};
        fixture.frame(&[GuiInputEvent::PointerMoved([347., 70.])]);
        fixture.frame(&[GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed: true,
        }]);
        fixture.frame(&[GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed: false,
        }]);
        assert!(!fixture.panel.open);
    }
    #[test]
    fn reordering_refuses_the_first_and_last_bounds() {
        assert_eq!(moved_index(0, 12, true, false), None);
        assert_eq!(moved_index(11, 12, false, true), None);
        assert_eq!(moved_index(5, 12, true, false), Some(4));
        assert_eq!(moved_index(5, 12, false, true), Some(6));
    }
    #[test]
    fn select_all_emits_only_the_required_changes() {
        let filter = GuiSelectionFilter {
            entries: vec![
                GuiSelectionFilterEntry {
                    kind: GuiSelectionKind::Face,
                    enabled: true,
                },
                GuiSelectionFilterEntry {
                    kind: GuiSelectionKind::Edge,
                    enabled: false,
                },
            ],
        };
        let mut actions = Vec::new();
        set_all(&filter, true, &mut actions);
        assert_eq!(
            actions,
            vec![GuiShellAction::SetSelectionEnabled {
                kind: GuiSelectionKind::Edge,
                enabled: true
            }]
        );
    }
}
