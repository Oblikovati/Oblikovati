// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{offset::redefine, planes::plane_rows};
fn request(first: &str, second: &str) -> CreateWorkPlaneArgs {
    CreateWorkPlaneArgs {
        kind: WORK_PLANE_TWO_LINES.0.into_owned(),
        refs: Some(vec![first.into(), second.into()]),
        ..Default::default()
    }
}
fn create(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    first: &str,
    second: &str,
) -> CreateWorkPlaneResult {
    Client::new(FileHost::new(files, source, "test"))
        .work_planes()
        .create(&request(first, second))
        .unwrap()
}
fn repick(first: &str, second: &str) -> RedefineWorkPlaneArgs {
    RedefineWorkPlaneArgs {
        index: 3,
        repick: Some(vec![
            SlotRepick {
                slot: 0,
                ref_: first.into(),
            },
            SlotRepick {
                slot: 1,
                ref_: second.into(),
            },
        ]),
        ..Default::default()
    }
}
fn normal(files: &DiscoveryWorkspace) -> Option<Vec<f64>> {
    plane_rows(files, true)[3].normal.clone()
}
#[test]
fn creation_lists_ordered_axis_slots_and_no_scalars() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = create(&mut files, &mut source, WORK_REF_X_AXIS, WORK_REF_Y_AXIS);
    assert!(created.healthy);
    let plane = plane_rows(&files, true).remove(3);
    assert_eq!(plane.kind, WORK_PLANE_TWO_LINES.0);
    assert_eq!(plane.origin, Some(vec![0.0; 3]));
    assert_eq!(normal(&files), Some(vec![0.0, 0.0, 1.0]));
    let slots = plane.slots.unwrap();
    let expected = [(0, "Line 1", "axis"), (1, "Line 2", "axis")];
    assert_eq!(
        slots
            .iter()
            .map(|slot| (slot.index, slot.label.as_str(), slot.kind.as_str()))
            .collect::<Vec<_>>(),
        expected
    );
    assert!(plane.scalars.is_none());
}
#[test]
fn user_axis_origin_places_the_plane_and_skew_second_origin_does_not() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let line = create_line(&mut files, &mut source);
    assert!(create(&mut files, &mut source, &line.ref_, WORK_REF_X_AXIS).healthy);
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![2.0, 3.0, 4.0])
    );
    assert_eq!(normal(&files), Some(vec![0.0, 1.0, 0.0]));
    redefine(
        &mut files,
        &mut source,
        &repick(WORK_REF_X_AXIS, &line.ref_),
    )
    .unwrap();
    assert_eq!(plane_rows(&files, true)[3].origin, Some(vec![0.0; 3]));
    assert_eq!(normal(&files), Some(vec![0.0, -1.0, 0.0]));
}
#[test]
fn parallel_creation_has_no_current_geometry_and_recovers_by_repick() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = create(&mut files, &mut source, WORK_REF_X_AXIS, WORK_REF_X_AXIS);
    assert!(!created.healthy);
    assert_eq!(normal(&files), None);
    assert_eq!(plane_rows(&files, true)[3].origin, None);
    let recovered = redefine(
        &mut files,
        &mut source,
        &repick(WORK_REF_X_AXIS, WORK_REF_Y_AXIS),
    )
    .unwrap();
    assert_eq!(recovered.plane.ref_, created.ref_);
    assert!(recovered.plane.healthy);
    assert_eq!(normal(&files), Some(vec![0.0, 0.0, 1.0]));
}
#[test]
fn ordered_repicks_and_display_share_one_undo_step_with_permanent_identity() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = create(&mut files, &mut source, WORK_REF_X_AXIS, WORK_REF_Y_AXIS);
    let mut args = repick(WORK_REF_Y_AXIS, WORK_REF_X_AXIS);
    args.grounded = Some(std::boxed::Box::new(true));
    let changed = redefine(&mut files, &mut source, &args).unwrap();
    assert_eq!(changed.plane.ref_, created.ref_);
    assert!(changed.plane.grounded);
    assert_eq!(normal(&files), Some(vec![0.0, 0.0, -1.0]));
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(normal(&files), Some(vec![0.0, 0.0, 1.0]));
    assert!(!plane_rows(&files, true)[3].grounded);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(normal(&files), Some(vec![0.0, 0.0, -1.0]));
}
#[test]
fn missing_malformed_and_wrong_kind_inputs_do_not_allocate_a_datum() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let before = owner_revision(&files, document);
    let mut missing = request(WORK_REF_X_AXIS, WORK_REF_Y_AXIS);
    missing.refs = None;
    for args in [
        missing,
        request(WORK_REF_CENTER, WORK_REF_Y_AXIS),
        request(WORK_REF_X_AXIS, WORK_REF_XY_PLANE),
        request("bad", WORK_REF_Y_AXIS),
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
fn invalid_second_repick_duplicate_slots_and_scalar_edits_are_atomic() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut source, WORK_REF_X_AXIS, WORK_REF_Y_AXIS);
    let before = owner_revision(&files, document);
    for mut args in invalid_revisions() {
        args.grounded = Some(std::boxed::Box::new(true));
        assert!(redefine(&mut files, &mut source, &args).is_err());
        assert_eq!(owner_revision(&files, document), before);
        assert_eq!(normal(&files), Some(vec![0.0, 0.0, 1.0]));
        assert!(!plane_rows(&files, true)[3].grounded);
    }
}
fn invalid_revisions() -> Vec<RedefineWorkPlaneArgs> {
    let mut duplicate = repick(WORK_REF_Y_AXIS, WORK_REF_Z_AXIS);
    duplicate.repick.as_mut().unwrap()[1].slot = 0;
    let mut scalar = repick(WORK_REF_Y_AXIS, WORK_REF_Z_AXIS);
    scalar.scalars = Some(vec![ScalarEdit {
        index: 0,
        value: "1 rad".into(),
    }]);
    let mut invalid_slot = repick(WORK_REF_Y_AXIS, WORK_REF_Z_AXIS);
    invalid_slot.repick.as_mut().unwrap()[1].slot = 2;
    vec![
        repick(WORK_REF_Y_AXIS, "bad"),
        repick(WORK_REF_Y_AXIS, WORK_REF_CENTER),
        duplicate,
        scalar,
        invalid_slot,
    ]
}
#[test]
fn forward_dependencies_decline_and_unchanged_repicks_are_noops() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut source, WORK_REF_X_AXIS, WORK_REF_Y_AXIS);
    let later = create_line(&mut files, &mut source);
    let before = owner_revision(&files, document);
    let forward = repick(WORK_REF_Y_AXIS, &later.ref_);
    assert!(redefine(&mut files, &mut source, &forward).is_err());
    let unchanged = repick(WORK_REF_X_AXIS, WORK_REF_Y_AXIS);
    redefine(&mut files, &mut source, &unchanged).unwrap();
    assert_eq!(owner_revision(&files, document), before);
}
#[test]
fn single_slot_edit_retains_the_other_axis() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut source, WORK_REF_X_AXIS, WORK_REF_Y_AXIS);
    let args = RedefineWorkPlaneArgs {
        index: 3,
        repick: Some(vec![SlotRepick {
            slot: 1,
            ref_: WORK_REF_Z_AXIS.into(),
        }]),
        ..Default::default()
    };
    redefine(&mut files, &mut source, &args).unwrap();
    assert_eq!(normal(&files), Some(vec![0.0, -1.0, 0.0]));
}
#[test]
fn save_reopen_and_copy_preserve_ordered_sources_and_authored_flip() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = create(&mut files, &mut source, WORK_REF_X_AXIS, WORK_REF_Y_AXIS);
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .work_planes()
        .flip_normal(&FlipWorkPlaneArgs { index: 3 })
        .unwrap();
    files.save(document, "test", &mut source).unwrap();
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(normal(&files), Some(vec![0.0, 0.0, -1.0]));
    let path = Path::new("/fixtures/two-axis-copy.obk");
    files
        .save_copy(document, path, None, "test", &mut source)
        .unwrap();
    super::history::open_copied_datum(&mut files, &mut source, path);
    assert_eq!(plane_rows(&files, true)[3].ref_, created.ref_);
    let args = repick(WORK_REF_Y_AXIS, WORK_REF_X_AXIS);
    redefine(&mut files, &mut source, &args).unwrap();
    assert_eq!(normal(&files), Some(vec![0.0, 0.0, 1.0]));
}
