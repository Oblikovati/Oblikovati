// SPDX-License-Identifier: GPL-2.0-only
//! Keep dialog choices separate from the applied sketch-session scope (#4107).
use crate::dockable_window::DockableWindow;
use oblikovati_api::{
    types::{
        INFERENCE_SCOPE_ALL_GEOMETRY, INFERENCE_SCOPE_CURRENT_COMMAND,
        INFERENCE_SCOPE_SELECTED_GEOMETRY, SketchInferenceScope,
    },
    wire::SketchInferenceScopeView,
};

#[derive(Default)]
pub(crate) struct InferenceScopeDialog {
    pub(crate) applied: Option<SketchInferenceScopeView>,
    pub(crate) draft: Option<SketchInferenceScopeView>,
    dock_state: crate::dockable_lifecycle::DockableState,
    pending: Option<SketchInferenceScopeView>,
    owner: Option<(u64, oblikovati_model::document_sketch::DocumentSketchId)>,
}
impl DockableWindow for InferenceScopeDialog {}
impl InferenceScopeDialog {
    pub(crate) fn open(&mut self, applied: SketchInferenceScopeView) {
        self.draft = Some(applied.clone());
        self.applied = Some(applied);
        self.dock_state.set_open(true);
    }
    pub(crate) fn choose(&mut self, mode: SketchInferenceScope) {
        let Some(draft) = self.draft.as_mut() else {
            return;
        };
        if mode == INFERENCE_SCOPE_CURRENT_COMMAND || mode == INFERENCE_SCOPE_ALL_GEOMETRY {
            draft.entities = Some(Vec::new());
        }
        draft.scope = mode;
    }
    pub(crate) fn select(&mut self, mut entities: Vec<u64>) {
        let Some(draft) = self.draft.as_mut() else {
            return;
        };
        if draft.scope == INFERENCE_SCOPE_SELECTED_GEOMETRY {
            entities.sort_unstable();
            entities.dedup();
            draft.entities = Some(entities);
        }
    }
    pub(crate) fn can_accept(&self) -> bool {
        self.draft.as_ref().is_some_and(|draft| {
            draft.scope != INFERENCE_SCOPE_SELECTED_GEOMETRY
                || draft.entities.as_ref().is_some_and(|ids| !ids.is_empty())
        })
    }
    pub(crate) fn cancel(&mut self) {
        self.draft = None;
        self.applied = None;
        self.pending = None;
        self.owner = None;
        self.dock_state.set_open(false);
    }
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Constraint Inference Scope", &self.dock_state)
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn stale_selection_waits_for_a_successful_replacement() {
        let sketch = oblikovati_model::document_sketch::DocumentSketchId(1);
        let replacement = SketchInferenceScopeView {
            sketch_index: 0,
            scope: INFERENCE_SCOPE_CURRENT_COMMAND,
            entities: Some(vec![]),
        };
        let mut state = crate::GuiCommandState {
            document: Some(1),
            active_sketch: Some(sketch),
            authoring: crate::GuiAuthoringContext::Sketch,
            sketch_inference_scope: Some(replacement.clone()),
            sketch_inference_scope_error: Some("deleted entity".into()),
            ..Default::default()
        };
        let mut dialog = InferenceScopeDialog::default();
        dialog.begin(&state);
        dialog.pending = Some(replacement);
        assert!(dialog.synchronize(&state).is_empty());
        assert!(dialog.draft.is_some());
        state.sketch_inference_scope_error = None;
        assert_eq!(
            dialog.synchronize(&state),
            [InferenceScopeEvent::EndSelection]
        );
        assert!(dialog.draft.is_none());
    }
    #[test]
    fn draft_requires_selection_and_cancel_keeps_applied_value() {
        let applied = SketchInferenceScopeView {
            sketch_index: 0,
            scope: INFERENCE_SCOPE_CURRENT_COMMAND,
            entities: Some(vec![]),
        };
        let mut dialog = InferenceScopeDialog::default();
        dialog.open(applied.clone());
        dialog.choose(INFERENCE_SCOPE_SELECTED_GEOMETRY);
        assert!(!dialog.can_accept());
        dialog.select(vec![3]);
        assert!(dialog.can_accept());
        assert_eq!(dialog.applied, Some(applied.clone()));
        dialog.cancel();
        dialog.open(applied);
        assert_eq!(dialog.draft.unwrap().scope, INFERENCE_SCOPE_CURRENT_COMMAND);
    }
}

