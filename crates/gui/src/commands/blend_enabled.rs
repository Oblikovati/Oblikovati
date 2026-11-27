// SPDX-License-Identifier: GPL-2.0-only
//! Admission for generation-scoped Fillet radius drafts.
use super::*;

pub(super) fn create(state: &GuiCommandState) -> bool {
    state.fillet_available
        && state.document.is_some()
        && state.authoring == GuiAuthoringContext::Part
        && !state.transaction_open
        && (state.selected_edge_count > 0
            || state
                .browser
                .as_ref()
                .is_some_and(|browser| browser.bodies.iter().any(|body| body.visible)))
}
pub(super) fn request(state: &GuiCommandState, request: &GuiFilletRequest) -> bool {
    create(state)
        && state.document == Some(request.document)
        && !request.edges.is_empty()
        && request.edges.len() == state.selected_edge_count
        && state.fillet_selection.as_ref() == Some(&(request.generation, request.edges.clone()))
        && !request.radius.trim().is_empty()
        && !request.radius.contains('\0')
}
pub(super) fn open(state: &GuiCommandState, feature: oblikovati_model::feature::FeatureId) -> bool {
    open_blend(state, feature, false)
}
pub(super) fn open_chamfer(
    state: &GuiCommandState,
    feature: oblikovati_model::feature::FeatureId,
) -> bool {
    open_blend(state, feature, true)
}
fn open_blend(
    state: &GuiCommandState,
    feature: oblikovati_model::feature::FeatureId,
    chamfer: bool,
) -> bool {
    state.fillet_available
        && state.document.is_some()
        && state.authoring == GuiAuthoringContext::Part
        && !state.transaction_open
        && edit_seed(state, feature, chamfer).is_some()
}
fn edit_seed(
    state: &GuiCommandState,
    feature: oblikovati_model::feature::FeatureId,
    chamfer: bool,
) -> Option<&crate::GuiFilletEditSeed> {
    let browser = state.browser.as_ref()?;
    if chamfer {
        browser.chamfer_edits.get(&feature)
    } else {
        browser.fillet_edits.get(&feature)
    }
}
pub(super) fn edit(state: &GuiCommandState, request: &GuiFilletEditRequest) -> bool {
    edit_blend(state, request, false)
}
pub(super) fn edit_chamfer(state: &GuiCommandState, request: &GuiFilletEditRequest) -> bool {
    edit_blend(state, request, true)
}
fn edit_blend(state: &GuiCommandState, request: &GuiFilletEditRequest, chamfer: bool) -> bool {
    open_blend(state, request.feature, chamfer)
        && state.document == Some(request.document)
        && !request.radius.trim().is_empty()
        && !request.radius.contains('\0')
        && edit_seed(state, request.feature, chamfer)
            .is_some_and(|seed| seed.generation == request.generation)
}
