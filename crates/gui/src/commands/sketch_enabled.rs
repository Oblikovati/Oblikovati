// SPDX-License-Identifier: GPL-2.0-only
//! Admission for a sketch attached to one current face (#4050).
use super::*;

pub(super) fn selected_face(state: &GuiCommandState, document: u64, key: &str) -> bool {
    state.document == Some(document)
        && state.authoring == GuiAuthoringContext::Part
        && !state.transaction_open
        && !state.active_express()
        && state.model_face_chosen
        && state.selected_edge_count == 0
        && state.selected_body_keys.is_empty()
        && matches!(state.selected_face_appearances.as_slice(), [(selected, _)] if selected == key)
}

pub(super) fn delete_selection(state: &GuiCommandState) -> bool {
    state.authoring == GuiAuthoringContext::Sketch
        && state.sketch_delete_available
        && !state.transaction_open
}
