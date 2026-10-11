// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn save_reopen_copy_and_normal_flip_keep_formula_identity_and_driver_dependencies() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    let created = create(&mut files, &mut guids, "width / 1 cm * 90 deg");
    let id = scalar_id(&files, document);
    flip(&mut files, &mut guids);
    files.save(document, "test", &mut guids).unwrap();
    let document = reopen_with_observer(&mut files, document, &mut guids, Default::default());
    assert_normal(&files, [0.0, 1.0, 0.0]);
    let copy = Path::new("/fixtures/line-angle-copy.obk");
    files
        .save_copy(document, copy, None, "test", &mut guids)
        .unwrap();
    let copied = super::super::history::open_copied_datum(&mut files, &mut guids, copy);
    assert_eq!(plane_rows(&files, true)[3].ref_, created.ref_);
    assert_source(&files, copied, id, "width / 1 cm * 90 deg");
    edit_driver(&mut files, copied, "2 cm");
    assert_normal(&files, [0.0, 0.0, 1.0]);
}
fn flip(files: &mut DiscoveryWorkspace, guids: &mut QueryGuids) {
    Client::new(FileHost::new(files, guids, "test"))
        .work_planes()
        .flip_normal(&FlipWorkPlaneArgs { index: 3 })
        .unwrap();
}
#[test]
fn aborted_creation_retires_both_ids_and_removes_the_owned_angle() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    files
        .commands(document)
        .unwrap()
        .begin("Angle group")
        .unwrap();
    let retired = create(&mut files, &mut guids, "90 deg");
    let retired_scalar = scalar_id(&files, document);
    files.commands(document).unwrap().abort().unwrap();
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    assert!(snapshot.parameters.get(retired_scalar).is_none());
    let kept = create(&mut files, &mut guids, "90 deg");
    assert_ne!(kept.ref_, retired.ref_);
    assert_ne!(scalar_id(&files, document), retired_scalar);
}
#[test]
fn abort_and_commit_veto_keep_the_last_formula_reference_and_display_state() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut guids, "90 deg");
    files
        .commands(document)
        .unwrap()
        .begin("Angle edits")
        .unwrap();
    redefine(
        &mut files,
        &mut guids,
        &revision("0 rad", WORK_REF_Y_AXIS, WORK_REF_YZ_PLANE),
    )
    .unwrap();
    files.commands(document).unwrap().abort().unwrap();
    assert_normal(&files, [0.0, -1.0, 0.0]);
}
#[test]
fn commit_veto_keeps_the_last_formula_and_reference() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut guids, "90 deg");
    files.save(document, "test", &mut guids).unwrap();
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut guids, observer);
    let before = owner_revision(&files, document);
    let args = revision("0 rad", WORK_REF_Y_AXIS, WORK_REF_YZ_PLANE);
    assert!(redefine(&mut files, &mut guids, &args).is_err());
    assert_eq!(owner_revision(&files, document).1, before.1);
    assert_source(&files, document, scalar_id(&files, document), "90 deg");
}
#[test]
fn first_scalar_edit_promotes_a_legacy_driver_without_changing_the_user_parameter() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    let driver = insert_legacy_angle(&mut files, document);
    let args = revision("90 deg", WORK_REF_X_AXIS, WORK_REF_XY_PLANE);
    redefine(&mut files, &mut guids, &args).unwrap();
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    assert_eq!(snapshot.parameters.get(driver).unwrap().source, "45 deg");
    assert_eq!(snapshot.parameters.get(driver).unwrap().kind, USER_PARAM);
    assert_ne!(scalar_id(&files, document), driver);
    assert_normal(&files, [0.0, -1.0, 0.0]);
}
fn insert_legacy_angle(files: &mut DiscoveryWorkspace, document: u64) -> u64 {
    files
        .commands(document)
        .unwrap()
        .run("Legacy angle", legacy_angle_fixture)
        .unwrap()
}
fn legacy_angle_fixture(
    snapshot: &mut oblikovati_model::history::DocumentSnapshot,
    ids: &mut oblikovati_model::history::RecipeReservations<'_>,
) -> Result<u64, String> {
    use oblikovati_model::feature::AuthoredFeature;
    let parameter = ids.allocate().map_err(|error| error.to_string())?;
    snapshot
        .parameters
        .insert_expression(parameter, "tilt", USER_PARAM, "45 deg")
        .map_err(|error| error.to_string())?;
    let owner = ids.allocate_feature().map_err(|error| error.to_string())?;
    let definition = legacy_angle_definition(
        oblikovati_model::feature::work_geometry::DatumAngle::parameter(parameter),
    );
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(owner, "Legacy", definition))
        .map_err(|error| error.to_string())?;
    Ok(parameter)
}
fn legacy_angle_definition(
    angle: oblikovati_model::feature::work_geometry::DatumAngle,
) -> WorkGeometryDefinition {
    use oblikovati_model::feature::{
        work_axis::WorkAxisReference,
        work_geometry::WorkPlaneLineAngle,
        work_plane::{WorkPlaneOrigin, WorkPlaneReference},
    };
    WorkGeometryDefinition::AtAngleAboutAxis(WorkPlaneLineAngle {
        line: WorkAxisReference::OriginX,
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        angle,
    })
}

#[test]
fn a_legacy_constant_acquires_one_owned_expression_on_first_edit() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    files
        .commands(document)
        .unwrap()
        .run("Legacy constant", legacy_constant_fixture)
        .unwrap();
    let args = revision("90 deg", WORK_REF_X_AXIS, WORK_REF_XY_PLANE);
    redefine(&mut files, &mut guids, &args).unwrap();
    assert_source(&files, document, scalar_id(&files, document), "90 deg");
    assert_normal(&files, [0.0, -1.0, 0.0]);
}
fn legacy_constant_fixture(
    snapshot: &mut oblikovati_model::history::DocumentSnapshot,
    ids: &mut oblikovati_model::history::RecipeReservations<'_>,
) -> Result<(), String> {
    use oblikovati_model::feature::{AuthoredFeature, work_geometry::DatumAngle};
    let definition = legacy_angle_definition(DatumAngle::constant(0.0).unwrap());
    let owner = ids.allocate_feature().map_err(|error| error.to_string())?;
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(owner, "Legacy", definition))
        .map(|_| ())
        .map_err(|error| error.to_string())
}
