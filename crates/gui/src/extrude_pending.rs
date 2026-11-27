// SPDX-License-Identifier: GPL-2.0-only
//! Retain an extrusion request through its active source sketch.
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use oblikovati_model::document_sketch::DocumentSketchId;

#[derive(Default)]
pub(super) struct PendingExtrude {
    document: Option<u64>,
    sketch: Option<DocumentSketchId>,
    finishing: bool,
}
impl PendingExtrude {
    pub(super) fn start(&mut self, document: Option<u64>) {
        *self = Self {
            document,
            ..Self::default()
        };
    }
    pub(super) fn observe(
        &mut self,
        state: &GuiCommandState,
        plane_active: bool,
        actions: &[GuiShellAction],
    ) -> Option<DocumentSketchId> {
        let document = self.document?;
        if state.document != Some(document) {
            *self = Self::default();
            return None;
        }
        if state.authoring == GuiAuthoringContext::Sketch {
            self.observe_sketch(state.active_sketch, actions);
            return None;
        }
        if self.finishing && state.authoring == GuiAuthoringContext::Part {
            let sketch = self.sketch;
            *self = Self::default();
            return sketch;
        }
        if !plane_active && !sketch_starting(actions) {
            *self = Self::default();
        }
        None
    }
    fn observe_sketch(&mut self, sketch: Option<DocumentSketchId>, actions: &[GuiShellAction]) {
        if self.sketch.is_some() && self.sketch != sketch {
            *self = Self::default();
            return;
        }
        self.sketch = sketch;
        self.finishing |= actions.contains(&GuiShellAction::FinishSketch);
    }
}
fn sketch_starting(actions: &[GuiShellAction]) -> bool {
    actions.iter().any(|action| {
        matches!(
            action,
            GuiShellAction::StartSketchOn(_)
                | GuiShellAction::StartSketchOnWorkPlane(_)
                | GuiShellAction::StartSketchAt { .. }
        )
    })
}

#[cfg(test)]
#[path = "extrude_pending_tests.rs"]
mod tests;
