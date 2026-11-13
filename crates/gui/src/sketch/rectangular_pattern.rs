// SPDX-License-Identifier: GPL-2.0-only
use crate::dockable_window::DockableWindow;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::{Key, Ui};
use oblikovati_model::history::SketchRectangularPatternRequest;
use std::collections::BTreeSet;

pub(crate) fn ribbon_button(
    ui: &Ui,
    state: &GuiCommandState,
    dialog: &mut RectangularPatternDialog,
) -> bool {
    let disabled = ui.begin_disabled(
        !state.sketch_pattern_available || !state.sketch_at_ready || state.transaction_open,
    );
    let clicked = crate::widget_abi::button(ui, "Rectangular", [108., 26.]);
    if clicked {
        dialog.open(state);
    }
    drop(disabled);
    clicked
}
#[derive(Default)]
pub(crate) struct RectangularPatternDialog {
    document: Option<u64>,
    counts: [i32; 2],
    spacing: [String; 2],
    fitted: bool,
    reverse: [bool; 2],
    directions: [[f64; 2]; 2],
    edit: Option<crate::GuiBrowserRectangularPattern>,
    last_preview: Option<SketchRectangularPatternRequest>,
    pending: Option<Option<(u64, bool)>>,
}
impl DockableWindow for RectangularPatternDialog {}
impl RectangularPatternDialog {
    fn open(&mut self, state: &GuiCommandState) {
        self.document = state.document;
        self.counts = [2, 1];
        self.spacing = ["10 mm".into(), "10 mm".into()];
        self.fitted = false;
        self.reverse = [false; 2];
        self.directions = [[1., 0.], [0., 1.]];
        self.edit = None;
        self.last_preview = None;
        self.pending = None;
    }
    pub(crate) fn open_edit(
        &mut self,
        state: &GuiCommandState,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        pattern: oblikovati_model::sketch::SketchEntityId,
    ) {
        let Some(row) = state.browser.as_ref().and_then(|browser| {
            browser
                .rectangular_patterns
                .iter()
                .find(|row| row.sketch == sketch && row.pattern.id == pattern)
        }) else {
            return;
        };
        let Ok(spacing) = &row.spacing else {
            return;
        };
        let Ok(first) = i32::try_from(row.pattern.definition.counts[0]) else {
            return;
        };
        let Ok(second) = i32::try_from(row.pattern.definition.counts[1]) else {
            return;
        };
        self.document = state.document;
        self.counts = [first, second];
        self.spacing = spacing.clone();
        self.fitted = row.pattern.definition.fitted;
        self.directions = row.pattern.definition.directions;
        self.reverse = [false; 2];
        for axis in 0..2 {
            let mut negative_axis = [0.; 2];
            negative_axis[axis] = -1.;
            if self.directions[axis] == negative_axis {
                self.directions[axis] = self.directions[axis].map(|value| -value);
                self.reverse[axis] = true;
            }
        }
        self.edit = Some(row.clone());
        self.last_preview = None;
        self.pending = None;
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.document != state.document
            || state.authoring != GuiAuthoringContext::Sketch
            || self
                .edit
                .as_ref()
                .is_some_and(|row| state.active_sketch != Some(row.sketch))
        {
            self.close(actions);
        }
        self.observe_result(state, actions);
        if self.document.is_none() {
            return;
        }
        Self::fit_dockable(ui, "Rectangular Pattern", [620., 220.], [390., 365.]);
        let mut open = true;
        Self::dockable_window(ui, "Rectangular Pattern", &mut open).build(|| {
            ui.text(if self.edit.is_some() {
                "Edit the saved pattern"
            } else {
                "Select seed geometry in the sketch"
            });
            ui.text("Direction 1 count");
            ui.input_int("##pattern-count-1", &mut self.counts[0])
                .build();
            ui.text("Direction 1 spacing");
            ui.input_text("##pattern-spacing-1", &mut self.spacing[0])
                .build();
            ui.checkbox("Reverse Direction 1", &mut self.reverse[0]);
            ui.text("Direction 2 count");
            ui.input_int("##pattern-count-2", &mut self.counts[1])
                .build();
            ui.text("Direction 2 spacing");
            ui.input_text("##pattern-spacing-2", &mut self.spacing[1])
                .build();
            ui.checkbox("Reverse Direction 2", &mut self.reverse[1]);
            ui.checkbox("Fitted", &mut self.fitted);
            self.update_preview(state, actions);
            let request = self.request(state);
            let disabled = ui.begin_disabled(request.is_none() || self.pending.is_some());
            if ui.button("OK")
                && let Some(request) = request
            {
                self.pending = Some(state.sketch_pattern_created);
                actions.push(match &self.edit {
                    Some(row) => GuiShellAction::EditSketchPattern {
                        pattern: row.pattern.id,
                        request,
                    },
                    None => GuiShellAction::CreateSketchPattern(request),
                });
            }
            drop(disabled);
            ui.same_line();
            if ui.button("Cancel")
                || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape)
            {
                self.close(actions);
            }
        });
        if !open {
            self.close(actions);
        }
    }
    fn request(&self, state: &GuiCommandState) -> Option<SketchRectangularPatternRequest> {
        let counts = [
            u32::try_from(self.counts[0]).ok()?,
            u32::try_from(self.counts[1]).ok()?,
        ];
        if counts.contains(&0) || self.spacing.iter().any(|value| value.trim().is_empty()) {
            return None;
        }
        let seed: Vec<_> = self
            .edit
            .as_ref()
            .map(|row| row.pattern.definition.seed.clone())
            .unwrap_or_else(|| {
                state
                    .sketch_selection
                    .entities
                    .iter()
                    .map(|(_, id)| *id)
                    .collect()
            });
        if seed.is_empty() {
            return None;
        }
        let mut directions = self.directions;
        for (axis, direction) in directions.iter_mut().enumerate() {
            if self.reverse[axis] {
                *direction = direction.map(|value| -value);
            }
        }
        Some(SketchRectangularPatternRequest {
            seed,
            counts,
            directions,
            spacing: self.spacing.clone(),
            fitted: self.fitted,
            suppressed: self.edit.as_ref().map_or_else(BTreeSet::new, |row| {
                row.pattern.definition.suppressed.clone()
            }),
        })
    }
    fn update_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state);
        if self.last_preview != request {
            self.last_preview = request.clone();
            actions.push(match (self.edit.as_ref(), request) {
                (Some(row), Some(request)) => GuiShellAction::PreviewSketchPatternEdit {
                    pattern: row.pattern.id,
                    request,
                },
                (_, request) => GuiShellAction::PreviewSketchPattern(request),
            });
        }
    }
    fn observe_result(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        if !self
            .pending
            .is_some_and(|before| before != state.sketch_pattern_created)
        {
            return;
        }
        if state
            .sketch_pattern_created
            .is_some_and(|(_, success)| success)
        {
            self.close(actions);
        } else {
            self.pending = None;
        }
    }
    fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            actions.push(GuiShellAction::PreviewSketchPattern(None));
        }
        self.last_preview = None;
        self.edit = None;
        self.pending = None;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchEntityId};
    fn state() -> GuiCommandState {
        let mut state = GuiCommandState {
            document: Some(7),
            authoring: GuiAuthoringContext::Sketch,
            sketch_at_ready: true,
            sketch_pattern_available: true,
            ..Default::default()
        };
        state
            .sketch_selection
            .entities
            .push((DocumentSketchId(2), SketchEntityId(4)));
        state
    }
    #[test]
    fn draft_builds_checked_requests_and_preview_only_when_changed() {
        let state = state();
        let mut dialog = RectangularPatternDialog::default();
        dialog.open(&state);
        let mut actions = Vec::new();
        dialog.update_preview(&state, &mut actions);
        assert_eq!(actions.len(), 1);
        let Some(request) = dialog.request(&state) else {
            panic!("expected valid draft");
        };
        assert_eq!(request.seed, [SketchEntityId(4)]);
        assert_eq!(request.counts, [2, 1]);
        dialog.update_preview(&state, &mut actions);
        assert_eq!(actions.len(), 1);
        dialog.counts[0] = 0;
        dialog.update_preview(&state, &mut actions);
        assert!(matches!(
            actions.last(),
            Some(GuiShellAction::PreviewSketchPattern(None))
        ));
        dialog.counts[0] = 3;
        dialog.reverse[0] = true;
        dialog.fitted = true;
        let request = dialog.request(&state).unwrap();
        assert_eq!(request.counts, [3, 1]);
        assert_eq!(request.directions[0], [-1., 0.]);
        assert!(request.fitted);
    }
    #[test]
    fn cancel_discards_the_draft_and_clears_the_preview() {
        let state = state();
        let mut dialog = RectangularPatternDialog::default();
        dialog.open(&state);
        let mut actions = Vec::new();
        dialog.update_preview(&state, &mut actions);
        dialog.close(&mut actions);
        assert!(dialog.document.is_none());
        assert!(matches!(
            actions.last(),
            Some(GuiShellAction::PreviewSketchPattern(None))
        ));
    }
    #[test]
    fn failed_creation_keeps_the_draft_open_for_retry() {
        let mut state = state();
        let mut dialog = RectangularPatternDialog::default();
        dialog.open(&state);
        dialog.pending = Some(None);
        let mut actions = Vec::new();
        state.sketch_pattern_created = Some((1, false));
        dialog.observe_result(&state, &mut actions);
        assert_eq!(dialog.document, Some(7));
        assert!(dialog.pending.is_none());
        dialog.pending = Some(state.sketch_pattern_created);
        state.sketch_pattern_created = Some((2, true));
        dialog.observe_result(&state, &mut actions);
        assert!(dialog.document.is_none());
        assert!(matches!(
            actions.last(),
            Some(GuiShellAction::PreviewSketchPattern(None))
        ));
    }
    #[test]
    fn command_requires_the_host_capability_and_a_ready_sketch() {
        let mut state = state();
        let mut dialog = RectangularPatternDialog::default();
        dialog.open(&state);
        let action = GuiShellAction::CreateSketchPattern(dialog.request(&state).unwrap());
        assert!(action.enabled(&state));
        state.sketch_pattern_available = false;
        assert!(!action.enabled(&state));
        assert!(GuiShellAction::PreviewSketchPattern(None).enabled(&state));
    }
}
