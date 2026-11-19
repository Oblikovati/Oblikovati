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
    policy: Option<Option<oblikovati_martensite::PrismResolutionPolicy>>,
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
    crate::part_view_replay::run(
        files,
        document,
        policy,
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
            let id = crate::part_design_views::copy_view(views, ids, request)?;
            crate::encode(&PartDesignViewResult {
                view: crate::part_design_views::info(views.get(id).map_err(Error)?),
            })
        },
    )
}

pub(crate) fn activate<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    topology: &mut Option<&mut dyn crate::PartTopologyAccess<O, W, F>>,
    display: &mut Option<&mut dyn PartViewDisplayAccess>,
    request: PartDesignViewRef,
) -> Result<Vec<u8>, Error> {
    let document = crate::part_owner::active_part(files)?;
    let available = crate::part_view_body_visibility::resolved_keys(files, topology, document)?;
    let policy = topology.as_deref().map(|owner| owner.resolution_policy());
    let replay = policy.and_then(|policy| {
        files
            .workspace()
            .owned_part_replay(document, policy)
            .ok()
            .map(|replay| (policy, replay))
    });
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
        let Some((policy, replay)) = replay else {
            return files
                .commands(document)
                .map_err(|cause| cause.to_string())?
                .commit_part_view_activation(prepared, target.clone());
        };
        let (candidate, transition) = files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .commit_part_view_activation_projected(
                prepared,
                target.clone(),
                |candidate, history| {
                    replay
                        .capture_unchanged(history, candidate)
                        .map_err(|cause| cause.to_string())
                },
            )?;
        crate::part_view_replay::adopt(files, document, policy, candidate, &transition);
        Ok(())
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
    let policy = topology.as_deref().map(|owner| owner.resolution_policy());
    let replay = policy.and_then(|policy| {
        files
            .workspace()
            .owned_part_replay(document, policy)
            .ok()
            .map(|replay| (policy, replay))
    });
    let prepared = files
        .commands(document)
        .map_err(|cause| Error(cause.to_string()))?
        .prepare_part_view_navigation(direction, &available)
        .map_err(Error)?;
    let target = prepared.camera().cloned();
    let Some(target) = target else {
        return commit_navigation(files, document, prepared, None, replay).map_err(Error);
    };
    let display = display
        .as_deref_mut()
        .ok_or_else(|| Error("part view history: expected an active desktop display".into()))?;
    let original = display.camera().map_err(Error)?;
    apply_display_transaction(display, &original, &target, || {
        commit_navigation(files, document, prepared, Some(&target), replay)
    })
}

fn commit_navigation<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    prepared: oblikovati_session::PreparedPartViewNavigation,
    camera: Option<&PartDesignViewCamera>,
    replay: Option<(
        Option<oblikovati_martensite::PrismResolutionPolicy>,
        oblikovati_session::OwnedPartReplay,
    )>,
) -> Result<(), String> {
    let Some((policy, replay)) = replay else {
        return files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .commit_part_view_navigation(prepared, camera);
    };
    let (candidate, transition) = files
        .commands(document)
        .map_err(|cause| cause.to_string())?
        .commit_part_view_navigation_projected(prepared, camera, |candidate, history| {
            replay
                .capture_unchanged(history, candidate)
                .map_err(|cause| cause.to_string())
        })?;
    crate::part_view_replay::adopt(files, document, policy, candidate, &transition);
    Ok(())
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
