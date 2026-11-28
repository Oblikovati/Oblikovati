// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::DocumentSketchId;

pub(super) fn open(state: &GuiCommandState) -> bool {
    let available_context = state.authoring == GuiAuthoringContext::Part
        || (state.authoring == GuiAuthoringContext::Sketch && state.active_sketch.is_some());
    available_context
        && state.extrude_available
        && !state.transaction_open
        && state.document.is_some()
        && state.browser.is_some()
}

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
    if !request.additional_profiles.is_empty()
        && (request.generation.is_none()
            || request.generation != state.document_generation
            || request.operation.is_some()
            || request
                .additional_profiles
                .iter()
                .any(|&index| index < 0 || index == request.profile))
    {
        return false;
    }
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
            draft.operation.is_none()
                && draft.additional_profiles.is_empty()
                && !name.is_empty()
                && !name.contains('\0')
        })
}

pub(super) fn open_edit(
    state: &GuiCommandState,
    feature: oblikovati_model::feature::FeatureId,
) -> bool {
    state.extrude_available
        && state.document.is_some()
        && state.authoring == GuiAuthoringContext::Part
        && !state.transaction_open
        && state
            .browser
            .as_ref()
            .is_some_and(|browser| browser.extrusion_edits.contains_key(&feature))
}

pub(super) fn edit_distance(state: &GuiCommandState, request: &GuiExtrudeEditRequest) -> bool {
    state.extrude_available
        && state.document == Some(request.document)
        && !state.transaction_open
        && !request.distance.trim().is_empty()
        && state
            .browser
            .as_ref()
            .is_some_and(|browser| browser.extrusion_edits.contains_key(&request.feature))
}
