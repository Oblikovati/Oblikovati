// SPDX-License-Identifier: GPL-2.0-only
//! Admission for a generation-scoped straight-hole draft.
use super::*;

pub(super) fn open(state: &GuiCommandState) -> bool {
    state.fillet_available
        && state.document.is_some()
        && state.authoring == GuiAuthoringContext::Part
        && !state.transaction_open
}
