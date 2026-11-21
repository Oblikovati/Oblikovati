// SPDX-License-Identifier: GPL-2.0-only
//! Admission for a generation-scoped straight-hole draft.
use super::*;

pub(super) fn open(state: &GuiCommandState) -> bool {
    state.fillet_available
        && state.document.is_some()
        && state.authoring == GuiAuthoringContext::Part
        && !state.transaction_open
}
pub(super) fn request(state: &GuiCommandState, request: &GuiHoleRequest) -> bool {
    blend_enabled::create(state)
        && state.document == Some(request.document)
        && state
            .fillet_selection
            .as_ref()
            .is_some_and(|(generation, _)| *generation == request.generation)
        && matches!(state.selected_face_appearances.as_slice(), [(key, _)] if *key == request.face)
        && !request.diameter.trim().is_empty()
}
