// SPDX-License-Identifier: GPL-2.0-only
//! The Create Block draft belongs to the GUI until the user accepts it.
use crate::{GuiAuthoringContext, GuiCommandState};
use imgui::{Key, Ui};

#[derive(Default)]
pub(crate) struct CreateBlockDialog {
    document: Option<u64>,
    name: String,
    description: String,
    visible: bool,
}
impl CreateBlockDialog {
    pub(crate) fn open(&mut self, state: &GuiCommandState) {
        self.document = state.document;
        self.name = next_name(&state.sketch_block_names);
        self.description.clear();
        self.visible = false;
    }
    pub(crate) fn panel(&mut self, ui: &Ui, state: &GuiCommandState) {
        if self.document != state.document || state.authoring != GuiAuthoringContext::Sketch {
            self.document = None;
        }
        if self.document.is_none() {
            return;
        }
        crate::widget_abi::fit_dialog(ui, "Create Block", [620., 260.], [390., 245.]);
        ui.window("Create Block").collapsible(false).build(|| {
            self.fields(ui);
            self.buttons(ui);
        });
    }
    fn fields(&mut self, ui: &Ui) {
        ui.text("Geometry");
        ui.same_line();
        let unavailable = ui.begin_disabled(true);
        ui.button("Select##block-geometry");
        ui.text("Insert Point");
        ui.same_line();
        ui.button("Select##block-insertion");
        drop(unavailable);
        ui.checkbox("Visibility", &mut self.visible);
        ui.input_text("Block Name", &mut self.name).build();
        ui.input_text("Description", &mut self.description).build();
    }
    fn buttons(&mut self, ui: &Ui) {
        let unavailable = ui.begin_disabled(true);
        ui.button("OK");
        ui.same_line();
        ui.button("Apply");
        drop(unavailable);
        ui.same_line();
        if ui.button("Cancel") || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape)
        {
            self.document = None;
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
    use super::next_name;
    #[test]
    fn proposed_name_skips_existing_definitions() {
        assert_eq!(next_name(&[]), "Block1");
        assert_eq!(next_name(&["Block1".into(), "Block3".into()]), "Block2");
    }
}
