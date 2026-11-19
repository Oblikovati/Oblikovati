// SPDX-License-Identifier: GPL-2.0-only
//! Inject the active desktop display into part-view transitions (#4118).
use oblikovati_api::{
    Error,
    contract::PartDesignView as _,
    types::{PartDesignViewCamera, ViewOrientationTypeEnum},
    wire::{CopyPartDesignViewArgs, PartDesignViewRef, PartDesignViewResult},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// A desktop camera boundary for atomic view changes.
/// Example: inject an owner with `host.with_part_view_display(&mut display)`.
pub trait PartViewDisplayAccess {
    /// Read the complete live camera; example: `display.camera()?` before a switch.
    fn camera(&mut self) -> Result<PartDesignViewCamera, String>;
    /// Resolve a standard orientation against the live camera; example: `display.orient(&camera, orientation)?`.
    fn orient(
        &mut self,
        camera: &PartDesignViewCamera,
        orientation: ViewOrientationTypeEnum,
    ) -> Result<PartDesignViewCamera, String>;
    /// Apply one camera; example: `display.apply(&camera)?` before committing a switch.
    fn apply(&mut self, camera: &PartDesignViewCamera) -> Result<(), String>;
}

pub(crate) fn copy<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    display: &mut Option<&mut dyn PartViewDisplayAccess>,
    request: CopyPartDesignViewArgs,
) -> Result<Vec<u8>, Error> {
    let document = crate::part_owner::active_part(files)?;
    let session = files
        .workspace()
        .session(document)
        .map_err(|cause| Error(cause.to_string()))?;
    let views = session
        .snapshot()
        .part_design_views
        .as_ref()
        .ok_or_else(|| Error("part view copy: expected part views".into()))?;
    let source = views.get(request.source_id).map_err(Error)?;
    let capture = source.active() && source.autosave_camera() && !source.locked();
    let camera = if capture {
        Some(
            display
                .as_deref_mut()
                .ok_or_else(|| Error("part view copy: expected an active display".into()))?
                .camera()
                .map_err(Error)?,
        )
    } else {
        None
    };
    files
        .commands(document)
        .map_err(|cause| Error(cause.to_string()))?
        .run_projected(
            "Copy Part View",
            |snapshot, ids| {
                let views = snapshot
                    .part_design_views
                    .as_mut()
                    .ok_or_else(|| Error("part view copy: expected retained part views".into()))?;
                if let Some(camera) = camera {
                    views
                        .capture_camera(request.source_id, camera)
                        .map_err(Error)?;
                }
                crate::part_design_views::copy_view(views, ids, request)
            },
            |snapshot, id| {
                let views = snapshot
                    .part_design_views
                    .as_ref()
                    .ok_or_else(|| Error("part view copy: expected retained views".into()))?;
                crate::encode(&PartDesignViewResult {
                    view: crate::part_design_views::info(views.get(id).map_err(Error)?),
                })
            },
        )
        .map_err(|cause| Error(cause.to_string()))
}

