// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{offset::redefine, planes::plane_rows};
fn request(base: &str, point: &str) -> CreateWorkPlaneArgs {
    CreateWorkPlaneArgs {
        kind: WORK_PLANE_PLANE_AND_POINT.0.into_owned(),
        refs: Some(vec![base.into(), point.into()]),
        ..Default::default()
    }
}
fn create(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    base: &str,
    point: &str,
) -> CreateWorkPlaneResult {
    Client::new(FileHost::new(files, source, "test"))
        .work_planes()
        .create(&request(base, point))
        .unwrap()
}
fn repick(index: i64, base: &str, point: &str) -> RedefineWorkPlaneArgs {
    RedefineWorkPlaneArgs {
        index,
        repick: Some(vec![
            SlotRepick {
                slot: 0,
                ref_: base.into(),
            },
            SlotRepick {
                slot: 1,
                ref_: point.into(),
            },
        ]),
        ..Default::default()
    }
}
#[test]
fn canonical_creation_reports_both_reference_slots_and_current_geometry() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [5.0, 6.0, 7.0], false);
    let created = create(&mut files, &mut source, WORK_REF_XY_PLANE, &point.ref_);
    assert!(created.healthy);
    let rows = plane_rows(&files, true);
    assert_eq!(rows[3].kind, WORK_PLANE_PLANE_AND_POINT.0);
    assert_eq!(rows[3].origin, Some(vec![5.0, 6.0, 7.0]));
    assert_eq!(rows[3].normal, Some(vec![0.0, 0.0, 1.0]));
    assert_slots(&rows[3]);
    assert!(rows[3].scalars.is_none());
}
#[test]
fn both_reference_edits_form_one_command_preserve_identity_and_undo() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [5.0, 6.0, 7.0], false);
    let created = create(&mut files, &mut source, WORK_REF_XY_PLANE, &point.ref_);
    let reply = redefine(
        &mut files,
        &mut source,
        &repick(3, WORK_REF_YZ_PLANE, WORK_REF_CENTER),
    )
    .unwrap();
    assert_eq!(reply.plane.ref_, created.ref_);
    assert_eq!(reply.plane.origin, Some(vec![0.0; 3]));
    assert_eq!(reply.plane.normal, Some(vec![1.0, 0.0, 0.0]));
    files.commands(document).unwrap().undo().unwrap();
    assert_origin(&files, [5.0, 6.0, 7.0]);
    files.commands(document).unwrap().redo().unwrap();
    assert_origin(&files, [0.0; 3]);
}
#[test]
fn wrong_reference_kinds_decline_before_allocation() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [0.0; 3], false);
    let before = owner_revision(&files, document);
    for args in [
        request(&point.ref_, WORK_REF_CENTER),
        request(WORK_REF_XY_PLANE, WORK_REF_XY_PLANE),
    ] {
        let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
            .work_planes()
            .create(&args);
        assert!(result.is_err());
        assert_eq!(owner_revision(&files, document), before);
    }
}
#[test]
fn duplicate_slots_and_scalar_edits_cannot_publish_display_flags() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut source, WORK_REF_XY_PLANE, WORK_REF_CENTER);
    let before = owner_revision(&files, document);
    for mut args in malformed_edits() {
        args.grounded = Some(std::boxed::Box::new(true));
        assert!(redefine(&mut files, &mut source, &args).is_err());
        assert_eq!(owner_revision(&files, document), before);
        assert!(!plane_rows(&files, true)[3].grounded);
    }
}
fn malformed_edits() -> Vec<RedefineWorkPlaneArgs> {
    let mut duplicate = repick(3, WORK_REF_XZ_PLANE, WORK_REF_CENTER);
    duplicate.repick.as_mut().unwrap()[1].slot = 0;
    let mut invalid = repick(3, WORK_REF_XZ_PLANE, WORK_REF_CENTER);
    invalid.repick.as_mut().unwrap()[1].slot = 2;
    let mut scalar = RedefineWorkPlaneArgs {
        index: 3,
        ..Default::default()
    };
    scalar.scalars = Some(vec![ScalarEdit {
        index: 0,
        value: "2 cm".into(),
    }]);
    vec![duplicate, invalid, scalar]
}
#[test]
fn a_bad_second_reference_cannot_publish_a_valid_first_reference() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut source, WORK_REF_XY_PLANE, WORK_REF_CENTER);
    let before = owner_revision(&files, document);
    assert!(
        redefine(
            &mut files,
            &mut source,
            &repick(3, WORK_REF_XZ_PLANE, WORK_REF_YZ_PLANE)
        )
        .is_err()
    );
    assert_eq!(owner_revision(&files, document), before);
    assert_eq!(
        plane_rows(&files, true)[3].normal,
        Some(vec![0.0, 0.0, 1.0])
    );
}
#[test]
fn same_references_are_no_ops_and_self_dependencies_are_refused() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = create(&mut files, &mut source, WORK_REF_XY_PLANE, WORK_REF_CENTER);
    let before = owner_revision(&files, document);
    redefine(
        &mut files,
        &mut source,
        &repick(3, WORK_REF_XY_PLANE, WORK_REF_CENTER),
    )
    .unwrap();
    assert_eq!(owner_revision(&files, document), before);
    let invalid = repick(3, &created.ref_, WORK_REF_CENTER);
    assert!(redefine(&mut files, &mut source, &invalid).is_err());
    assert_eq!(owner_revision(&files, document), before);
}
#[test]
fn reopen_and_copy_preserve_both_source_references_and_normal_flip() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [5.0, 6.0, 7.0], false);
    let created = create(&mut files, &mut source, WORK_REF_XZ_PLANE, &point.ref_);
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .work_planes()
        .flip_normal(&FlipWorkPlaneArgs { index: 3 })
        .unwrap();
    files.save(document, "test", &mut source).unwrap();
    let document =
        reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    let path = Path::new("/fixtures/parallel-copy.obk");
    files
        .save_copy(document, path, None, "test", &mut source)
        .unwrap();
    super::history::open_copied_datum(&mut files, &mut source, path);
    assert_copied_plane(&files, &created.ref_);
}
#[test]
fn empty_creation_references_are_a_named_decline() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let mut args = request(WORK_REF_XY_PLANE, WORK_REF_CENTER);
    args.refs = None;
    let error = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .work_planes()
        .create(&args)
        .unwrap_err();
    assert!(error.to_string().contains("exactly two references"));
}

