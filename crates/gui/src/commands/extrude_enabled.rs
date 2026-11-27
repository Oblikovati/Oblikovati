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
    let through_all = matches!(
        request.reference_extent,
        Some(GuiExtrudeReferenceExtent::ThroughAll)
    );
    if through_all
        && request.operation != Some(oblikovati_model::feature::sketch_prism::PrismBoolean::Cut)
    {
        return false;
    }
    let two_bounds = matches!(
        request.reference_extent,
        Some(GuiExtrudeReferenceExtent::ToFace { .. } | GuiExtrudeReferenceExtent::FromTo { .. })
    );
    if request.operation.is_some()
        && (two_bounds
            || request.generation.is_none()
            || request.generation != state.document_generation
            || !crate::extrude_dialog::single_body(state))
    {
        return false;
    }
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

pub(super) fn commit_draft(
    state: &GuiCommandState,
    draft: &GuiExtrudePreview,
    name: Option<&str>,
) -> bool {
    preview(state, draft)
        && state.extrude_draft_preview.as_ref() == Some(draft)
        && name.is_none_or(|name| {
            draft.operation.is_none() && !name.is_empty() && !name.contains('\0')
        })
}
