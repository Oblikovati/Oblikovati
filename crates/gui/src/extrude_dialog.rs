// SPDX-License-Identifier: GPL-2.0-only
//! Desktop extrusion draft. The document changes only when the owner accepts Apply.
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Key, Ui};
use oblikovati_model::document_sketch::DocumentSketchId;

#[derive(Default)]
pub(crate) struct ExtrudeDialog {
    pub requested: bool,
    document: Option<u64>,
    sketch: Option<DocumentSketchId>,
    profile: i32,
    distance: String,
    previewed: bool,
}
pub(crate) fn start_action(state: &GuiCommandState) -> GuiShellAction {
    if state
        .browser
        .as_ref()
        .is_some_and(|browser| browser.sketches.is_empty())
    {
        GuiShellAction::ChooseSketchPlane
    } else {
        GuiShellAction::OpenExtrude
    }
}
impl ExtrudeDialog {
    fn synchronize(&mut self, state: &GuiCommandState) {
        if self.requested {
            self.requested = false;
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
        actions: &mut Vec<GuiShellAction>,
    ) {
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
        crate::widget_abi::fit_dialog(ui, "Extrude", [640., 280.], [420., 200.]);
        ui.window("Extrude").build(|| {
            let mut edited = self.sketch_choice(ui, state);
            edited |= ui.input_int("Profile", &mut self.profile).build();
            edited |= ui.input_text("Distance", &mut self.distance).build();
            if edited && self.previewed {
                actions.push(GuiShellAction::PreviewExtrude(None));
                self.previewed = false;
            }
            self.controls(ui, state, document, actions);
        });
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
            .is_some_and(|action| action.dispatchable(state));
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
                    document,
                    sketch,
                    profile,
                    distance,
                },
            )));
            self.previewed = true;
        }
        ui.same_line();
        let pressed = crate::widget_abi::button(ui, "Apply", [0.; 2]);
        if enabled && (pressed || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Enter))
        {
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
        assert_eq!(start_action(&state), GuiShellAction::ChooseSketchPlane);
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
