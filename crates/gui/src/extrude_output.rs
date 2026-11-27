// SPDX-License-Identifier: GPL-2.0-only
use super::{ExtrudeDialog, GuiCommandState, GuiShellAction};
use crate::GuiExtrudeDirection;
use imgui::Ui;
use oblikovati_model::feature::sketch_prism::PrismBoolean;

pub(crate) fn single_body(state: &GuiCommandState) -> bool {
    state
        .browser
        .as_ref()
        .is_some_and(|browser| browser.bodies.len() == 1)
}

impl ExtrudeDialog {
    pub(super) fn reference_direction(&self) -> GuiExtrudeDirection {
        if self.reference.through_all() {
            self.direction
        } else {
            GuiExtrudeDirection::Positive
        }
    }

    pub(super) fn reference_operation(&self) -> Option<PrismBoolean> {
        self.reference.through_all().then_some(PrismBoolean::Cut)
    }

    pub(super) fn draw_output(&mut self, ui: &Ui, state: &GuiCommandState) -> bool {
        if let Some(feature) = self.edit_feature {
            self.draw_edit_output(ui, state, feature);
            return false;
        }
        if !(self.reference.numeric() || self.reference.through_all()) {
            return self.operation.take().is_some();
        }
        let Some(browser) = &state.browser else {
            return false;
        };
        let previous = self.operation;
        let boolean_available = single_body(state);
        let through_all = self.reference.through_all();
        if through_all {
            self.operation = Some(PrismBoolean::Cut);
        } else if !boolean_available {
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
                .disabled(through_all)
                .build()
            {
                self.operation = None;
            }
            for operation in [
                PrismBoolean::Join,
                PrismBoolean::Cut,
                PrismBoolean::Intersect,
            ] {
                if ui
                    .selectable_config(operation.to_string())
                    .disabled(!boolean_available || (through_all && operation != PrismBoolean::Cut))
                    .selected(self.operation == Some(operation))
                    .build()
                {
                    self.choose_boolean_operation(operation);
                }
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

    fn draw_edit_output(
        &self,
        ui: &Ui,
        state: &GuiCommandState,
        feature: oblikovati_model::feature::FeatureId,
    ) {
        let Some(body) = state
            .browser
            .as_ref()
            .and_then(|browser| browser.bodies.iter().find(|body| body.feature == feature))
        else {
            return;
        };
        if ui.collapsing_header("Output", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            crate::theme::property_label(ui, "Body Name");
            ui.text(&body.name);
            // A saved closed profile fixes the section for this distance edit.
            let _disabled = ui.begin_disabled(true);
            ui.checkbox("Match Shape", &mut true);
        }
    }

    fn choose_boolean_operation(&mut self, operation: PrismBoolean) {
        if matches!(operation, PrismBoolean::Cut | PrismBoolean::Intersect)
            && self.operation != Some(operation)
            && self.direction == GuiExtrudeDirection::Positive
        {
            self.direction = GuiExtrudeDirection::Negative;
        }
        self.operation = Some(operation);
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
