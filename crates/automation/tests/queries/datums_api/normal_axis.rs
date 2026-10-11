// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{offset::redefine, planes::plane_rows};

fn request(axis: &str, point: &str) -> CreateWorkPlaneArgs {
    CreateWorkPlaneArgs {
        kind: WORK_PLANE_NORMAL_TO_CURVE.0.into_owned(),
        refs: Some(vec![axis.into(), point.into()]),
        ..Default::default()
    }
}
fn create(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    axis: &str,
    point: &str,
) -> CreateWorkPlaneResult {
    Client::new(FileHost::new(files, source, "test"))
        .work_planes()
        .create(&request(axis, point))
        .unwrap()
}
fn repick(axis: &str, point: &str) -> RedefineWorkPlaneArgs {
    RedefineWorkPlaneArgs {
        index: 3,
        repick: Some(vec![
            SlotRepick {
                slot: 0,
                ref_: axis.into(),
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
fn canonical_creation_lists_current_geometry_and_go_curve_point_slots() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [5.0, 6.0, 7.0], false);
    assert!(create(&mut files, &mut source, WORK_REF_Z_AXIS, &point.ref_).healthy);
    let plane = plane_rows(&files, true).remove(3);
    assert_eq!(plane.kind, WORK_PLANE_NORMAL_TO_CURVE.0);
    assert_eq!(plane.origin, Some(vec![5.0, 6.0, 7.0]));
    assert_normal(&files, [0.0, 0.0, 1.0]);
    let slots = plane.slots.unwrap();
    let actual = slots
        .iter()
        .map(|slot| (slot.index, slot.label.as_str(), slot.kind.as_str()))
        .collect::<Vec<_>>();
    assert_eq!(actual, [(0, "Curve", "axis"), (1, "Point", "point")]);
    assert!(plane.scalars.is_none());
}
#[test]
fn all_origin_axes_and_user_axis_use_the_point_origin() {
    for (axis, normal) in [
        (WORK_REF_X_AXIS, [1.0, 0.0, 0.0]),
        (WORK_REF_Y_AXIS, [0.0, 1.0, 0.0]),
        (WORK_REF_Z_AXIS, [0.0, 0.0, 1.0]),
    ] {
        let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
        assert!(create(&mut files, &mut source, axis, WORK_REF_CENTER).healthy);
        assert_normal(&files, normal);
    }
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let axis = create_line(&mut files, &mut source);
    assert!(create(&mut files, &mut source, &axis.ref_, WORK_REF_CENTER).healthy);
    assert_eq!(plane_rows(&files, true)[3].origin, Some(vec![0.0; 3]));
    assert_normal(&files, [0.0, 0.0, 1.0]);
}
#[test]
fn repicks_share_one_command_with_display_and_preserve_identity_through_undo() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [5.0, 6.0, 7.0], false);
    let created = create(&mut files, &mut source, WORK_REF_Z_AXIS, WORK_REF_CENTER);
    let mut args = repick(WORK_REF_X_AXIS, &point.ref_);
    args.grounded = Some(std::boxed::Box::new(true));
    let reply = redefine(&mut files, &mut source, &args).unwrap();
    assert_eq!(reply.plane.ref_, created.ref_);
    assert!(reply.plane.grounded);
    assert_eq!(reply.plane.origin, Some(vec![5.0, 6.0, 7.0]));
    assert_normal(&files, [1.0, 0.0, 0.0]);
    files.commands(document).unwrap().undo().unwrap();
    assert_normal(&files, [0.0, 0.0, 1.0]);
    assert!(!plane_rows(&files, true)[3].grounded);
    files.commands(document).unwrap().redo().unwrap();
    assert_normal(&files, [1.0, 0.0, 0.0]);
}
#[test]
fn malformed_creation_refs_decline_before_permanent_allocation() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let before = owner_revision(&files, document);
    let mut missing = request(WORK_REF_Z_AXIS, WORK_REF_CENTER);
    missing.refs = None;
    for args in [
        missing,
        request(WORK_REF_CENTER, WORK_REF_CENTER),
        request(WORK_REF_Z_AXIS, WORK_REF_XY_PLANE),
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
fn invalid_repick_cannot_publish_an_earlier_good_axis_or_display_change() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut source, WORK_REF_Z_AXIS, WORK_REF_CENTER);
    let before = owner_revision(&files, document);
    for mut args in invalid_revisions() {
        args.grounded = Some(std::boxed::Box::new(true));
        assert!(redefine(&mut files, &mut source, &args).is_err());
        assert_eq!(owner_revision(&files, document), before);
        assert_normal(&files, [0.0, 0.0, 1.0]);
        assert!(!plane_rows(&files, true)[3].grounded);
    }
}
fn invalid_revisions() -> Vec<RedefineWorkPlaneArgs> {
    let mut duplicate = repick(WORK_REF_X_AXIS, WORK_REF_CENTER);
    duplicate.repick.as_mut().unwrap()[1].slot = 0;
    let mut scalar = repick(WORK_REF_X_AXIS, WORK_REF_CENTER);
    scalar.scalars = Some(vec![ScalarEdit {
        index: 0,
        value: "1 cm".into(),
    }]);
    vec![
        repick(WORK_REF_X_AXIS, "bad"),
        duplicate,
        scalar,
        repick(WORK_REF_XY_PLANE, WORK_REF_CENTER),
    ]
}
#[test]
fn forward_axis_dependencies_are_atomic_declines() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut source, WORK_REF_Z_AXIS, WORK_REF_CENTER);
    let later = create_line(&mut files, &mut source);
    let before = owner_revision(&files, document);
    assert!(
        redefine(
            &mut files,
            &mut source,
            &repick(&later.ref_, WORK_REF_CENTER)
        )
        .is_err()
    );
    assert_eq!(owner_revision(&files, document), before);
    assert_normal(&files, [0.0, 0.0, 1.0]);
}
#[test]
fn single_axis_and_point_edits_keep_the_other_driver_and_unchanged_edits_are_noops() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [5.0, 6.0, 7.0], false);
    create(&mut files, &mut source, WORK_REF_Z_AXIS, &point.ref_);
    let before = owner_revision(&files, document);
    let unchanged = repick(WORK_REF_Z_AXIS, &point.ref_);
    redefine(&mut files, &mut source, &unchanged).unwrap();
    assert_eq!(owner_revision(&files, document), before);
    edit_driver(&mut files, &mut source, 0, WORK_REF_Y_AXIS);
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![5.0, 6.0, 7.0])
    );
    edit_driver(&mut files, &mut source, 1, WORK_REF_CENTER);
    assert_normal(&files, [0.0, 1.0, 0.0]);
    assert_eq!(plane_rows(&files, true)[3].origin, Some(vec![0.0; 3]));
}
#[test]
fn reopen_copy_and_normal_flip_preserve_current_sources() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [5.0, 6.0, 7.0], false);
    let created = create(&mut files, &mut source, WORK_REF_Z_AXIS, &point.ref_);
    flip(&mut files, &mut source);
    let document = save_and_reopen(&mut files, document, &mut source);
    assert_normal(&files, [0.0, 0.0, -1.0]);
    copy_and_open(&mut files, document, &mut source);
    assert_eq!(plane_rows(&files, true)[3].ref_, created.ref_);
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![5.0, 6.0, 7.0])
    );
    assert_normal(&files, [0.0, 0.0, -1.0]);
    let revised = repick(WORK_REF_X_AXIS, WORK_REF_CENTER);
    redefine(&mut files, &mut source, &revised).unwrap();
    assert_normal(&files, [-1.0, 0.0, 0.0]);
}
fn flip(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    Client::new(FileHost::new(files, source, "test"))
        .work_planes()
        .flip_normal(&FlipWorkPlaneArgs { index: 3 })
        .unwrap();
}
fn copy_and_open(files: &mut DiscoveryWorkspace, document: u64, source: &mut QueryGuids) {
    let path = Path::new("/fixtures/normal-axis-copy.obk");
    files
        .save_copy(document, path, None, "test", source)
        .unwrap();
    super::history::open_copied_datum(files, source, path);
}

fn edit_driver(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    slot: i64,
    reference: &str,
) {
    let args = RedefineWorkPlaneArgs {
        index: 3,
        repick: Some(vec![SlotRepick {
            slot,
            ref_: reference.into(),
        }]),
        ..Default::default()
    };
    redefine(files, source, &args).unwrap();
}
fn save_and_reopen(files: &mut DiscoveryWorkspace, document: u64, source: &mut QueryGuids) -> u64 {
    files.save(document, "test", source).unwrap();
    reopen_with_observer(files, document, source, Default::default())
}
