// SPDX-License-Identifier: GPL-2.0-only
//! The Create Block draft belongs to the GUI until the user accepts it.
use crate::dockable_window::DockableWindow;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::{Key, Ui};

pub(crate) fn ribbon_button(
    ui: &Ui,
    state: &GuiCommandState,
    dialog: &mut CreateBlockDialog,
    actions: &mut Vec<GuiShellAction>,
    size: [f32; 2],
    icons: &crate::icons::GuiRibbonIcons,
) {
    let disabled = ui.begin_disabled(!state.sketch_at_ready || state.transaction_open);
    let clicked = if icons.appearance == crate::ribbon_command_layout::RibbonAppearance::Normal {
        crate::widget_abi::button(ui, "Create Block", size)
    } else {
        icons.button(
            ui,
            "Create Block",
            crate::icons::IconGlyph::CreateBlock,
            size,
        )
    };
    if clicked {
        dialog.open(state);
        actions.push(GuiShellAction::SelectBlockGeometry(true));
    }
    drop(disabled);
}

#[derive(Default)]
pub(crate) struct CreateBlockDialog {
    document: Option<u64>,
    name: String,
    description: String,
    visible: bool,
    pending: Option<(String, bool)>,
    dock_state: crate::dockable_lifecycle::DockableState,
}
impl DockableWindow for CreateBlockDialog {}
impl CreateBlockDialog {
    pub(crate) fn open(&mut self, state: &GuiCommandState) {
        self.set_document(state.document);
        self.name = next_name(&state.sketch_block_names);
        self.description.clear();
        self.visible = false;
        self.pending = None;
    }
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Create Block", &self.dock_state)
    }
    fn set_document(&mut self, document: Option<u64>) {
        self.document = document;
        self.dock_state.set_open(document.is_some());
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.document != state.document || state.authoring != GuiAuthoringContext::Sketch {
            if self.document.is_some() {
                actions.push(GuiShellAction::SelectBlockGeometry(false));
            }
            self.set_document(None);
        }
        self.finish_pending(state, actions);
        if self.document.is_none() {
            return;
        }
        Self::fit_dockable(ui, "Create Block", [620., 260.], [390., 245.]);
        let mut open = true;
        Self::dockable_window(ui, "Create Block", &mut open).build(|| {
            self.fields(ui);
            self.buttons(ui, state, actions);
        });
        if !open && self.document.is_some() {
            self.set_document(None);
            actions.push(GuiShellAction::SelectBlockGeometry(false));
        }
    }
    fn fields(&mut self, ui: &Ui) {
        ui.text("Geometry");
        ui.same_line();
        crate::widget_abi::toggle_button(ui, "Select##block-geometry", [0., 0.], true);
        ui.text("Insert Point");
        ui.same_line();
        let unavailable = ui.begin_disabled(true);
        ui.button("Select##block-insertion");
        ui.checkbox("Visibility", &mut self.visible);
        drop(unavailable);
        ui.input_text("Block Name", &mut self.name).build();
        let unavailable = ui.begin_disabled(true);
        ui.input_text("Description", &mut self.description).build();
        drop(unavailable);
    }
    fn buttons(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let action = GuiShellAction::CreateSketchBlock(self.name.clone());
        let enabled = self.pending.is_none() && action.dispatchable(state);
        let unavailable = ui.begin_disabled(!enabled);
        if ui.button("OK") && enabled {
            self.pending = Some((self.name.clone(), true));
            actions.push(action.clone());
        }
        ui.same_line();
        if ui.button("Apply") && enabled {
            self.pending = Some((self.name.clone(), false));
            actions.push(action);
        }
        drop(unavailable);
        ui.same_line();
        if ui.button("Cancel") || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape)
        {
            self.set_document(None);
            actions.push(GuiShellAction::SelectBlockGeometry(false));
        }
    }
    fn finish_pending(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let Some((name, close)) = self.pending.take() else {
            return;
        };
        if state.sketch_block_created.as_deref() != Some(name.as_str()) {
            return;
        }
        if close {
            self.set_document(None);
            actions.push(GuiShellAction::SelectBlockGeometry(false));
        } else {
            self.name = next_name(&state.sketch_block_names);
            self.description.clear();
        }
    }
}
fn next_name(names: &[String]) -> String {
    for ordinal in 1.. {
        let name = format!("Block{ordinal}");
        if !names.contains(&name) {
            return name;
        }
    }
    unreachable!("a finite registry cannot contain every block name")
}

#[cfg(test)]
mod tests {
    use super::{CreateBlockDialog, GuiCommandState, GuiShellAction, next_name};
    #[test]
    fn proposed_name_skips_existing_definitions() {
        assert_eq!(next_name(&[]), "Block1");
        assert_eq!(next_name(&["Block1".into(), "Block3".into()]), "Block2");
    }
    #[test]
    fn owner_acknowledgement_controls_apply_and_close() {
        let mut dialog = CreateBlockDialog {
            document: Some(1),
            pending: Some(("Block1".into(), false)),
            ..Default::default()
        };
        let mut actions = Vec::new();
        let mut state = GuiCommandState {
            sketch_block_names: vec!["Block1".into()],
            ..Default::default()
        };
        dialog.finish_pending(&state, &mut actions);
        assert!(dialog.name.is_empty());
        dialog.pending = Some(("Block1".into(), false));
        state.sketch_block_created = Some("Block1".into());
        dialog.finish_pending(&state, &mut actions);
        assert_eq!(dialog.name, "Block2");
        dialog.pending = Some(("Block1".into(), true));
        dialog.finish_pending(&state, &mut actions);
        assert_eq!(dialog.document, None);
        assert_eq!(actions, vec![GuiShellAction::SelectBlockGeometry(false)]);
    }
}