/// Requests that the desktop applies after the current GUI frame.
#[derive(Clone, Debug, PartialEq)]
pub(crate) enum InferenceScopeEvent {
    BeginSelection(Vec<u64>),
    EndSelection,
    Apply(SketchInferenceScopeView),
}

impl InferenceScopeDialog {
    pub(crate) fn begin(&mut self, state: &crate::GuiCommandState) -> Vec<InferenceScopeEvent> {
        let (Some(document), Some(sketch), Some(scope)) = (
            state.document,
            state.active_sketch,
            state.sketch_inference_scope.clone(),
        ) else {
            return Vec::new();
        };
        self.open(scope.clone());
        self.owner = Some((document, sketch));
        if scope.scope == INFERENCE_SCOPE_SELECTED_GEOMETRY {
            return vec![InferenceScopeEvent::BeginSelection(
                scope.entities.unwrap_or_default(),
            )];
        }
        vec![InferenceScopeEvent::EndSelection]
    }
    pub(crate) fn synchronize(
        &mut self,
        state: &crate::GuiCommandState,
    ) -> Vec<InferenceScopeEvent> {
        let Some((document, sketch)) = self.owner else {
            return Vec::new();
        };
        let ended = state.document != Some(document)
            || state.active_sketch != Some(sketch)
            || state.authoring != crate::GuiAuthoringContext::Sketch;
        let accepted = state.sketch_inference_scope_error.is_none()
            && self
                .pending
                .as_ref()
                .is_some_and(|pending| state.sketch_inference_scope.as_ref() == Some(pending));
        if ended || accepted {
            self.cancel();
            return vec![InferenceScopeEvent::EndSelection];
        }
        Vec::new()
    }
    pub(crate) fn panel(
        &mut self,
        ui: &imgui::Ui,
        selected: &[u64],
        error: Option<&str>,
    ) -> Vec<InferenceScopeEvent> {
        if self.draft.is_none() {
            return Vec::new();
        }
        let mut events = Vec::new();
        let mut open = true;
        Self::fit_dockable(ui, "Constraint Inference Scope", [420., 215.], [320., 190.]);
        Self::dockable_window(ui, "Constraint Inference Scope", &mut open).build(|| {
            if let Some(error) = error {
                ui.text_wrapped(format!("The saved selection is unavailable: {error}. Choose a replacement and select OK."));
            }
            self.choices(ui, &mut events);
            self.selection(ui, selected);
            self.buttons(ui, &mut events);
        });
        if !open {
            self.cancel();
            events.push(InferenceScopeEvent::EndSelection);
        }
        events
    }
    fn choices(&mut self, ui: &imgui::Ui, events: &mut Vec<InferenceScopeEvent>) {
        for (label, mode) in [
            (
                "Geometry in current command",
                INFERENCE_SCOPE_CURRENT_COMMAND,
            ),
            ("All Geometry", INFERENCE_SCOPE_ALL_GEOMETRY),
            ("Select", INFERENCE_SCOPE_SELECTED_GEOMETRY),
        ] {
            let active = self.draft.as_ref().is_some_and(|draft| draft.scope == mode);
            if ui.radio_button_bool(label, active) && !active {
                let selecting = mode == INFERENCE_SCOPE_SELECTED_GEOMETRY;
                self.choose(mode);
                events.push(if selecting {
                    InferenceScopeEvent::BeginSelection(Vec::new())
                } else {
                    InferenceScopeEvent::EndSelection
                });
            }
        }
    }
    fn selection(&mut self, ui: &imgui::Ui, selected: &[u64]) {
        let selecting = self
            .draft
            .as_ref()
            .is_some_and(|draft| draft.scope == INFERENCE_SCOPE_SELECTED_GEOMETRY);
        if selecting {
            self.select(selected.to_vec());
        }
        let disabled = ui.begin_disabled(!selecting);
        ui.button(format!(
            "Geometry: {} selected",
            if selecting { selected.len() } else { 0 }
        ));
        drop(disabled);
    }
    fn buttons(&mut self, ui: &imgui::Ui, events: &mut Vec<InferenceScopeEvent>) {
        let disabled = ui.begin_disabled(!self.can_accept());
        if ui.button("OK")
            && let Some(draft) = self.draft.clone()
        {
            self.pending = Some(draft.clone());
            events.push(InferenceScopeEvent::Apply(draft));
        }
        drop(disabled);
        ui.same_line();
        if ui.button("Cancel") {
            self.cancel();
            events.push(InferenceScopeEvent::EndSelection);
        }
    }
}
