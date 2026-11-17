// SPDX-License-Identifier: GPL-2.0-only
//! The Body Properties dialog: a body's physical values on request, and its appearance override.
use crate::dockable_window::DockableWindow;
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
/// One appearance available to a selector.
pub type GuiAppearanceChoice = crate::GuiAssetChoice;
#[derive(Default, Debug)]
pub(crate) struct BodyPropertiesDialog {
    opening: bool,
    body: Option<FeatureId>,
    /// A choice OK has not applied yet; closing any other way forgets it.
    pending: Option<String>,
    document: Option<u64>,
    dock_state: crate::dockable_lifecycle::DockableState,
}
impl DockableWindow for BodyPropertiesDialog {}
impl BodyPropertiesDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Body Properties", &self.dock_state)
    }
    fn set_document(&mut self, document: Option<u64>) {
        self.document = document;
        self.dock_state.set_open(document.is_some());
    }
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
    pub(crate) fn sync_before_docking(&mut self, state: &GuiCommandState, suspended: bool) {
        let missing_body = state.browser.as_ref().is_some_and(|browser| {
            self.body
                .is_some_and(|id| !browser.bodies.iter().any(|body| body.feature == id))
        });
        if !self.opening
            && self.document.is_some()
            && (self.document != state.document || missing_body)
        {
            self.body = None;
            self.pending = None;
            self.set_document(None);
        }
        self.dock_state.set_suspended(suspended);
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
            self.set_document(state.document);
        }
        if self.document.is_none() || self.document != state.document {
            self.body = None;
            self.pending = None;
            self.set_document(None);
            return;
        }
        let body = self
            .body
            .and_then(|id| browser.bodies.iter().find(|body| body.feature == id));
        let Some(body) = body else {
            self.set_document(None);
            return;
        };
        // To one side: the chosen solid stays in view behind it, near the middle of the window.
        let [width, _] = ui.io().display_size;
        Self::fit_dockable(
            ui,
            TITLE,
            [(width - SIZE[0] - SIDE_MARGIN).max(0.), TOP_MARGIN],
            SIZE,
        );
        let mut open = true;
        let document = state.document.unwrap_or(0);
        let mut closed = false;
        Self::dockable_window(ui, TITLE, &mut open).build(|| {
            self.identity_row(ui, body, document, actions);
            physical_rows(ui, body);
            self.appearance_row(ui, body, browser);
            muted_checkbox(ui);
            ui.separator();
            closed = self.end_buttons(ui, body, document, actions);
        });
        if closed || !open {
            self.body = None;
            self.pending = None;
            self.set_document(None);
        }
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
    ) -> bool {
        if crate::widget_abi::button(ui, "OK", [80., 0.]) {
            if let Some(appearance) = self.pending.take() {
                actions.push(GuiShellAction::SetBodyAppearance {
                    document,
                    key: body.key.clone(),
                    appearance,
                });
            }
            return true;
        }
        ui.same_line();
        crate::widget_abi::button(ui, "Cancel", [80., 0.])
            || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape)
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
    #[test]
    fn deleted_body_closes_nested_window_before_docking() {
        let mut dialog = BodyPropertiesDialog {
            body: Some(FeatureId(1)),
            ..Default::default()
        };
        dialog.set_document(Some(1));
        let state = GuiCommandState {
            document: Some(1),
            browser: Some(GuiModelBrowser::default()),
            ..Default::default()
        };
        dialog.sync_before_docking(&state, false);
        assert!(dialog.body.is_none());
        assert!(!dialog.dock_state.is_requested());
    }
    #[test]
    fn document_change_closes_nested_window_before_docking() {
        let mut dialog = BodyPropertiesDialog {
            body: Some(FeatureId(1)),
            ..Default::default()
        };
        dialog.set_document(Some(1));
        dialog.sync_before_docking(
            &GuiCommandState {
                document: Some(2),
                ..Default::default()
            },
            false,
        );
        assert!(dialog.body.is_none());
        assert!(!dialog.dock_state.is_requested());
    }

    use super::*;
    fn body() -> GuiBrowserBody {
        GuiBrowserBody {
            name: "Solid1".into(),
            feature: FeatureId(1),
            key: "ab".into(),
            index: 0,
            visible: true,
            faces: 6,
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
