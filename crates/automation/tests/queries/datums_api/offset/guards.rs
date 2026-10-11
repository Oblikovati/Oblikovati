// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn invalid_plane_indices_and_scalar_slots_leave_the_owner_unchanged() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_offset(&mut files, &mut source, "10 mm");
    let before = owner_revision(&files, document);
    for index in [-1, 0, 1, 2, 4, i64::MAX] {
        let args = revision(index, "5 mm", WORK_REF_XY_PLANE);
        assert!(redefine(&mut files, &mut source, &args).is_err());
    }
    for slot in [-1, 1, i64::MAX] {
        let mut args = revision(3, "5 mm", WORK_REF_XY_PLANE);
        args.scalars.as_mut().unwrap()[0].index = slot;
        assert!(redefine(&mut files, &mut source, &args).is_err());
        args.scalars.as_mut().unwrap()[0].index = 0;
        args.repick.as_mut().unwrap()[0].slot = slot;
        assert!(redefine(&mut files, &mut source, &args).is_err());
    }
    assert_eq!(owner_revision(&files, document), before);
}

#[test]
fn duplicate_slots_and_malformed_display_fields_are_explicit_refusals() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_offset(&mut files, &mut source, "10 mm");
    let before = owner_revision(&files, document);
    for args in unsupported_edits() {
        assert!(redefine(&mut files, &mut source, &args).is_err());
    }
    assert_eq!(owner_revision(&files, document), before);
}
fn unsupported_edits() -> Vec<RedefineWorkPlaneArgs> {
    let original = revision(3, "5 mm", WORK_REF_XY_PLANE);
    let mut scalar = original.clone();
    let edit = scalar.scalars.as_ref().unwrap()[0].clone();
    scalar.scalars.as_mut().unwrap().push(edit);
    let mut reference = original.clone();
    let repick = reference.repick.as_ref().unwrap()[0].clone();
    reference.repick.as_mut().unwrap().push(repick);
    let mut edits = malformed_rectangles(&original);
    edits.extend([scalar, reference]);
    edits
}
fn malformed_rectangles(original: &RedefineWorkPlaneArgs) -> Vec<RedefineWorkPlaneArgs> {
    [
        vec![Some(vec![0.0; 3])],
        vec![None, None],
        vec![Some(vec![0.0; 2]), Some(vec![0.0; 3])],
    ]
    .into_iter()
    .map(|size| RedefineWorkPlaneArgs {
        size: Some(size),
        ..original.clone()
    })
    .collect()
}

#[test]
fn a_forward_repick_cannot_adopt_an_earlier_scalar_edit() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_offset(&mut files, &mut source, "width / 2");
    let later = create_offset(&mut files, &mut source, "5 mm");
    let before = owner_revision(&files, document);
    assert!(redefine(&mut files, &mut source, &revision(3, "8 cm", &later.ref_)).is_err());
    assert_source(&files, document, scalar_id(&files, document), "width / 2");
    assert_eq!(owner_revision(&files, document), before);
}

#[test]
fn repick_only_and_empty_edits_keep_the_owned_scalar() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_offset(&mut files, &mut source, "width / 2");
    let id = scalar_id(&files, document);
    let mut args = revision(3, "unused", WORK_REF_YZ_PLANE);
    args.scalars = None;
    let reply = redefine(&mut files, &mut source, &args).unwrap();
    assert_eq!(reply.plane.origin, Some(vec![0.5, 0.0, 0.0]));
    assert_eq!(scalar_id(&files, document), id);
    assert_source(&files, document, id, "width / 2");
    args.repick = None;
    let before = owner_revision(&files, document);
    redefine(&mut files, &mut source, &args).unwrap();
    assert_eq!(owner_revision(&files, document), before);
}

#[test]
fn fixed_planes_refuse_offset_scalar_edits_without_mutation() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = super::super::planes::fixed(&mut files, &mut source, false, true);
    let before = owner_revision(&files, document);
    let args = revision(created.index, "5 mm", WORK_REF_XY_PLANE);
    assert_datum_decline(
        redefine(&mut files, &mut source, &args),
        "offset constructor",
    );
    assert_eq!(owner_revision(&files, document), before);
}
