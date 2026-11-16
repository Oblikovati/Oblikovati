// SPDX-License-Identifier: GPL-2.0-only
//! Admission for generation-scoped Fillet radius drafts.
use super::*;

pub(super) fn create(state: &GuiCommandState) -> bool {
    state.fillet_available
        && state.document.is_some()
        && state.authoring == GuiAuthoringContext::Part
        && !state.transaction_open
        && state.selected_edge_count > 0
}
pub(super) fn open(state: &GuiCommandState, feature: oblikovati_model::feature::FeatureId) -> bool {
    state.fillet_available
        && state.document.is_some()
        && state.authoring == GuiAuthoringContext::Part
        && !state.transaction_open
        && state
            .browser
            .as_ref()
            .is_some_and(|browser| browser.fillet_edits.contains_key(&feature))
}
pub(super) fn edit(state: &GuiCommandState, request: &GuiFilletEditRequest) -> bool {
    open(state, request.feature)
        && state.document == Some(request.document)
        && !request.radius.trim().is_empty()
        && !request.radius.contains('\0')
        && state
            .browser
            .as_ref()
            .and_then(|browser| browser.fillet_edits.get(&request.feature))
            .is_some_and(|seed| seed.generation == request.generation)
}
