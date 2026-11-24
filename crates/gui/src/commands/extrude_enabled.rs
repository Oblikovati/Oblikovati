// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::DocumentSketchId;

fn context(state: &GuiCommandState, document: u64, sketch: DocumentSketchId, profile: i64) -> bool {
    state.extrude_available
        && state.document == Some(document)
        && state.authoring == GuiAuthoringContext::Part
        && !state.transaction_open
        && profile >= 0
        && state
            .browser
            .as_ref()
            .is_some_and(|browser| browser.sketches.iter().any(|row| row.id == sketch))
}

pub(super) fn numeric(
    state: &GuiCommandState,
    document: u64,
    sketch: DocumentSketchId,
    profile: i64,
    distance: &str,
) -> bool {
    context(state, document, sketch, profile)
        && !distance.trim().is_empty()
        && !distance.contains('\0')
}

pub(super) fn reference(state: &GuiCommandState, request: &GuiExtrudePreview) -> bool {
    context(state, request.document, request.sketch, request.profile)
        && request.generation.is_some()
        && request.generation == state.document_generation
        && request.reference_extent.is_some()
}

pub(super) fn preview(state: &GuiCommandState, request: &GuiExtrudePreview) -> bool {
    if request.reference_extent.is_some() {
        reference(state, request)
    } else {
        numeric(
            state,
            request.document,
            request.sketch,
            request.profile,
            &request.distance,
        )
    }
}
