// SPDX-License-Identifier: GPL-2.0-only
//! Keep the applied sketch available as the source for a fresh profile selection.
use super::{ExtrudeDialog, GuiCommandState, GuiShellAction};
use crate::GuiExtrudeProfilePick;

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
        self.profile_pick_sketch = None;
        self.distance_editor = Default::default();
        self.edit_feature = None;
        if let Some(draft) = carried {
            self.sketch = None;
            if !self.release_sketch_on_apply {
                self.profile_pick_sketch = Some(draft.sketch);
            }
            self.operation = draft.operation;
            self.reference.clear_picks();
        } else {
            self.initialize_fresh_draft(state);
        }
    }

    fn profile_pick_target(&self, state: &GuiCommandState) -> Option<GuiExtrudeProfilePick> {
        let document = self.document?;
        let sketch = self.profile_pick_sketch?;
        let visible = state
            .browser
            .as_ref()?
            .sketches
            .iter()
            .any(|row| row.id == sketch && row.visible);
        (state.document == Some(document) && self.sketch.is_none() && visible).then_some(
            GuiExtrudeProfilePick {
                document,
                generation: state.document_generation?,
                sketch,
            },
        )
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
            self.sketch = Some(pick.sketch);
            self.profile = profile;
            self.profile_pick_sketch = None;
            self.commit_error = None;
        }
    }
}
