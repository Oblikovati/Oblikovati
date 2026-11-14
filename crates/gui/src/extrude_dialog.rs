// SPDX-License-Identifier: GPL-2.0-only
//! Desktop extrusion draft. The document changes only when the owner accepts Apply.
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Key, Ui};
use oblikovati_model::document_sketch::DocumentSketchId;
#[path = "extrude_pending.rs"]
mod pending;

#[derive(Default)]
pub(crate) struct ExtrudeDialog {
    pub requested: bool,
    document: Option<u64>,
    sketch: Option<DocumentSketchId>,
    profile: i32,
    distance: String,
    previewed: bool,
    attempted: Option<crate::GuiExtrudePreview>,
    pending: pending::PendingExtrude,
    submitted: Option<crate::GuiExtrudePreview>,
    commit_error: Option<String>,
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
    pub(crate) fn wait_for_sketch(&mut self, document: Option<u64>) {
        self.pending.start(document);
    }
    fn synchronize(&mut self, state: &GuiCommandState) {
        if self.requested {
            self.requested = false;
            self.submitted = None;
            self.commit_error = None;
            self.attempted = None;
            self.document = state.document;
            self.sketch = state.browser.as_ref().and_then(|browser| {
                browser
                    .selected_sketches
                    .first()
                    .copied()
                    .or_else(|| browser.sketches.first().map(|row| row.id))
            });
            self.profile = 0;
            self.distance.clear();
        }
        if self.document != state.document
            || !state.extrude_available
            || state.authoring != crate::GuiAuthoringContext::Part
        {
            self.document = None;
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
            self.document = state.document;
            self.sketch = Some(sketch);
            self.profile = 0;
            self.distance = "10 mm".into();
            self.attempted = None;
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
            let mut edited = self.sketch_choice(ui, state);
            ui.text("Profiles");
            if state.extrude_preview_active {
                ui.text("1 Profile");
            }
            edited |= ui.input_int("Profile", &mut self.profile).build();
            edited |= ui.input_text("Distance A", &mut self.distance).build();
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
            self.document = None;
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
            actions.push(GuiShellAction::PreviewExtrude(Some(request)));
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
        let action = self.sketch.map(|sketch| GuiShellAction::Extrude {
            document,
            sketch,
            profile: i64::from(self.profile),
            distance: self.distance.clone(),
        });
        let enabled = action
            .as_ref()
            .is_some_and(|action| action.dispatchable(state))
            && !edited
            && self
                .current_distance_validation(state, document)
                .is_some_and(|result| result.error.is_none());
        let disabled = ui.begin_disabled(!enabled);
        if crate::widget_abi::button(ui, "Preview", [0.; 2])
            && enabled
            && let Some(GuiShellAction::Extrude {
                document,
                sketch,
                profile,
                distance,
            }) = action.clone()
        {
            actions.push(GuiShellAction::PreviewExtrude(Some(
                crate::GuiExtrudePreview {
                    generation: self
                        .current_distance_validation(state, document)
                        .map(|result| result.generation),
                    document,
                    sketch,
                    profile,
                    distance,
                },
            )));
            self.previewed = true;
        }
        ui.same_line();
        let pressed = crate::widget_abi::button(ui, "OK", [0.; 2]);
        if enabled && (pressed || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Enter))
        {
            self.submitted = self.sketch.map(|sketch| crate::GuiExtrudePreview {
                generation: None,
                document,
                sketch,
                profile: i64::from(self.profile),
                distance: self.distance.clone(),
            });
            actions.extend(action);
            self.document = None;
            self.previewed = false;
        }
        drop(disabled);
        ui.same_line();
        if crate::widget_abi::button(ui, "Cancel", [0.; 2])
            || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape)
        {
            if self.previewed {
                actions.push(GuiShellAction::PreviewExtrude(None));
                self.previewed = false;
            }
            self.document = None;
        }
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
        self.document = Some(request.document);
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
}
