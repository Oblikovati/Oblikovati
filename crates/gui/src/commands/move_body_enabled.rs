// SPDX-License-Identifier: GPL-2.0-only
//! Admission for a generation-scoped body motion draft.
use super::*;
pub(super) fn request(state: &GuiCommandState, request: &GuiMoveBodyRequest) -> bool {
    hole_enabled::open(state)
        && state.document == Some(request.document)
        && state
            .fillet_selection
            .as_ref()
            .is_some_and(|(generation, _)| *generation == request.generation)
        && state.browser.as_ref().is_some_and(|browser| {
            browser.bodies.iter().any(|body| {
                body.index == request.input.body_index
                    && crate::move_body_dialog::key_bytes(&body.key).as_deref()
                        == Some(request.body_key.as_slice())
            })
        })
}
