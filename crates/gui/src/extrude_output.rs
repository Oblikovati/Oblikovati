// SPDX-License-Identifier: GPL-2.0-only
use super::{ExtrudeDialog, GuiCommandState, GuiShellAction};
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
        if self.edit_feature.is_some() || !self.reference.numeric() {
            return self.operation.take().is_some();
        }
        let Some(browser) = &state.browser else {
            return false;
        };
        let previous = self.operation;
        let join_available = single_body(state);
        if !join_available {
            self.operation = None;
        }
        if !ui.collapsing_header("Output", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            return self.operation != previous;
        }
        crate::theme::property_label(ui, "Operation");
        let label = self
            .operation
            .map_or("New Solid".into(), |operation| operation.to_string());
        if let Some(_choices) = ui.begin_combo("##extrude-operation", label) {
            if ui
                .selectable_config("New Solid")
                .selected(self.operation.is_none())
                .build()
            {
                self.operation = None;
            }
            if ui
                .selectable_config("Join")
                .disabled(!join_available)
                .selected(self.operation == Some(PrismBoolean::Join))
                .build()
            {
                self.operation = Some(PrismBoolean::Join);
            }
        }
        if self.operation.is_none() {
            crate::theme::property_label(ui, "Body Name");
            ui.input_text("##extrude-body-name", &mut self.body_name)
                .hint(format!("Solid{}", browser.bodies.len() + 1))
                .build();
        }
        self.operation != previous
    }

    pub(super) fn creation_action(&self, request: &crate::GuiExtrudePreview) -> GuiShellAction {
        let body_name = self.body_name.trim();
        let named_new_solid = request.operation.is_none()
            && request.reference_extent.is_none()
            && !body_name.is_empty();
        if request.reference_extent.is_some() || request.operation.is_some() || named_new_solid {
            return GuiShellAction::CommitExtrudeDraft(
                request.clone(),
                named_new_solid.then(|| body_name.to_owned()),
            );
        }
        GuiShellAction::Extrude {
            document: request.document,
            sketch: request.sketch,
            profile: request.profile,
            distance: request.distance.clone(),
            second_distance: request.second_distance.clone(),
            direction: request.direction,
        }
    }
}
