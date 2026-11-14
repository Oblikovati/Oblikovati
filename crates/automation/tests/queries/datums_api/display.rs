// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{
    offset::{create_offset, edit_driver, redefine, revision, scalar_id},
    planes::{fixed, plane_rows},
};
#[path = "display/history.rs"]
mod history;

fn explicit_size() -> Vec<Option<Vec<f64>>> {
    vec![Some(vec![1.0, 2.0, 0.0]), Some(vec![3.0, 4.0, 0.0])]
}
fn display_args(index: i64) -> RedefineWorkPlaneArgs {
    RedefineWorkPlaneArgs {
        index,
        auto_resize: Some(std::boxed::Box::new(true)),
        grounded: Some(std::boxed::Box::new(true)),
        size: Some(explicit_size()),
        ..Default::default()
    }
}
fn assert_explicit(info: &WorkPlaneInfo) {
    assert!(!info.auto_resize && info.grounded);
    assert_eq!(info.size, Some(explicit_size()));
}

#[test]
fn fixed_display_edits_keep_frame_and_key_and_size_disables_auto_resize() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = fixed(&mut files, &mut source, false, true);
    let reply = redefine(&mut files, &mut source, &display_args(created.index)).unwrap();
    assert_explicit(&reply.plane);
    assert_eq!(reply.plane.ref_, created.ref_);
    assert_eq!(reply.plane.origin, Some(vec![0.0, 0.0, 5.0]));
    assert_eq!(reply.plane.normal, Some(vec![0.0, 0.0, 1.0]));
}

#[test]
fn mixed_scalar_reference_and_display_edit_is_one_atomic_command() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_offset(&mut files, &mut source, "width / 2");
    let scalar = scalar_id(&files, document);
    let mut args = revision(3, "width * 2", WORK_REF_YZ_PLANE);
    args.auto_resize = display_args(3).auto_resize;
    args.grounded = display_args(3).grounded;
    args.size = Some(explicit_size());
    let reply = redefine(&mut files, &mut source, &args).unwrap();
    assert_explicit(&reply.plane);
    assert_eq!(reply.plane.origin, Some(vec![2.0, 0.0, 0.0]));
    assert_eq!(scalar_id(&files, document), scalar);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![0.0, 0.0, 0.5])
    );
    assert!(!plane_rows(&files, true)[3].grounded);
}

#[test]
fn malformed_size_cannot_publish_earlier_scalar_and_flag_edits() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_offset(&mut files, &mut source, "width / 2");
    let before = owner_revision(&files, document);
    let mut args = revision(3, "width * 2", WORK_REF_YZ_PLANE);
    args.grounded = Some(std::boxed::Box::new(true));
    args.size = Some(vec![Some(vec![0.0; 2]), Some(vec![0.0; 3])]);
    assert!(redefine(&mut files, &mut source, &args).is_err());
    assert_eq!(owner_revision(&files, document), before);
    assert!(!plane_rows(&files, true)[3].grounded);
}

#[test]
fn defaults_report_current_rectangles_and_origin_extents() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    fixed(&mut files, &mut source, false, true);
    let rows = plane_rows(&files, true);
    assert_eq!(
        rows[0].size,
        Some(vec![Some(vec![-5.0, -5.0, 0.0]), Some(vec![5.0, 5.0, 0.0])])
    );
    assert_eq!(
        rows[3].size,
        Some(vec![Some(vec![-5.0, -5.0, 5.0]), Some(vec![5.0, 5.0, 5.0])])
    );
    assert!(!rows[3].grounded && !rows[3].auto_resize);
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .work_planes()
        .flip_normal(&FlipWorkPlaneArgs { index: 3 })
        .unwrap();
    let expected = vec![Some(vec![-5.0, 5.0, 5.0]), Some(vec![5.0, -5.0, 5.0])];
    assert_eq!(plane_rows(&files, true)[3].size, Some(expected));
}

#[test]
fn failed_geometry_has_no_derived_rectangle_but_keeps_explicit_source() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_offset(&mut files, &mut source, "width / 2");
    edit_driver(&mut files, document, "missing");
    assert!(plane_rows(&files, true)[3].size.is_none());
    let reply = redefine(&mut files, &mut source, &display_args(3)).unwrap();
    assert!(!reply.plane.healthy && reply.plane.origin.is_none());
    assert_explicit(&reply.plane);
    edit_driver(&mut files, document, "8 cm");
    assert_explicit(&plane_rows(&files, true)[3]);
}

#[test]
fn resetting_flags_and_empty_size_are_metadata_no_ops_when_already_default() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    fixed(&mut files, &mut source, false, true);
    let before = owner_revision(&files, document);
    let args = RedefineWorkPlaneArgs {
        index: 3,
        auto_resize: Some(std::boxed::Box::new(false)),
        grounded: Some(std::boxed::Box::new(false)),
        size: Some(vec![]),
        ..Default::default()
    };
    redefine(&mut files, &mut source, &args).unwrap();
    assert_eq!(owner_revision(&files, document), before);
}

#[test]
fn grounded_and_auto_resize_flags_do_not_freeze_a_live_offset() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_offset(&mut files, &mut source, "width / 2");
    let mut args = display_args(3);
    args.size = None;
    let reply = redefine(&mut files, &mut source, &args).unwrap();
    assert!(reply.plane.grounded && reply.plane.auto_resize);
    edit_driver(&mut files, document, "8 cm");
    let plane = &plane_rows(&files, true)[3];
    assert_eq!(plane.origin, Some(vec![0.0, 0.0, 4.0]));
    assert_live_rectangle(plane);
    files.save(document, "test", &mut source).unwrap();
    reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    let plane = &plane_rows(&files, true)[3];
    assert!(plane.grounded && plane.auto_resize);
    assert_eq!(plane.origin, Some(vec![0.0, 0.0, 4.0]));
}

fn assert_live_rectangle(plane: &WorkPlaneInfo) {
    assert_eq!(
        plane.size,
        Some(vec![Some(vec![-5.0, -5.0, 4.0]), Some(vec![5.0, 5.0, 4.0])])
    );
}