fn assert_origin(files: &DiscoveryWorkspace, at: [f64; 3]) {
    assert_eq!(plane_rows(files, true)[3].origin, Some(at.to_vec()));
}

#[test]
fn single_through_point_edit_keeps_the_base_and_display_edits_share_the_command() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [5.0, 6.0, 7.0], false);
    create(&mut files, &mut source, WORK_REF_XZ_PLANE, WORK_REF_CENTER);
    let mut args = repick(3, WORK_REF_XZ_PLANE, &point.ref_);
    args.repick.as_mut().unwrap().remove(0);
    args.auto_resize = Some(std::boxed::Box::new(true));
    let reply = redefine(&mut files, &mut source, &args).unwrap();
    assert!(reply.plane.auto_resize);
    assert_eq!(reply.plane.normal, Some(vec![0.0, -1.0, 0.0]));
    assert_origin(&files, [5.0, 6.0, 7.0]);
    files.commands(document).unwrap().undo().unwrap();
    assert_origin(&files, [0.0; 3]);
    assert!(!plane_rows(&files, true)[3].auto_resize);
}

fn assert_slots(plane: &WorkPlaneInfo) {
    let slots = plane.slots.as_ref().unwrap();
    let actual = slots
        .iter()
        .map(|slot| (slot.index, slot.kind.as_str()))
        .collect::<Vec<_>>();
    assert_eq!(actual, [(0, "plane"), (1, "point")]);
}
fn assert_copied_plane(files: &DiscoveryWorkspace, reference: &str) {
    let plane = &plane_rows(files, true)[3];
    assert_eq!(plane.ref_, reference);
    assert_eq!(plane.origin, Some(vec![5.0, 6.0, 7.0]));
    assert_eq!(plane.normal, Some(vec![0.0, 1.0, 0.0]));
}
#[test]
fn group_abort_restores_the_previous_reference_source() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut source, WORK_REF_XY_PLANE, WORK_REF_CENTER);
    let before = plane_rows(&files, true);
    files
        .commands(document)
        .unwrap()
        .begin("Plane group")
        .unwrap();
    redefine(
        &mut files,
        &mut source,
        &repick(3, WORK_REF_XZ_PLANE, WORK_REF_CENTER),
    )
    .unwrap();
    files.commands(document).unwrap().abort().unwrap();
    assert_eq!(plane_rows(&files, true), before);
}
#[test]
fn commit_veto_restores_the_previous_reference_source() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut source, WORK_REF_XY_PLANE, WORK_REF_CENTER);
    files.save(document, "test", &mut source).unwrap();
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let before = owner_revision(&files, document);
    let revision = repick(3, WORK_REF_XZ_PLANE, WORK_REF_CENTER);
    assert!(redefine(&mut files, &mut source, &revision).is_err());
    assert_eq!(owner_revision(&files, document).1, before.1);
    assert_eq!(
        plane_rows(&files, true)[3].normal,
        Some(vec![0.0, 0.0, 1.0])
    );
}
