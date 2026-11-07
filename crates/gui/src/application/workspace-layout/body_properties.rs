// SPDX-License-Identifier: GPL-2.0-only
//! The Body Properties dialog: a body's physical values on request, and its appearance override.
use crate::{GuiBrowserBody, GuiCommandState, GuiModelBrowser, GuiShellAction};
use imgui::{Key, Ui};
use oblikovati_model::feature::FeatureId;

const TITLE: &str = "Body Properties";
const SIZE: [f32; 2] = [520., 340.];
const SIDE_MARGIN: f32 = 20.;
const TOP_MARGIN: f32 = 180.;
const NOT_CALCULATED: &str = "N/A";
/// The selector's row for a body with no appearance of its own.
pub(crate) const AS_PART: &str = "As Part";

/// A body's calculated values, as the dialog shows them.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiBodyPhysical {
    pub mass: String,
    pub area: String,
    pub volume: String,
    pub center: [String; 3],
}
/// One appearance the selector offers.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiAppearanceChoice {
    pub id: String,
    pub name: String,
}
#[derive(Default, Debug)]
pub(crate) struct BodyPropertiesDialog {
    opening: bool,
    body: Option<FeatureId>,
    /// A choice OK has not applied yet; closing any other way forgets it.
    pending: Option<String>,
}
impl BodyPropertiesDialog {
    /// Open for one body; example: from its browser row's menu.
    pub(crate) fn request(&mut self, body: FeatureId) {
        self.opening = true;
        self.body = Some(body);
        self.pending = None;
    }
    /// A frame must follow so the dialog opens without waiting for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        self.opening
    }
    /// Draw while its body exists; example: inside the browser panel, each frame.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        browser: &GuiModelBrowser,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if std::mem::take(&mut self.opening) {
            ui.open_popup(TITLE);
        }
        // To one side: the chosen solid stays in view behind it, near the middle of the window.
        let [width, _] = ui.io().display_size;
        crate::widget_abi::window_geometry(
            ui,
            [(width - SIZE[0] - SIDE_MARGIN).max(0.), TOP_MARGIN],
            SIZE,
            imgui::Condition::Appearing,
        );
        let mut open = true;
        let Some(_dialog) = ui.modal_popup_config(TITLE).opened(&mut open).begin_popup() else {
            return;
        };
        let body = self
            .body
            .and_then(|id| browser.bodies.iter().find(|body| body.feature == id));
        let Some(body) = body else {
            ui.close_current_popup();
            return;
        };
        let document = state.document.unwrap_or(0);
        self.identity_row(ui, body, document, actions);
        physical_rows(ui, body);
        self.appearance_row(ui, body, browser);
        muted_checkbox(ui);
        ui.separator();
        self.end_buttons(ui, body, document, actions);
    }
    fn identity_row(
        &self,
        ui: &Ui,
        body: &GuiBrowserBody,
        document: u64,
        actions: &mut Vec<GuiShellAction>,
    ) {
        ui.text(&body.name);
        ui.same_line();
        if crate::widget_abi::button(ui, "Update", [0.; 2]) {
            actions.push(GuiShellAction::UpdateBodyProperties {
                document,
                body: body.index,
            });
        }
    }
    fn chosen_appearance(&self, body: &GuiBrowserBody) -> String {
        self.pending
            .clone()
            .unwrap_or_else(|| body.appearance.clone())
    }
    fn appearance_row(&mut self, ui: &Ui, body: &GuiBrowserBody, browser: &GuiModelBrowser) {
        let chosen = self.chosen_appearance(body);
        let preview = browser
            .appearances
            .iter()
            .find(|choice| choice.id == chosen)
            .map_or(AS_PART, |choice| choice.name.as_str());
        let Some(_list) = ui.begin_combo("Body Appearance", preview) else {
            return;
        };
        if ui
            .selectable_config(AS_PART)
            .selected(chosen.is_empty())
            .build()
        {
            self.pending = Some(String::new());
        }
        for choice in &browser.appearances {
            let row = ui
                .selectable_config(&choice.name)
                .selected(choice.id == chosen)
                .build();
            if row {
                self.pending = Some(choice.id.clone());
            }
        }
    }
    fn end_buttons(
        &mut self,
        ui: &Ui,
        body: &GuiBrowserBody,
        document: u64,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if crate::widget_abi::button(ui, "OK", [80., 0.]) {
            if let Some(appearance) = self.pending.take() {
                actions.push(GuiShellAction::SetBodyAppearance {
                    document,
                    key: body.key.clone(),
                    appearance,
                });
            }
            ui.close_current_popup();
        }
        ui.same_line();
        let cancelled = crate::widget_abi::button(ui, "Cancel", [80., 0.])
            || ui.is_key_pressed_no_repeat(Key::Escape);
        if cancelled {
            ui.close_current_popup();
        }
    }
}
fn physical_rows(ui: &Ui, body: &GuiBrowserBody) {
    let shown = |pick: fn(&GuiBodyPhysical) -> &str| {
        body.physical
            .as_ref()
            .map_or(NOT_CALCULATED, pick)
            .to_owned()
    };
    ui.text("General Properties");
    ui.text(format!("Mass: {}", shown(|row| &row.mass)));
    ui.text(format!("Area: {}", shown(|row| &row.area)));
    ui.text(format!("Volume: {}", shown(|row| &row.volume)));
    ui.text(format!("X: {}", shown(|row| &row.center[0])));
    ui.text(format!("Y: {}", shown(|row| &row.center[1])));
    ui.text(format!("Z: {}", shown(|row| &row.center[2])));
}
/// Listed as observed; this build keeps no overrides for it to clear.
fn muted_checkbox(ui: &Ui) {
    let _muted = ui.begin_disabled(true);
    let mut unchecked = false;
    ui.checkbox("Clear All Overrides", &mut unchecked);
}

#[cfg(test)]
mod tests {
    use super::*;
    fn body() -> GuiBrowserBody {
        GuiBrowserBody {
            name: "Solid1".into(),
            feature: FeatureId(1),
            key: "ab".into(),
            index: 0,
            appearance: "black".into(),
            physical: None,
        }
    }
    #[test]
    fn a_pending_choice_outranks_the_stored_one_until_the_dialog_reopens() {
        let mut dialog = BodyPropertiesDialog::default();
        dialog.request(FeatureId(1));
        assert_eq!(dialog.chosen_appearance(&body()), "black");
        dialog.pending = Some(String::new());
        assert_eq!(dialog.chosen_appearance(&body()), "");
        dialog.request(FeatureId(1));
        assert_eq!(dialog.chosen_appearance(&body()), "black");
    }
}
