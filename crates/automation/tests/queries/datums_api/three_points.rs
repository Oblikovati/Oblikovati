// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{offset::redefine, planes::plane_rows};

fn request(refs: &[&str]) -> CreateWorkPlaneArgs {
    CreateWorkPlaneArgs {
        kind: WORK_PLANE_THREE_POINTS.0.into_owned(),
        refs: Some(refs.iter().map(|reference| (*reference).into()).collect()),
        ..Default::default()
    }
}
fn drivers(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) -> [String; 3] {
    let second = create_point(files, source, [1.0, 0.0, 0.0], false);
    let third = create_point(files, source, [0.0, 1.0, 0.0], false);
    [WORK_REF_CENTER.into(), second.ref_, third.ref_]
}
fn create(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    refs: &[String; 3],
) -> CreateWorkPlaneResult {
    let borrowed = refs.each_ref().map(String::as_str);
    Client::new(FileHost::new(files, source, "test"))
        .work_planes()
        .create(&request(&borrowed))
        .unwrap()
}
fn repick(index: i64, replacements: &[(i64, &str)]) -> RedefineWorkPlaneArgs {
    RedefineWorkPlaneArgs {
        index,
        repick: Some(
            replacements
                .iter()
                .map(|(slot, reference)| SlotRepick {
                    slot: *slot,
                    ref_: (*reference).into(),
                })
                .collect(),
        ),
        ..Default::default()
    }
}
#[test]
fn typed_creation_lists_current_frame_and_ordered_reference_slots() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let refs = drivers(&mut files, &mut source);
    let created = create(&mut files, &mut source, &refs);
    assert!(created.healthy);
    let plane = plane_rows(&files, true).remove(3);
    assert_eq!(plane.kind, WORK_PLANE_THREE_POINTS.0);
    assert_eq!(plane.origin, Some(vec![0.0; 3]));
    assert_eq!(plane.normal, Some(vec![0.0, 0.0, 1.0]));
    assert_reference_slots(&plane);
    assert!(plane.scalars.is_none());
}
fn assert_reference_slots(plane: &WorkPlaneInfo) {
    let slots = plane.slots.as_ref().unwrap();
    assert_eq!(
        slots
            .iter()
            .map(|slot| (slot.index, slot.label.as_str(), slot.kind.as_str()))
            .collect::<Vec<_>>(),
        [
            (0, "Point 1", "point"),
            (1, "Point 2", "point"),
            (2, "Point 3", "point")
        ]
    );
}

