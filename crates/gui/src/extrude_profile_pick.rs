// SPDX-License-Identifier: GPL-2.0-only
//! Pick certified regions for a New Solid draft, including after Clear or Apply.
use super::{ExtrudeDialog, GuiCommandState, GuiShellAction};
use crate::GuiExtrudeProfilePick;
use imgui::Ui;

impl ExtrudeDialog {
    pub(super) fn open_requested_draft(&mut self, state: &GuiCommandState) {
        self.requested = false;
        let apply_more = std::mem::take(&mut self.start_without_sketch);
        let carried = self
            .submitted
            .take()
            .filter(|draft| apply_more && state.document == Some(draft.document));
        self.body_name.clear();
        self.commit_error = None;
        self.attempted = None;
        self.set_document(state.document);
        self.profile = 0;
        self.additional_profiles.clear();
        self.profile_pick_sketch = None;
        self.distance_editor = Default::default();
        self.edit_feature = None;
        if let Some(draft) = carried {
            self.sketch = None;
            if !self.release_sketch_on_apply {
                self.await_profile_pick(draft.sketch);
            }
            self.operation = draft.operation;
            self.reference.clear_picks();
        } else {
            self.initialize_fresh_draft(state);
        }
    }

    pub(super) fn sketch_choice(&mut self, ui: &Ui, state: &GuiCommandState) -> bool {
        let Some(browser) = state.browser.as_ref() else {
            return false;
        };
        let shown = browser
            .sketches
            .iter()
            .find(|row| Some(row.id) == self.sketch)
            .map_or("Choose a sketch", |row| row.name.as_str());
        crate::theme::property_label(ui, "Sketch");
        let Some(_list) = ui.begin_combo("##extrude-sketch", shown) else {
            return false;
        };
        let previous = self.sketch;
        for row in &browser.sketches {
            if ui
                .selectable_config(&row.name)
                .selected(Some(row.id) == self.sketch)
                .build()
            {
                self.sketch = Some(row.id);
            }
        }
        if self.sketch != previous {
            self.profile = 0;
            self.additional_profiles.clear();
            self.profile_pick_sketch = None;
            self.profile_pick_request = self.profile_pick_request.wrapping_add(1);
            return true;
        }
        false
    }
    pub(super) fn await_profile_pick(
        &mut self,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
    ) {
        self.sketch = None;
        self.profile = 0;
        self.additional_profiles.clear();
        self.profile_pick_sketch = Some(sketch);
        // A previous viewport response may still be in the owner snapshot at this generation.
        self.profile_pick_request = self.profile_pick_request.wrapping_add(1);
    }

    fn profile_pick_target(&self, state: &GuiCommandState) -> Option<GuiExtrudeProfilePick> {
        let document = self.document?;
        if self.edit_feature.is_some() || self.source_edit || self.reference.awaits_pick() {
            return None;
        }
        let sketch = self
            .profile_pick_sketch
            .or_else(|| self.sketch.filter(|_| self.operation.is_none()))?;
        let visible = state
            .browser
            .as_ref()?
            .sketches
            .iter()
            .any(|row| row.id == sketch && row.visible);
        (state.document == Some(document) && visible).then_some(GuiExtrudeProfilePick {
            request: self.profile_pick_request,
            document,
            generation: state.document_generation?,
            sketch,
        })
    }

    pub(crate) fn route_profile_pick(
        &self,
        state: &GuiCommandState,
        actions: &mut [GuiShellAction],
    ) {
        let Some(pick) = self.profile_pick_target(state) else {
            return;
        };
        for action in actions {
            if let GuiShellAction::ClickModel {
                toggle: false,
                extrude_profile,
                ..
            } = action
            {
                *extrude_profile = Some(pick);
            }
        }
    }

    pub(super) fn observe_profile_pick(&mut self, state: &GuiCommandState) {
        if let Some((pick, profile)) = state.extrude_profile_pick
            && self.profile_pick_target(state) == Some(pick)
        {
            // A hit is a single response, including a click on an already selected region.
            self.profile_pick_request = self.profile_pick_request.wrapping_add(1);
            if self.sketch.is_none() {
                self.sketch = Some(pick.sketch);
                self.profile = profile;
                self.profile_pick_sketch = None;
            } else if self.profile != profile
                && !self.additional_profiles.contains(&i64::from(profile))
            {
                self.additional_profiles.push(i64::from(profile));
            }
            self.commit_error = None;
        }
    }
}
