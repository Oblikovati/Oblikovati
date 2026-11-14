// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{offset::redefine, planes::plane_rows};

fn request(line: &str, point: &str) -> CreateWorkPlaneArgs {
    CreateWorkPlaneArgs {
        kind: WORK_PLANE_LINE_AND_POINT.0.into_owned(),
        refs: Some(vec![line.into(), point.into()]),
        ..Default::default()
    }
}
fn create(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    line: &str,
    point: &str,
) -> CreateWorkPlaneResult {
    Client::new(FileHost::new(files, source, "test"))
        .work_planes()
        .create(&request(line, point))
        .unwrap()
}
fn repick(line: &str, point: &str) -> RedefineWorkPlaneArgs {
    RedefineWorkPlaneArgs {
        index: 3,
        repick: Some(vec![
            SlotRepick {
                slot: 0,
                ref_: line.into(),
            },
            SlotRepick {
                slot: 1,
                ref_: point.into(),
            },
        ]),
        ..Default::default()
    }
}
fn assert_normal(files: &DiscoveryWorkspace, normal: [f64; 3]) {
    assert_eq!(plane_rows(files, true)[3].normal, Some(normal.to_vec()));
}
#[test]
fn canonical_creation_resolves_origin_axes_and_reports_the_go_reference_slots() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [0.0, 1.0, 0.0], false);
    let created = create(&mut files, &mut source, WORK_REF_X_AXIS, &point.ref_);
    assert!(created.healthy);
    let plane = plane_rows(&files, true).remove(3);
    assert_eq!(plane.kind, WORK_PLANE_LINE_AND_POINT.0);
    assert_normal(&files, [0.0, 0.0, 1.0]);
    let slots = plane.slots.unwrap();
    let actual = slots
        .iter()
        .map(|slot| (slot.index, slot.label.as_str(), slot.kind.as_str()))
        .collect::<Vec<_>>();
    assert_eq!(actual, [(0, "Line", "axis"), (1, "Through point", "point")]);
    assert!(plane.scalars.is_none());
}
#[test]
fn user_axis_creation_keeps_its_current_origin() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let axis = create_line(&mut files, &mut source);
    let created = create(&mut files, &mut source, &axis.ref_, WORK_REF_CENTER);
    assert!(created.healthy);
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![2.0, 3.0, 4.0])
    );
}
#[test]
fn all_origin_axis_references_produce_the_expected_oriented_planes() {
    for (axis, at, normal) in [
        (WORK_REF_X_AXIS, [0.0, 1.0, 0.0], [0.0, 0.0, 1.0]),
        (WORK_REF_Y_AXIS, [0.0, 0.0, 1.0], [1.0, 0.0, 0.0]),
        (WORK_REF_Z_AXIS, [1.0, 0.0, 0.0], [0.0, 1.0, 0.0]),
    ] {
        let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
        let point = create_point(&mut files, &mut source, at, false);
        assert!(create(&mut files, &mut source, axis, &point.ref_).healthy);
        assert_normal(&files, normal);
    }
}
#[test]
fn repicks_and_display_edits_share_one_history_step_and_keep_plane_identity() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let first = create_point(&mut files, &mut source, [0.0, 1.0, 0.0], false);
    let next = create_point(&mut files, &mut source, [1.0, 0.0, 0.0], false);
    let created = create(&mut files, &mut source, WORK_REF_X_AXIS, &first.ref_);
    let mut args = repick(WORK_REF_Y_AXIS, &next.ref_);
    args.grounded = Some(std::boxed::Box::new(true));
    let reply = redefine(&mut files, &mut source, &args).unwrap();
    assert_eq!(reply.plane.ref_, created.ref_);
    assert!(reply.plane.grounded);
    assert_normal(&files, [0.0, 0.0, -1.0]);
    files.commands(document).unwrap().undo().unwrap();
    assert_normal(&files, [0.0, 0.0, 1.0]);
    assert!(!plane_rows(&files, true)[3].grounded);
    files.commands(document).unwrap().redo().unwrap();
    assert_normal(&files, [0.0, 0.0, -1.0]);
}
#[test]
fn wrong_reference_kinds_and_missing_refs_are_refused_before_allocation() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let before = owner_revision(&files, document);
    let mut missing = request(WORK_REF_X_AXIS, WORK_REF_CENTER);
    missing.refs = None;
    for args in [
        missing,
        request(WORK_REF_CENTER, WORK_REF_CENTER),
        request(WORK_REF_X_AXIS, WORK_REF_XY_PLANE),
        request("bad", WORK_REF_CENTER),
    ] {
        assert!(
            Client::new(FileHost::new(&mut files, &mut source, "test"))
                .work_planes()
                .create(&args)
                .is_err()
        );
        assert_eq!(owner_revision(&files, document), before);
    }
}
#[test]
fn bad_second_repick_duplicate_slots_and_scalar_edits_are_atomic_declines() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [0.0, 1.0, 0.0], false);
    create(&mut files, &mut source, WORK_REF_X_AXIS, &point.ref_);
    let before = owner_revision(&files, document);
    for mut args in invalid_revisions() {
        args.grounded = Some(std::boxed::Box::new(true));
        assert!(redefine(&mut files, &mut source, &args).is_err());
        assert_eq!(owner_revision(&files, document), before);
        assert_normal(&files, [0.0, 0.0, 1.0]);
    }
}
fn invalid_revisions() -> Vec<RedefineWorkPlaneArgs> {
    let mut duplicate = repick(WORK_REF_Z_AXIS, WORK_REF_CENTER);
    duplicate.repick.as_mut().unwrap()[1].slot = 0;
    let mut scalar = repick(WORK_REF_Z_AXIS, WORK_REF_CENTER);
    scalar.scalars = Some(vec![ScalarEdit {
        index: 0,
        value: "1 cm".into(),
    }]);
    vec![
        repick(WORK_REF_Z_AXIS, "bad"),
        duplicate,
        scalar,
        repick(WORK_REF_XY_PLANE, WORK_REF_CENTER),
    ]
}
#[test]
fn collinear_creation_keeps_local_health_and_single_point_repick_recovers() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [0.0, 1.0, 0.0], false);
    assert!(!create(&mut files, &mut source, WORK_REF_X_AXIS, WORK_REF_CENTER).healthy);
    let failed = plane_rows(&files, true).remove(3);
    assert!(!failed.healthy && failed.origin.is_none() && failed.normal.is_none());
    assert!(!failed.reason.is_empty());
    let mut args = repick(WORK_REF_X_AXIS, &point.ref_);
    args.repick.as_mut().unwrap().remove(0);
    assert!(
        redefine(&mut files, &mut source, &args)
            .unwrap()
            .plane
            .healthy
    );
    assert_normal(&files, [0.0, 0.0, 1.0]);
}
#[test]
fn forward_axis_repick_cannot_publish_source_or_display_changes() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [0.0, 1.0, 0.0], false);
    create(&mut files, &mut source, WORK_REF_X_AXIS, &point.ref_);
    let later = create_line(&mut files, &mut source);
    let before = owner_revision(&files, document);
    let mut args = repick(&later.ref_, WORK_REF_CENTER);
    args.grounded = Some(std::boxed::Box::new(true));
    assert!(redefine(&mut files, &mut source, &args).is_err());
    assert_eq!(owner_revision(&files, document), before);
    assert_normal(&files, [0.0, 0.0, 1.0]);
}
#[test]
fn save_reopen_copy_and_single_axis_repick_keep_sources_and_authored_flip() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [0.0, 1.0, 0.0], false);
    let created = create(&mut files, &mut source, WORK_REF_X_AXIS, &point.ref_);
    flip(&mut files, &mut source);
    files.save(document, "test", &mut source).unwrap();
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_normal(&files, [0.0, 0.0, -1.0]);
    copy_and_open(&mut files, document, &mut source);
    assert_normal(&files, [0.0, 0.0, -1.0]);
    assert_eq!(plane_rows(&files, true)[3].ref_, created.ref_);
    let mut args = repick(WORK_REF_Z_AXIS, WORK_REF_CENTER);
    args.repick.as_mut().unwrap().remove(1);
    redefine(&mut files, &mut source, &args).unwrap();
    assert_normal(&files, [1.0, 0.0, 0.0]);
}
fn flip(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    Client::new(FileHost::new(files, source, "test"))
        .work_planes()
        .flip_normal(&FlipWorkPlaneArgs { index: 3 })
        .unwrap();
}
fn copy_and_open(files: &mut DiscoveryWorkspace, document: u64, source: &mut QueryGuids) {
    let path = Path::new("/fixtures/line-point-copy.obk");
    files
        .save_copy(document, path, None, "test", source)
        .unwrap();
    super::history::open_copied_datum(files, source, path);
}