pub(crate) fn activate<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    topology: &mut Option<&mut dyn crate::PartTopologyAccess<O, W, F>>,
    display: &mut Option<&mut dyn PartViewDisplayAccess>,
    request: PartDesignViewRef,
) -> Result<Vec<u8>, Error> {
    let document = crate::part_owner::active_part(files)?;
    let available = crate::part_view_body_visibility::resolved_keys(files, topology, document)?;
    let display = display
        .as_deref_mut()
        .ok_or_else(|| Error("part view activation: expected an active desktop display".into()))?;
    let original = display.camera().map_err(Error)?;
    let prepared = files
        .commands(document)
        .map_err(|cause| Error(cause.to_string()))?
        .prepare_part_view_activation(request.id, original.clone(), &available)
        .map_err(Error)?;
    let target = match prepared.camera() {
        Some(saved) => saved.clone(),
        None => display
            .orient(&original, prepared.orientation())
            .map_err(Error)?,
    };
    let views = files
        .workspace()
        .session(document)
        .map_err(|cause| Error(cause.to_string()))?
        .snapshot()
        .part_design_views
        .as_ref()
        .ok_or_else(|| Error("part view activation: expected retained views".into()))?;
    let mut view = crate::part_design_views::info(views.get(request.id).map_err(Error)?);
    view.active = true;
    let reply = crate::encode(&PartDesignViewResult { view })?;
    apply_display_transaction(display, &original, &target, || {
        files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .commit_part_view_activation(prepared, target.clone())
    })?;
    Ok(reply)
}
fn apply_display_transaction(
    display: &mut dyn PartViewDisplayAccess,
    original: &PartDesignViewCamera,
    target: &PartDesignViewCamera,
    commit: impl FnOnce() -> Result<(), String>,
) -> Result<(), Error> {
    if let Err(cause) = display.apply(target) {
        return Err(restore_camera(display, original, &cause));
    }
    if let Err(cause) = commit() {
        return Err(restore_camera(display, original, &cause));
    }
    Ok(())
}
fn restore_camera(
    display: &mut dyn PartViewDisplayAccess,
    original: &PartDesignViewCamera,
    cause: &str,
) -> Error {
    match display.apply(original) {
        Ok(()) => Error(cause.into()),
        Err(restore) => Error(format!(
            "view activation failed: {cause}; camera restore failed: {restore}"
        )),
    }
}

pub(crate) fn navigate<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    topology: &mut Option<&mut dyn crate::PartTopologyAccess<O, W, F>>,
    display: &mut Option<&mut dyn PartViewDisplayAccess>,
    document: u64,
    direction: oblikovati_session::PartViewNavigationDirection,
) -> Result<(), Error> {
    if files.workspace().active_document() != Some(document) {
        return Err(Error(format!(
            "part view history document {document}: expected the active displayed document"
        )));
    }
    let available = crate::part_view_body_visibility::resolved_keys(files, topology, document)?;
    let prepared = files
        .commands(document)
        .map_err(|cause| Error(cause.to_string()))?
        .prepare_part_view_navigation(direction, &available)
        .map_err(Error)?;
    let target = prepared.camera().cloned();
    let Some(target) = target else {
        return files
            .commands(document)
            .map_err(|cause| Error(cause.to_string()))?
            .commit_part_view_navigation(prepared, None)
            .map_err(Error);
    };
    let display = display
        .as_deref_mut()
        .ok_or_else(|| Error("part view history: expected an active desktop display".into()))?;
    let original = display.camera().map_err(Error)?;
    apply_display_transaction(display, &original, &target, || {
        files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .commit_part_view_navigation(prepared, Some(&target))
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    struct FakeDisplay {
        current: PartDesignViewCamera,
        fail_next: bool,
    }
    impl PartViewDisplayAccess for FakeDisplay {
        fn camera(&mut self) -> Result<PartDesignViewCamera, String> {
            Ok(self.current.clone())
        }
        fn orient(
            &mut self,
            camera: &PartDesignViewCamera,
            _: ViewOrientationTypeEnum,
        ) -> Result<PartDesignViewCamera, String> {
            Ok(camera.clone())
        }
        fn apply(&mut self, camera: &PartDesignViewCamera) -> Result<(), String> {
            self.current = camera.clone();
            if std::mem::take(&mut self.fail_next) {
                return Err("display failed after changing camera".into());
            }
            Ok(())
        }
    }
    #[test]
    fn failed_display_or_commit_restores_the_previous_camera() {
        let before = PartDesignViewCamera {
            scale: 10.,
            ..Default::default()
        };
        let after = PartDesignViewCamera {
            scale: 20.,
            ..Default::default()
        };
        let mut display = FakeDisplay {
            current: before.clone(),
            fail_next: true,
        };
        let mut called = false;
        assert!(
            apply_display_transaction(&mut display, &before, &after, || {
                called = true;
                Ok(())
            })
            .is_err()
        );
        assert!(!called);
        assert_eq!(display.current, before);
        assert!(
            apply_display_transaction(&mut display, &before, &after, || Err(
                "commit refused".into()
            ))
            .is_err()
        );
        assert_eq!(display.current, before);
    }
}