#[test]
fn repicks_are_one_atomic_command_and_keep_plane_identity_through_undo() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let refs = drivers(&mut files, &mut source);
    let created = create(&mut files, &mut source, &refs);
    let mut args = repick(3, &[(1, &refs[2]), (2, &refs[1])]);
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
fn assert_normal(files: &DiscoveryWorkspace, normal: [f64; 3]) {
    assert_eq!(plane_rows(files, true)[3].normal, Some(normal.to_vec()));
}
#[test]
fn malformed_references_do_not_allocate_a_plane() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let before = owner_revision(&files, document);
    for refs in [
        vec![],
        vec![WORK_REF_CENTER],
        vec![WORK_REF_CENTER; 4],
        vec![WORK_REF_CENTER, WORK_REF_CENTER, WORK_REF_XY_PLANE],
        vec![WORK_REF_CENTER, WORK_REF_CENTER, "bad"],
    ] {
        assert!(
            Client::new(FileHost::new(&mut files, &mut source, "test"))
                .work_planes()
                .create(&request(&refs))
                .is_err()
        );
        assert_eq!(owner_revision(&files, document), before);
    }
}
#[test]
fn bad_final_repick_or_invalid_slots_cannot_publish_earlier_edits() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let refs = drivers(&mut files, &mut source);
    let created = create(&mut files, &mut source, &refs);
    let before = owner_revision(&files, document);
    for mut args in invalid_revisions(&refs, &created.ref_) {
        args.grounded = Some(std::boxed::Box::new(true));
        assert!(redefine(&mut files, &mut source, &args).is_err());
        assert_eq!(owner_revision(&files, document), before);
        assert_normal(&files, [0.0, 0.0, 1.0]);
        assert!(!plane_rows(&files, true)[3].grounded);
    }
}
fn invalid_revisions(refs: &[String; 3], plane: &str) -> Vec<RedefineWorkPlaneArgs> {
    let mut scalar = repick(3, &[]);
    scalar.scalars = Some(vec![ScalarEdit {
        index: 0,
        value: "1 cm".into(),
    }]);
    vec![
        repick(3, &[(1, &refs[2]), (2, "bad")]),
        repick(3, &[(1, &refs[2]), (1, &refs[1])]),
        repick(3, &[(-1, WORK_REF_CENTER)]),
        repick(3, &[(3, WORK_REF_CENTER)]),
        repick(3, &[(2, plane)]),
        scalar,
    ]
}
#[test]
fn degenerate_source_retains_health_then_a_valid_repick_recovers_current_geometry() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let refs = drivers(&mut files, &mut source);
    let repeated = [refs[0].clone(), refs[1].clone(), refs[1].clone()];
    assert!(!create(&mut files, &mut source, &repeated).healthy);
    let row = plane_rows(&files, true).remove(3);
    assert!(!row.healthy && row.normal.is_none() && row.origin.is_none());
    assert!(!row.reason.is_empty());
    redefine(&mut files, &mut source, &repick(3, &[(2, &refs[2])])).unwrap();
    assert_normal(&files, [0.0, 0.0, 1.0]);
}
#[test]
fn unchanged_repick_has_no_generation_and_group_abort_restores_source() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let refs = drivers(&mut files, &mut source);
    create(&mut files, &mut source, &refs);
    let before = owner_revision(&files, document);
    redefine(&mut files, &mut source, &repick(3, &[(2, &refs[2])])).unwrap();
    assert_eq!(owner_revision(&files, document), before);
    let mut commands = files.commands(document).unwrap();
    commands.begin("Three points").unwrap();
    redefine(
        &mut files,
        &mut source,
        &repick(3, &[(1, &refs[2]), (2, &refs[1])]),
    )
    .unwrap();
    files.commands(document).unwrap().abort().unwrap();
    assert_normal(&files, [0.0, 0.0, 1.0]);
}
#[test]
fn save_reopen_and_independent_copy_keep_sources_and_authored_flip() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let refs = drivers(&mut files, &mut source);
    let created = create(&mut files, &mut source, &refs);
    flip(&mut files, &mut source);
    files.save(document, "test", &mut source).unwrap();
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_normal(&files, [0.0, 0.0, -1.0]);
    copy_and_open(&mut files, document, &mut source);
    assert_normal(&files, [0.0, 0.0, -1.0]);
    assert_eq!(plane_rows(&files, true)[3].ref_, created.ref_);
    redefine(
        &mut files,
        &mut source,
        &repick(3, &[(1, &refs[2]), (2, &refs[1])]),
    )
    .unwrap();
    assert_normal(&files, [0.0, 0.0, 1.0]);
}

fn flip(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    Client::new(FileHost::new(files, source, "test"))
        .work_planes()
        .flip_normal(&FlipWorkPlaneArgs { index: 3 })
        .unwrap();
}

fn copy_and_open(files: &mut DiscoveryWorkspace, document: u64, source: &mut QueryGuids) {
    let path = Path::new("/fixtures/three-point-copy.obk");
    files
        .save_copy(document, path, None, "test", source)
        .unwrap();
    super::history::open_copied_datum(files, source, path);
}

#[test]
fn forward_reference_repick_cannot_publish_display_or_source_edits() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let refs = drivers(&mut files, &mut source);
    create(&mut files, &mut source, &refs);
    let later = create_point(&mut files, &mut source, [0.0, 0.0, 1.0], false);
    let before = owner_revision(&files, document);
    let mut args = repick(3, &[(2, &later.ref_)]);
    args.grounded = Some(std::boxed::Box::new(true));
    assert!(redefine(&mut files, &mut source, &args).is_err());
    assert_eq!(owner_revision(&files, document), before);
    assert_normal(&files, [0.0, 0.0, 1.0]);
    assert!(!plane_rows(&files, true)[3].grounded);
}

#[test]
fn commit_veto_restores_the_previous_three_point_source() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let refs = drivers(&mut files, &mut source);
    create(&mut files, &mut source, &refs);
    files.save(document, "test", &mut source).unwrap();
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let before = owner_revision(&files, document);
    let revision = repick(3, &[(1, &refs[2]), (2, &refs[1])]);
    assert!(redefine(&mut files, &mut source, &revision).is_err());
    assert_eq!(owner_revision(&files, document).1, before.1);
    assert_normal(&files, [0.0, 0.0, 1.0]);
}
