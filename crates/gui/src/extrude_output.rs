// SPDX-License-Identifier: GPL-2.0-only
use super::{ExtrudeDialog, GuiCommandState};
use imgui::Ui;
use oblikovati_model::feature::sketch_prism::PrismBoolean;

pub(crate) fn single_body(state: &GuiCommandState) -> bool {
    state
        .browser
        .as_ref()
        .is_some_and(|browser| browser.bodies.len() == 1)
}

impl ExtrudeDialog {
    pub(super) fn draw_output(&mut self, ui: &Ui, state: &GuiCommandState) -> bool {
        if self.edit_feature.is_some() || !self.reference.numeric() || !single_body(state) {
            return self.operation.take().is_some();
        }
        if !ui.collapsing_header("Output", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            return false;
        }
        crate::theme::property_label(ui, "Operation");
        let previous = self.operation;
        let label = previous.map_or("New Solid".into(), |operation| operation.to_string());
        if let Some(_choices) = ui.begin_combo("##extrude-operation", label) {
            if ui
                .selectable_config("New Solid")
                .selected(previous.is_none())
                .build()
            {
                self.operation = None;
            }
            if ui
                .selectable_config("Join")
                .selected(previous == Some(PrismBoolean::Join))
                .build()
            {
                self.operation = Some(PrismBoolean::Join);
            }
        }
        self.operation != previous
    }
}
