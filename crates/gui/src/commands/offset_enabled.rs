// SPDX-License-Identifier: GPL-2.0-only
//! Admission for validated offset-plane creation.
use super::*;

pub(super) fn create(
    state: &GuiCommandState,
    document: u64,
    base: &GuiOffsetPlaneBase,
    offset: &str,
) -> bool {
    state.authoring == GuiAuthoringContext::Part
        && !state.transaction_open
        && state.document == Some(document)
        && !offset.trim().is_empty()
        && !offset.contains('\0')
        && state
            .offset_plane_validation
            .as_ref()
            .is_some_and(|result| {
                result.document == document && result.offset == offset && result.error.is_none()
            })
        && match base {
            GuiOffsetPlaneBase::Origin(_) => true,
            GuiOffsetPlaneBase::WorkPlane(id) => state.browser.as_ref().is_some_and(|browser| {
                browser
                    .features
                    .iter()
                    .any(|row| row.id == *id && row.kind == "Work Plane")
            }),
            GuiOffsetPlaneBase::SelectedFace => state.model_face_chosen,
        }
}
