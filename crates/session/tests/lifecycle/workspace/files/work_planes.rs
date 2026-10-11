// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId,
        work_plane::{
            WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneOrigin, WorkPlaneReference,
            WorkPlaneResolver,
        },
    },
    history::RecipeReservations,
};
fn add_plane(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<FeatureId, String> {
    let id = ids.allocate_feature().map_err(|error| error.to_string())?;
    let definition = WorkPlaneDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: WorkPlaneDistance::Parameter(1),
    };
    snapshot
        .work_planes
        .append(AuthoredFeature::work_plane(id, "Driven", definition))
        .map_err(|error| error.to_string())?;
    Ok(id)
}
fn snapshot(host: &Host, id: u64) -> &DocumentSnapshot {
    host.workspace().session(id).unwrap().snapshot()
}
fn assert_plane_height(host: &Host, id: u64, plane: FeatureId, expected: f64) {
    let snapshot = snapshot(host, id);
    let evaluated = snapshot
        .work_planes
        .evaluate_work_planes(&snapshot.parameters)
        .unwrap();
    assert_eq!(evaluated.plane(plane).unwrap().recipe().origin[2], expected);
}
fn create_plane(host: &mut Host, id: u64) -> FeatureId {
    host.commands(id)
        .unwrap()
        .run("Create plane", add_plane)
        .unwrap()
}
fn reopen(host: &mut Host, id: u64, source: &mut SequenceGuids) -> u64 {
    host.close_discarding_changes(id).unwrap();
    open_file(host, "a.obk", source, OpenActivation::Activate)
}
#[test]
fn saved_program_reopens_and_parameter_edits_follow_undo_and_redo() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let plane = create_plane(&mut host, id);
    assert_plane_height(&host, id, plane, 1.0);
    host.save(id, "alpha", &mut source).unwrap();
    let before = snapshot(&host, id).work_planes.clone();
    let id = reopen(&mut host, id, &mut source);
    assert!(
        snapshot(&host, id)
            .work_planes
            .same_work_plane_recipe(&before)
    );
    assert!(!host.workspace().needs_save(id).unwrap());
    edit_and_navigate(&mut host, id, plane);
    copy_and_verify(&mut host, id, plane, &mut source);
}
fn edit_and_navigate(host: &mut Host, id: u64, plane: FeatureId) {
    host.commands(id)
        .unwrap()
        .run("Move", |snapshot, _| {
            snapshot.parameters.set_expression(1, "5 cm")
        })
        .unwrap();
    assert_plane_height(host, id, plane, 5.0);
    host.commands(id).unwrap().undo().unwrap();
    assert_plane_height(host, id, plane, 1.0);
    assert!(!host.workspace().needs_save(id).unwrap());
    host.commands(id).unwrap().redo().unwrap();
    assert_plane_height(host, id, plane, 5.0);
    assert!(host.workspace().needs_save(id).unwrap());
}
fn copy_and_verify(host: &mut Host, id: u64, plane: FeatureId, source: &mut SequenceGuids) {
    let source_identity = host.workspace().identity(id).unwrap().clone();
    host.save_copy(id, &path("plane-copy.obk"), None, "alpha", source)
        .unwrap();
    let copied = open_file(host, "plane-copy.obk", source, OpenActivation::Background);
    assert_plane_height(host, copied, plane, 5.0);
    assert_eq!(create_plane(host, copied), FeatureId(2));
    assert_eq!(host.workspace().identity(id).unwrap(), &source_identity);
    assert!(host.workspace().needs_save(id).unwrap());
}
fn reject_reserved_plane(host: &mut Host, id: u64) {
    let result = host.commands(id).unwrap().run("Rejected", |_, ids| {
        assert_eq!(ids.allocate_feature().unwrap(), FeatureId(1));
        Err::<(), _>("Rejected")
    });
    assert!(result.is_err());
    assert!(snapshot(host, id).work_planes.features().is_empty());
    assert!(
        host.workspace()
            .session(id)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
    assert!(host.workspace().needs_save(id).unwrap());
}
#[test]
fn failed_and_deleted_reservations_remain_dirty_and_survive_save_reopen() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    reject_reserved_plane(&mut host, id);
    host.save(id, "alpha", &mut source).unwrap();
    assert!(!host.workspace().needs_save(id).unwrap());
    let id = reopen(&mut host, id, &mut source);
    let plane = create_plane(&mut host, id);
    assert_eq!(plane, FeatureId(2));
    host.commands(id)
        .unwrap()
        .run("Delete plane", |snapshot, _| {
            snapshot.work_planes.remove(plane)
        })
        .unwrap();
    host.save(id, "alpha", &mut source).unwrap();
    let id = reopen(&mut host, id, &mut source);
    assert_eq!(create_plane(&mut host, id), FeatureId(3));
}
#[test]
fn created_document_keeps_template_program_and_deleted_reservations() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    create_plane(&mut host, id);
    host.save(id, "alpha", &mut source).unwrap();
    let package = DocumentPackage::decode(&host.files().contents.borrow()[&path("a.obk")]).unwrap();
    let created = host
        .create_unbound(package, RecordingObserver::default(), "alpha", &mut source)
        .unwrap()
        .session_id;
    assert!(
        snapshot(&host, created)
            .work_planes
            .same_work_plane_recipe(&snapshot(&host, id).work_planes)
    );
    assert_eq!(create_plane(&mut host, created), FeatureId(2));
    assert_eq!(snapshot(&host, id).work_planes.features().len(), 1);
}
#[test]
fn failed_storage_keeps_source_baseline_and_current_program() {
    let files = WorkspaceFiles {
        fail_write: true,
        ..Default::default()
    };
    files
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(1));
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), files);
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    create_plane(&mut host, id);
    let original = host.files().contents.borrow()[&path("a.obk")].clone();
    let identity = host.workspace().identity(id).unwrap().clone();

    assert!(host.save(id, "alpha", &mut source).is_err());
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], original);
    assert_failed_save_state(&host, id, &identity);
}
fn assert_failed_save_state(host: &Host, id: u64, identity: &FileIdentity) {
    assert_eq!(host.workspace().identity(id).unwrap(), identity);
    assert!(host.workspace().needs_save(id).unwrap());
    assert_eq!(snapshot(host, id).work_planes.features().len(), 1);
}
fn replace_distance(host: &mut Host, id: u64, plane: FeatureId, distance: f64) {
    let definition = WorkPlaneDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: WorkPlaneDistance::Constant(distance),
    };
    host.commands(id)
        .unwrap()
        .run("Distance", |snapshot, _| {
            snapshot
                .work_planes
                .replace(plane, definition, definition.reads())
        })
        .unwrap();
}
#[test]
fn signed_zero_edit_is_an_undo_step_and_changes_the_saved_baseline() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let plane = create_plane(&mut host, id);
    replace_distance(&mut host, id, plane, 0.0);
    host.save(id, "alpha", &mut source).unwrap();
    replace_distance(&mut host, id, plane, -0.0);
    assert!(host.workspace().needs_save(id).unwrap());
    host.commands(id).unwrap().undo().unwrap();
    assert!(!host.workspace().needs_save(id).unwrap());
    host.commands(id).unwrap().redo().unwrap();
    assert!(host.workspace().needs_save(id).unwrap());
    host.save(id, "alpha", &mut source).unwrap();
    let before = snapshot(&host, id).work_planes.clone();
    let id = reopen(&mut host, id, &mut source);
    assert!(before.same_work_plane_recipe(&snapshot(&host, id).work_planes));
}
#[test]
fn unsupported_legacy_feature_content_is_refused_before_document_registration() {
    let mut host = host(RecordingWorkspace::default());
    let input = b"schemaVersion: 2\ndocumentType: 1\nmodel: {features: [{kind: extrude}]}\n";
    host.files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), input.to_vec());
    let error = host
        .open(
            &path("a.obk"),
            RecordingObserver::default(),
            "alpha",
            &mut SequenceGuids(1000),
            OpenActivation::Activate,
        )
        .unwrap_err();
    assert!(error.to_string().contains("explicit legacy feature import"));
    assert!(host.workspace().documents().is_empty());
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], input);
}

#[path = "work_planes/linked_sketch.rs"]
mod linked_sketch;
