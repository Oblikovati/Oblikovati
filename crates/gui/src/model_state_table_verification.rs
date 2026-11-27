// SPDX-License-Identifier: GPL-2.0-only
//! Match table verification results to the staged draft (#4119).
use super::*;

impl ModelStateTable {
    pub(super) fn current_verification(
        &self,
        state: &GuiCommandState,
    ) -> Option<oblikovati_api::wire::ModelStateTableVerification> {
        let (document, checked, result) = state.model_state_verification.as_ref()?;
        let draft = self.draft.as_ref()?;
        (Some(*document) == self.document && *checked == request(draft)).then(|| result.clone())
    }
    pub(super) fn verify(&self, actions: &mut Vec<GuiShellAction>) {
        let (Some(document), Some(view)) = (self.document, self.draft.as_ref()) else {
            return;
        };
        actions.push(GuiShellAction::VerifyModelStateTable {
            document,
            request: request(view),
        });
    }
    pub(super) fn verification_modal(&mut self, ui: &Ui) {
        if self.show_verification_error {
            ui.open_popup("Table contains an error");
            self.show_verification_error = false;
        }
        if let Some(_popup) = ui
            .modal_popup_config("Table contains an error")
            .always_auto_resize(true)
            .begin_popup()
        {
            for diagnostic in &self.extra.diagnostics {
                let name = self
                    .draft
                    .as_ref()
                    .and_then(|view| view.available_parameters.as_deref())
                    .unwrap_or_default()
                    .iter()
                    .find(|item| item.id == diagnostic.parameter_id)
                    .map_or("Table", |item| item.name.as_str());
                if diagnostic.parameter_id == 0 {
                    ui.text(&diagnostic.message);
                } else {
                    ui.text(format!(
                        "{name}: {} cannot be evaluated.",
                        diagnostic.expression
                    ));
                }
            }
            if ui.button("OK") {
                ui.close_current_popup();
            }
        }
    }
}
