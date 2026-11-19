// SPDX-License-Identifier: GPL-2.0-only
//! Typed edits of stored part views. Display application remains a desktop operation (#4118).
use oblikovati_api::{Error, contract::PartDesignView as _, wire::*};
use oblikovati_model::{
    history::{DocumentSnapshot, RecipeReservations},
    part_design_view::{PartDesignView, PartDesignViews},
};
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};

pub(super) enum PartViewEdit {
    Copy(CopyPartDesignViewArgs),
    Rename(RenamePartDesignViewArgs),
    Options(SetPartDesignViewOptionsArgs),
    Camera(CapturePartDesignViewCameraArgs),
}
impl PartViewEdit {
    pub(super) fn label(&self) -> &'static str {
        match self {
            Self::Copy(_) => "Copy Part View",
            Self::Rename(_) => "Rename Part View",
            Self::Options(_) => "Edit Part View Options",
            Self::Camera(_) => "Capture Part View Camera",
        }
    }
    pub(super) fn apply(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<Vec<u8>, Error> {
        let views = snapshot
            .part_design_views
            .as_mut()
            .ok_or_else(|| Error("part view edit: expected an active part document".into()))?;
        let id = match self {
            Self::Copy(request) => copy_view(views, ids, request)?,
            Self::Rename(request) => {
                views.rename(request.id, &request.name).map_err(Error)?;
                request.id
            }
            Self::Options(request) => {
                views.set_options(&request).map_err(Error)?;
                request.id
            }
            Self::Camera(request) => {
                views
                    .capture_camera(request.id, request.camera)
                    .map_err(Error)?;
                request.id
            }
        };
        crate::encode(&PartDesignViewResult {
            view: info(views.get(id).map_err(Error)?),
        })
    }
}
pub(crate) fn copy_view(
    views: &mut PartDesignViews,
    ids: &mut RecipeReservations<'_>,
    request: CopyPartDesignViewArgs,
) -> Result<u64, Error> {
    views.get(request.source_id).map_err(Error)?;
    let name = if request.name.is_empty() {
        views.generated_name()
    } else {
        request.name
    };
    let id = ids
        .allocate_part_design_view()
        .map_err(|cause| Error(cause.to_string()))?;
    views.copy(request.source_id, id, &name).map_err(Error)?;
    Ok(id)
}
pub(crate) fn info(view: &PartDesignView) -> PartDesignViewInfo {
    PartDesignViewInfo {
        id: view.id(),
        name: view.name(),
        built_in: view.built_in(),
        primary: view.primary(),
        active: view.active(),
        locked: view.locked(),
        published: view.published(),
        autosave_camera: view.autosave_camera(),
        annotation_scale: view.annotation_scale(),
        standard_orientation: view.standard_orientation(),
        camera: view.camera().cloned().map(Box::new),
        hidden_body_keys: Some(view.hidden_body_keys().iter().cloned().collect()),
    }
}
pub(super) fn list<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
) -> Result<PartDesignViewsResult, Error> {
    let views = part_views(workspace)?;
    Ok(PartDesignViewsResult {
        views: Some(views.list().iter().map(info).collect()),
    })
}

pub(super) fn camera<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    request: PartDesignViewRef,
) -> Result<oblikovati_api::types::PartDesignViewCamera, Error> {
    part_views(workspace)?
        .get(request.id)
        .map_err(Error)?
        .camera()
        .cloned()
        .ok_or_else(|| {
            Error(format!(
                "part view ID {}: expected a captured camera",
                request.id
            ))
        })
}

fn part_views<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
) -> Result<&PartDesignViews, Error> {
    let id = workspace
        .active_document()
        .ok_or_else(|| Error("part view query: expected an active part document".into()))?;
    let session = workspace
        .session(id)
        .map_err(|cause| Error(cause.to_string()))?;
    session
        .snapshot()
        .part_design_views
        .as_ref()
        .ok_or_else(|| Error(format!("document {id}: expected part view content")))
}
