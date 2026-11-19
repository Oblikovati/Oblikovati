// SPDX-License-Identifier: GPL-2.0-only
//! Admission for a sketch attached to one current face (#4050).
use super::*;

pub(super) fn explode_context(
    state: &GuiCommandState,
    token: u64,
    instance: oblikovati_model::sketch::SketchEntityId,
) -> bool {
    state.sketch_at_ready
        && !state.transaction_open
        && state
            .sketch_context
            .as_ref()
            .is_some_and(|context| context.token == token && context.block == Some(instance))
}

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

pub(super) fn create_block(state: &GuiCommandState, name: &str) -> bool {
    state.sketch_at_ready
        && !state.transaction_open
        && !name.trim().is_empty()
        && !name.contains('\0')
        && !state.sketch_block_names.iter().any(|item| item == name)
        && !state.sketch_selection.entities.is_empty()
}

pub(super) fn select_browser_block(
    state: &GuiCommandState,
    document: u64,
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
    instance: oblikovati_model::sketch::SketchEntityId,
) -> bool {
    state.document == Some(document)
        && state.authoring == GuiAuthoringContext::Sketch
        && state.sketch_at_ready
        && !state.transaction_open
        && state.browser.as_ref().is_some_and(|browser| {
            browser
                .block_instances
                .iter()
                .any(|row| row.sketch == sketch && row.id == instance)
        })
}
