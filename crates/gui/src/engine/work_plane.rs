// SPDX-License-Identifier: GPL-2.0-only
//! Work Plane browser selection and dialog synchronization helpers.

use crate::{GuiCommandState, GuiShellAction};

pub(super) fn selected_work_plane(
    state: &GuiCommandState,
) -> Option<oblikovati_model::feature::FeatureId> {
    let browser = state.browser.as_ref()?;
    let selected = browser.selected_feature?;
    browser
        .features
        .iter()
        .find(|feature| feature.id == selected && feature.kind == "Work Plane")
        .map(|feature| feature.id)
}

pub(super) fn parse_browser_picks(
    plane_ref: Option<&str>,
    picked_axis: Option<oblikovati_model::feature::work_axis::WorkAxisReference>,
) -> (
    Option<crate::GuiOriginPlaneKind>,
    Option<oblikovati_model::feature::work_axis::WorkAxisReference>,
    Option<oblikovati_model::feature::FeatureId>,
) {
    use oblikovati_api::types::{WORK_REF_XY_PLANE, WORK_REF_XZ_PLANE, WORK_REF_YZ_PLANE};
    let origin = match plane_ref {
        Some(WORK_REF_XY_PLANE) => Some(crate::GuiOriginPlaneKind::Xy),
        Some(WORK_REF_XZ_PLANE) => Some(crate::GuiOriginPlaneKind::Xz),
        Some(WORK_REF_YZ_PLANE) => Some(crate::GuiOriginPlaneKind::Yz),
        _ => None,
    };
    let work_plane = plane_ref
        .and_then(|value| value.strip_prefix("datum/v1/"))
        .and_then(|id| id.parse::<u64>().ok())
        .map(oblikovati_model::feature::FeatureId);
    (origin, picked_axis, work_plane)
}

pub(super) fn sync_offset_plane(
    commands: &GuiCommandState,
    workspace: &mut crate::application::workspace_layout::WorkspaceLayout,
    offset_plane: &mut crate::offset_plane::OffsetPlaneDialog,
    work_plane: &mut crate::work_plane_dialog::WorkPlaneDialog,
    selected: &mut Vec<GuiShellAction>,
) {
    let started_mode = selected.iter().rev().find_map(|action| match action {
        GuiShellAction::WorkPlane(crate::GuiWorkPlaneAction::Start(mode)) => Some(*mode),
        _ => None,
    });
    if let Some(mode) = started_mode {
        offset_plane.close(selected);
        work_plane.open(commands, mode);
    }
    let open_offset = selected.contains(&GuiShellAction::OpenOffsetPlane);
    selected.retain(|action| !matches!(action, GuiShellAction::OpenOffsetPlane));
    if open_offset {
        work_plane.close(selected);
    }
    if !open_offset && !offset_plane.active() {
        return;
    }
    let base = workspace
        .take_selected_origin_plane()
        .map(crate::GuiOffsetPlaneBase::Origin)
        .or_else(|| selected_work_plane(commands).map(crate::GuiOffsetPlaneBase::WorkPlane))
        .or_else(|| {
            commands
                .model_face_chosen
                .then_some(crate::GuiOffsetPlaneBase::SelectedFace)
        });
    if open_offset && let Some(document) = commands.document {
        offset_plane.open(document, base);
    } else if let Some(base) = base {
        offset_plane.choose(base);
    }
}
