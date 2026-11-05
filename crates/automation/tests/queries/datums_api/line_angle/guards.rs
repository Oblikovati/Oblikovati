// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn invalid_sources_and_reference_shapes_decline_before_datum_allocation() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    let before = owner_revision(&files, document);
    for args in invalid_creations() {
        assert!(
            Client::new(FileHost::new(&mut files, &mut guids, "test"))
                .work_planes()
                .create(&args)
                .is_err()
        );
        assert_eq!(owner_revision(&files, document), before);
    }
}
fn invalid_creations() -> Vec<CreateWorkPlaneArgs> {
    let mut missing = request("90 deg", WORK_REF_X_AXIS, WORK_REF_XY_PLANE);
    missing.refs = None;
    let mut requests = vec![
        missing,
        request("90 deg", WORK_REF_CENTER, WORK_REF_XY_PLANE),
        request("90 deg", WORK_REF_X_AXIS, WORK_REF_CENTER),
    ];
    requests.extend(
        ["", "1 cm", "missing", "1 / 0 rad"]
            .map(|source| request(source, WORK_REF_X_AXIS, WORK_REF_XY_PLANE)),
    );
    requests
}
#[test]
fn bad_second_repick_cannot_publish_an_earlier_formula_or_display_edit() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut guids, "90 deg");
    let before = owner_revision(&files, document);
    for mut args in invalid_edits() {
        args.grounded = Some(std::boxed::Box::new(true));
        assert!(redefine(&mut files, &mut guids, &args).is_err());
        assert_eq!(owner_revision(&files, document), before);
        assert_source(&files, document, scalar_id(&files, document), "90 deg");
        assert_normal(&files, [0.0, -1.0, 0.0]);
    }
}
fn invalid_edits() -> Vec<RedefineWorkPlaneArgs> {
    let mut duplicate = revision("0 rad", WORK_REF_Y_AXIS, WORK_REF_YZ_PLANE);
    duplicate.repick.as_mut().unwrap()[1].slot = 0;
    let mut scalar = revision("0 rad", WORK_REF_Y_AXIS, WORK_REF_YZ_PLANE);
    scalar.scalars.as_mut().unwrap()[0].index = 1;
    let mut duplicates = scalar.clone();
    let edit = ScalarEdit {
        index: 0,
        value: "0 rad".into(),
    };
    duplicates.scalars = Some(vec![edit; 2]);
    vec![
        duplicate,
        scalar,
        duplicates,
        revision("0 rad", WORK_REF_Y_AXIS, "bad"),
        revision("1 cm", WORK_REF_Y_AXIS, WORK_REF_YZ_PLANE),
    ]
}
#[test]
fn forward_geometry_drivers_do_not_publish_an_earlier_scalar_edit() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut guids, "90 deg");
    let later = create_line(&mut files, &mut guids);
    let before = owner_revision(&files, document);
    let args = revision("0 rad", &later.ref_, WORK_REF_XY_PLANE);
    assert!(redefine(&mut files, &mut guids, &args).is_err());
    assert_eq!(owner_revision(&files, document), before);
    assert_normal(&files, [0.0, -1.0, 0.0]);
}
#[test]
fn undefined_geometry_is_unhealthy_and_a_repick_recovers_the_same_identity() {
    let (mut files, _, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    let args = request("90 deg", WORK_REF_X_AXIS, WORK_REF_YZ_PLANE);
    let created = Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .work_planes()
        .create(&args)
        .unwrap();
    assert!(!created.healthy);
    assert!(plane_rows(&files, true)[3].normal.is_none());
    let reply = redefine(
        &mut files,
        &mut guids,
        &revision("90 deg", WORK_REF_X_AXIS, WORK_REF_XY_PLANE),
    )
    .unwrap();
    assert!(reply.plane.healthy);
    assert_eq!(reply.plane.ref_, created.ref_);
    assert_normal(&files, [0.0, -1.0, 0.0]);
}
#[test]
fn user_axis_origin_places_the_plane_with_current_direction() {
    let (mut files, _, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    let line = create_line(&mut files, &mut guids);
    let args = request("90 deg", &line.ref_, WORK_REF_YZ_PLANE);
    let created = Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .work_planes()
        .create(&args)
        .unwrap();
    assert!(created.healthy);
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![2.0, 3.0, 4.0])
    );
    assert_normal(&files, [0.0, 1.0, 0.0]);
}

#[test]
fn a_self_referencing_owned_formula_cannot_publish_a_cyclic_scalar() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut guids, "90 deg");
    let id = scalar_id(&files, document);
    let name = files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .parameters
        .get(id)
        .unwrap()
        .name
        .clone();
    let before = owner_revision(&files, document);
    let args = revision(&name, WORK_REF_X_AXIS, WORK_REF_XY_PLANE);
    assert!(redefine(&mut files, &mut guids, &args).is_err());
    assert_eq!(owner_revision(&files, document), before);
    assert_source(&files, document, id, "90 deg");
}

#[test]
fn identical_offset_edits_share_the_unit_context_noop_check() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    super::super::offset::create_offset(&mut files, &mut guids, "width / 2");
    let before = owner_revision(&files, document);
    let args = super::super::offset::revision(3, "width / 2", WORK_REF_XY_PLANE);
    redefine(&mut files, &mut guids, &args).unwrap();
    assert_eq!(owner_revision(&files, document), before);
}

#[test]
fn angular_display_overflow_declines_before_owner_adoption() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    angular_units(&mut files, document, "deg");
    let before = owner_revision(&files, document);
    let args = request(
        "1.7976931348623157e308 rad",
        WORK_REF_X_AXIS,
        WORK_REF_XY_PLANE,
    );
    let outcome = Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .work_planes()
        .create(&args);
    assert_datum_decline(outcome, "finite scalar");
    assert_eq!(owner_revision(&files, document), before);
}
