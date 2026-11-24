// SPDX-License-Identifier: GPL-2.0-only
//! Desktop extrusion draft. The document changes only when the owner accepts Apply.
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiExtrudeDirection, GuiShellAction};
use imgui::{Key, Ui};
use oblikovati_model::document_sketch::DocumentSketchId;
#[path = "extrude_direction.rs"]
mod direction;
#[path = "extrude_pending.rs"]
mod pending;

#[derive(Default)]
pub(crate) struct ExtrudeDialog {
    pub requested: bool,
    document: Option<u64>,
    dock_state: crate::dockable_lifecycle::DockableState,
    sketch: Option<DocumentSketchId>,
    profile: i32,
    distance: String,
    direction: GuiExtrudeDirection,
    previewed: bool,
    attempted: Option<crate::GuiExtrudePreview>,
    pending: pending::PendingExtrude,
    submitted: Option<crate::GuiExtrudePreview>,
    commit_error: Option<String>,
    edit_feature: Option<oblikovati_model::feature::FeatureId>,
    submitted_edit: Option<crate::GuiExtrudeEditRequest>,
}
pub(crate) fn start_action(state: &GuiCommandState) -> GuiShellAction {
    if state
        .browser
        .as_ref()
        .is_some_and(|browser| browser.sketches.is_empty())
    {
        GuiShellAction::ChooseSketchPlaneForExtrude
    } else {
        GuiShellAction::OpenExtrude
    }
}
impl DockableWindow for ExtrudeDialog {}
impl ExtrudeDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Extrude", &self.dock_state)
    }
    fn set_document(&mut self, document: Option<u64>) {
        self.document = document;
        self.dock_state.set_open(document.is_some());
    }
    pub(crate) fn open_edit(
        &mut self,
        state: &GuiCommandState,
        feature: oblikovati_model::feature::FeatureId,
    ) {
        let Some(seed) = state
            .browser
            .as_ref()
            .and_then(|browser| browser.extrusion_edits.get(&feature))
        else {
            return;
        };
        self.set_document(state.document);
        self.sketch = Some(seed.sketch);
        self.profile = 0;
        self.distance = seed.distance.clone();
        self.direction = seed.direction;
        self.edit_feature = Some(feature);
        self.attempted = None;
        self.commit_error = None;
    }
    pub(crate) fn wait_for_sketch(&mut self, document: Option<u64>) {
        self.pending.start(document);
    }
    fn synchronize(&mut self, state: &GuiCommandState) {
        if self.requested {
            self.requested = false;
            self.submitted = None;
            self.commit_error = None;
            self.attempted = None;
            self.set_document(state.document);
            self.sketch = state.browser.as_ref().and_then(|browser| {
                browser
                    .selected_sketches
                    .first()
                    .copied()
                    .or_else(|| browser.sketches.first().map(|row| row.id))
            });
            self.profile = 0;
            self.distance.clear();
            self.direction = GuiExtrudeDirection::Positive;
            self.edit_feature = None;
        }
        if self.document != state.document
            || !state.extrude_available
            || state.authoring != crate::GuiAuthoringContext::Part
        {
            self.set_document(None);
            self.edit_feature = None;
        }
    }
    pub(crate) fn sync_before_docking(&mut self, state: &GuiCommandState) {
        if !self.requested {
            self.synchronize(state);
        }
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        plane_active: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if let Some(sketch) = self.pending.observe(state, plane_active, actions) {
            self.set_document(state.document);
            self.sketch = Some(sketch);
            self.profile = 0;
            self.distance = "10 mm".into();
            self.direction = GuiExtrudeDirection::Positive;
            self.attempted = None;
            self.edit_feature = None;
        }
        if self.requested && self.previewed {
            actions.push(GuiShellAction::PreviewExtrude(None));
            self.previewed = false;
        }
        self.synchronize(state);
        if self.document.is_none() && self.previewed {
            actions.push(GuiShellAction::PreviewExtrude(None));
            self.previewed = false;
        }
        let Some(document) = self.document else {
            return;
        };
        Self::fit_dockable(ui, "Extrude", [640., 280.], [420., 200.]);
        let mut open = true;
        Self::dockable_window(ui, "Extrude", &mut open).build(|| {
            let disabled_sketch = ui.begin_disabled(self.edit_feature.is_some());
            let mut edited = self.sketch_choice(ui, state);
            ui.text("Profiles");
            if state.extrude_preview_active {
                ui.text("1 Profile");
            }
            edited |= ui.input_int("Profile", &mut self.profile).build();
            drop(disabled_sketch);
            edited |= ui.input_text("Distance A", &mut self.distance).build();
            edited |= direction::draw(ui, &mut self.direction, self.edit_feature.is_some());
            if edited {
                self.commit_error = None;
            }
            if edited && self.previewed {
                actions.push(GuiShellAction::PreviewExtrude(None));
                self.previewed = false;
            }
            self.refresh_preview(state, document, edited, actions);
            let validation = self.current_distance_validation(state, document);
            if !self.distance.trim().is_empty() && validation.is_none() {
                actions.push(GuiShellAction::ValidateExtrudeDistance {
                    document,
                    distance: self.distance.clone(),
                });
            }
            if let Some(error) = validation.and_then(|result| result.error.as_ref()) {
                ui.text_colored([1.0, 0.25, 0.25, 1.0], format!("Invalid distance: {error}"));
            }
            if let Some(error) = &self.commit_error {
                ui.text_colored([1.0, 0.25, 0.25, 1.0], format!("Extrusion failed: {error}"));
            }
            self.controls(ui, state, document, edited, actions);
        });
        if !open {
            if self.previewed {
                actions.push(GuiShellAction::PreviewExtrude(None));
                self.previewed = false;
            }
            self.set_document(None);
            self.edit_feature = None;
        }
    }
    fn refresh_preview(
        &mut self,
        state: &GuiCommandState,
        document: u64,
        edited: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let request = self.valid_preview_request(state, document, edited);
        if request == self.attempted {
            return;
        }
        if self.previewed {
            actions.push(GuiShellAction::PreviewExtrude(None));
            self.previewed = false;
        }
        self.attempted = request.clone();
        if let Some(request) = request {
            if let Some(feature) = self.edit_feature {
                actions.push(GuiShellAction::PreviewExtrudeEdit(
                    crate::GuiExtrudeEditRequest {
                        generation: request.generation,
                        document,
                        feature,
                        distance: request.distance,
                    },
                ));
            } else {
                actions.push(GuiShellAction::PreviewExtrude(Some(request)));
            }
            self.previewed = true;
        }
    }
    fn valid_preview_request(
        &self,
        state: &GuiCommandState,
        document: u64,
        edited: bool,
    ) -> Option<crate::GuiExtrudePreview> {
        let validation = self.current_distance_validation(state, document)?;
        if edited || validation.error.is_some() {
            return None;
        }
        Some(crate::GuiExtrudePreview {
            generation: Some(validation.generation),
            document,
            sketch: self.sketch?,
            profile: i64::from(self.profile),
            distance: self.distance.clone(),
            direction: self.direction,
        })
    }
    fn sketch_choice(&mut self, ui: &Ui, state: &GuiCommandState) -> bool {
        let Some(browser) = state.browser.as_ref() else {
            return false;
        };
        let shown = browser
            .sketches
            .iter()
            .find(|row| Some(row.id) == self.sketch)
            .map_or("Choose a sketch", |row| row.name.as_str());
        let Some(_list) = ui.begin_combo("Sketch", shown) else {
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
        self.sketch != previous
    }
    fn controls(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        document: u64,
        edited: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let preview = self.valid_preview_request(state, document, edited);
        let enabled = preview
            .as_ref()
            .is_some_and(|request| self.creation_action(request).dispatchable(state));
        let disabled = ui.begin_disabled(!enabled);
        let request = preview.as_ref().filter(|_| enabled);
        self.preview_control(ui, request, actions);
        ui.same_line();
        let committed = self.commit_control(ui, request, actions);
        drop(disabled);
        if committed {
            return;
        }
        ui.same_line();
        self.cancel_control(ui, actions);
    }
    fn creation_action(&self, request: &crate::GuiExtrudePreview) -> GuiShellAction {
        GuiShellAction::Extrude {
            document: request.document,
            sketch: request.sketch,
            profile: request.profile,
            distance: request.distance.clone(),
            direction: request.direction,
        }
    }
    fn edit_request(
        &self,
        request: &crate::GuiExtrudePreview,
    ) -> Option<crate::GuiExtrudeEditRequest> {
        Some(crate::GuiExtrudeEditRequest {
            generation: request.generation,
            document: request.document,
            feature: self.edit_feature?,
            distance: request.distance.clone(),
        })
    }
    fn preview_control(
        &mut self,
        ui: &Ui,
        request: Option<&crate::GuiExtrudePreview>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !crate::widget_abi::button(ui, "Preview", [0.; 2]) {
            return;
        }
        let Some(request) = request else {
            return;
        };
        let action = match self.edit_request(request) {
            Some(edit) => GuiShellAction::PreviewExtrudeEdit(edit),
            None => GuiShellAction::PreviewExtrude(Some(request.clone())),
        };
        actions.push(action);
        self.previewed = true;
    }
    fn commit_control(
        &mut self,
        ui: &Ui,
        request: Option<&crate::GuiExtrudePreview>,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        let pressed = crate::widget_abi::button(ui, "OK", [0.; 2]);
        if !pressed && !(ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Enter)) {
            return false;
        }
        let Some(request) = request else {
            return false;
        };
        if let Some(edit) = self.edit_request(request) {
            self.submitted_edit = Some(edit.clone());
            actions.push(GuiShellAction::EditExtrudeDistance(edit));
        } else {
            let mut submitted = request.clone();
            submitted.generation = None;
            self.submitted = Some(submitted);
            actions.push(self.creation_action(request));
        }
        self.set_document(None);
        self.edit_feature = None;
        self.previewed = false;
        true
    }
    fn cancel_control(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        if !crate::widget_abi::button(ui, "Cancel", [0.; 2])
            && !(ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape))
        {
            return;
        }
        if self.previewed {
            actions.push(GuiShellAction::PreviewExtrude(None));
            self.previewed = false;
        }
        self.set_document(None);
        self.edit_feature = None;
    }
    fn current_distance_validation<'a>(
        &self,
        state: &'a GuiCommandState,
        document: u64,
    ) -> Option<&'a crate::GuiExtrudeDistanceValidation> {
        state
            .extrude_distance_validation
            .as_ref()
            .filter(|result| result.document == document && result.distance == self.distance)
    }
    pub(crate) fn reject(&mut self, request: &crate::GuiExtrudePreview, cause: &str) {
        if self.submitted.as_ref() != Some(request) {
            return;
        }
        self.set_document(Some(request.document));
        self.commit_error = Some(cause.into());
    }
    pub(crate) fn reject_edit(&mut self, request: &crate::GuiExtrudeEditRequest, cause: &str) {
        if self.submitted_edit.as_ref() != Some(request) {
            return;
        }
        self.set_document(Some(request.document));
        self.edit_feature = Some(request.feature);
        self.commit_error = Some(cause.into());
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{GuiAuthoringContext, GuiBrowserSketch, GuiModelBrowser};
    fn part() -> GuiCommandState {
        GuiCommandState {
            document: Some(7),
            authoring: GuiAuthoringContext::Part,
            extrude_available: true,
            browser: Some(GuiModelBrowser {
                sketches: vec![GuiBrowserSketch {
                    id: DocumentSketchId(4),
                    inspection: Ok(Default::default()),
                    name: "Sketch".into(),
                    visible: true,
                    consumed_by: None,
                    line_style: Default::default(),
                    dimensions_shown: false,
                }],
                ..Default::default()
            }),
            ..Default::default()
        }
    }
    #[test]
    fn stale_extrusion_closes_before_docking_without_consuming_a_new_request() {
        let mut dialog = ExtrudeDialog::default();
        dialog.set_document(Some(7));
        let mut state = part();
        state.document = Some(8);
        dialog.sync_before_docking(&state);
        assert!(dialog.document.is_none());
        dialog.requested = true;
        dialog.sync_before_docking(&state);
        assert!(dialog.requested);
    }
    #[test]
    fn draft_is_bound_to_the_open_document() {
        let mut state = part();
        let mut dialog = ExtrudeDialog {
            requested: true,
            ..Default::default()
        };
        dialog.synchronize(&state);
        assert_eq!(dialog.document, Some(7));
        assert_eq!(dialog.sketch, Some(DocumentSketchId(4)));
        dialog.distance = "size".into();
        state.document = Some(8);
        dialog.synchronize(&state);
        assert_eq!(dialog.document, None);
        assert_eq!(start_action(&state), GuiShellAction::OpenExtrude);
        state.browser.as_mut().unwrap().sketches.clear();
        assert_eq!(
            start_action(&state),
            GuiShellAction::ChooseSketchPlaneForExtrude
        );
    }
    #[test]
    fn extrusion_requires_capability_current_sketch_and_nonempty_distance() {
        let mut state = part();
        let action = GuiShellAction::Extrude {
            document: 7,
            sketch: DocumentSketchId(4),
            profile: 0,
            distance: "size".into(),
            direction: GuiExtrudeDirection::Positive,
        };
        assert!(action.dispatchable(&state));
        state.extrude_available = false;
        assert!(!action.dispatchable(&state));
        state.extrude_available = true;
        state.transaction_open = true;
        assert!(!action.dispatchable(&state));
        state.transaction_open = false;
        state.document = Some(8);
        assert!(!action.dispatchable(&state));
    }
    #[test]
    fn edit_seed_keeps_its_source_and_new_extrusion_clears_edit_mode() {
        let mut state = part();
        let feature = oblikovati_model::feature::FeatureId(9);
        state.browser.as_mut().unwrap().extrusion_edits.insert(
            feature,
            crate::GuiExtrudeEditSeed {
                sketch: DocumentSketchId(4),
                distance: "size".into(),
                direction: GuiExtrudeDirection::Symmetric,
            },
        );
        let mut dialog = ExtrudeDialog::default();
        dialog.open_edit(&state, feature);
        assert_eq!(dialog.edit_feature, Some(feature));
        assert_eq!(dialog.distance, "size");
        assert_eq!(dialog.direction, GuiExtrudeDirection::Symmetric);
        dialog.requested = true;
        dialog.synchronize(&state);
        assert_eq!(dialog.edit_feature, None);
        assert!(dialog.distance.is_empty());
        assert_eq!(dialog.direction, GuiExtrudeDirection::Positive);
    }
}
