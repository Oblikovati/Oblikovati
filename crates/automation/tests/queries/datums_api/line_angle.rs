// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{
    offset::{edit_driver, redefine, scalar_id},
    planes::plane_rows,
};
use oblikovati_model::quantity::Dimension;
use std::f64::consts::FRAC_PI_2;
#[path = "line_angle/guards.rs"]
mod guards;
#[path = "line_angle/history.rs"]
mod history;
fn request(expression: &str, line: &str, base: &str) -> CreateWorkPlaneArgs {
    CreateWorkPlaneArgs {
        kind: WORK_PLANE_LINE_PLANE_ANGLE.0.into_owned(),
        angle: expression.into(),
        refs: Some(vec![line.into(), base.into()]),
        ..Default::default()
    }
}
fn create(
    files: &mut DiscoveryWorkspace,
    guids: &mut QueryGuids,
    expression: &str,
) -> CreateWorkPlaneResult {
    Client::new(FileHost::new(files, guids, "test"))
        .work_planes()
        .create(&request(expression, WORK_REF_X_AXIS, WORK_REF_XY_PLANE))
        .unwrap()
}
fn revision(expression: &str, line: &str, base: &str) -> RedefineWorkPlaneArgs {
    RedefineWorkPlaneArgs {
        index: 3,
        scalars: Some(vec![ScalarEdit {
            index: 0,
            value: expression.into(),
        }]),
        repick: Some(vec![
            SlotRepick {
                slot: 0,
                ref_: line.into(),
            },
            SlotRepick {
                slot: 1,
                ref_: base.into(),
            },
        ]),
        ..Default::default()
    }
}
fn assert_normal(files: &DiscoveryWorkspace, expected: [f64; 3]) {
    let actual = plane_rows(files, true)[3].normal.clone().unwrap();
    let bound = oblikovati_model::sketch::SketchPlaneOptions::default().angular_deviation;
    assert!(
        actual
            .into_iter()
            .zip(expected)
            .all(|(a, b)| (a - b).abs() <= bound)
    );
}
fn assert_source(files: &DiscoveryWorkspace, document: u64, id: u64, expression: &str) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let parameter = snapshot.parameters.get(id).unwrap();
    assert_eq!(parameter.source, expression);
    assert_eq!(parameter.kind, MODEL_PARAM);
    assert_eq!(
        snapshot.work_geometry.features()[0].owned_scalars()[0].role,
        "angle"
    );
}
fn angular_units(files: &mut DiscoveryWorkspace, document: u64, unit: &str) {
    files
        .commands(document)
        .unwrap()
        .run("Angular units", |snapshot, _| {
            let mut units = snapshot.parameters.units().clone();
            units.set_preferred(Dimension::Angle, unit).unwrap();
            snapshot.parameters.set_unit_preferences(units)
        })
        .unwrap();
}
#[test]
fn created_plane_keeps_a_live_formula_and_owned_angular_parameter() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    let expression = "width / 1 cm * 90 deg";
    assert!(create(&mut files, &mut guids, expression).healthy);
    let id = scalar_id(&files, document);
    assert_source(&files, document, id, expression);
    assert_normal(&files, [0.0, -1.0, 0.0]);
    edit_driver(&mut files, document, "2 cm");
    assert_normal(&files, [0.0, 0.0, -1.0]);
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    assert_eq!(snapshot.parameters.drivers(id).unwrap(), [1]);
}
#[test]
fn listing_uses_go_reference_labels_and_current_preferred_angular_units() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    angular_units(&mut files, document, "deg");
    create(&mut files, &mut guids, "90 deg");
    let plane = plane_rows(&files, true).remove(3);
    assert_eq!(plane.kind, WORK_PLANE_LINE_PLANE_ANGLE.0);
    assert_slots(&plane);
    let scalar = &plane.scalars.unwrap()[0];
    assert_eq!(
        (scalar.label.as_str(), scalar.unit.as_str(), scalar.value),
        ("Angle", "deg", 90.0)
    );
    angular_units(&mut files, document, "rad");
    assert_eq!(
        plane_rows(&files, true)[3].scalars.as_ref().unwrap()[0].value,
        FRAC_PI_2
    );
}
#[test]
fn repicks_and_formula_share_one_history_step_and_keep_scalar_identity() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    let created = create(&mut files, &mut guids, "90 deg");
    let id = scalar_id(&files, document);
    let mut args = revision("0 rad", WORK_REF_Y_AXIS, WORK_REF_YZ_PLANE);
    args.grounded = Some(std::boxed::Box::new(true));
    let reply = redefine(&mut files, &mut guids, &args).unwrap();
    assert_eq!(reply.plane.ref_, created.ref_);
    assert!(reply.plane.grounded);
    assert_eq!(scalar_id(&files, document), id);
    assert_source(&files, document, id, "0 rad");
    assert_normal(&files, [1.0, 0.0, 0.0]);
    files.commands(document).unwrap().undo().unwrap();
    assert_normal(&files, [0.0, -1.0, 0.0]);
    assert!(!plane_rows(&files, true)[3].grounded);
    files.commands(document).unwrap().redo().unwrap();
    assert_normal(&files, [1.0, 0.0, 0.0]);
}
#[test]
fn original_bare_angle_context_survives_unit_changes_and_reopen() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    angular_units(&mut files, document, "deg");
    create(&mut files, &mut guids, "90");
    let id = scalar_id(&files, document);
    angular_units(&mut files, document, "rad");
    files.save(document, "test", &mut guids).unwrap();
    let document = reopen_with_observer(&mut files, document, &mut guids, Default::default());
    assert_source(&files, document, id, "90");
    assert_normal(&files, [0.0, -1.0, 0.0]);
    let args = revision("0.25", WORK_REF_X_AXIS, WORK_REF_XY_PLANE);
    redefine(&mut files, &mut guids, &args).unwrap();
    assert_normal(&files, [0.0, -0.25_f64.sin(), 0.25_f64.cos()]);
}
#[test]
fn failed_formula_drivers_have_no_stale_frame_and_recover() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut guids, "width / 1 cm * 90 deg");
    edit_driver(&mut files, document, "missing");
    let plane = plane_rows(&files, true).remove(3);
    assert!(!plane.healthy && plane.origin.is_none() && plane.normal.is_none());
    assert!(plane.scalars.is_none());
    edit_driver(&mut files, document, "1 cm");
    assert_normal(&files, [0.0, -1.0, 0.0]);
}
#[test]
fn repick_only_preserves_formula_and_identical_commands_do_not_create_generations() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    create(&mut files, &mut guids, "90 deg");
    let mut args = revision("90 deg", WORK_REF_X_AXIS, WORK_REF_XY_PLANE);
    let before = owner_revision(&files, document);
    redefine(&mut files, &mut guids, &args).unwrap();
    assert_eq!(owner_revision(&files, document), before);
    args.scalars = None;
    args.repick.as_mut().unwrap().remove(1);
    args.repick.as_mut().unwrap()[0].ref_ = WORK_REF_Y_AXIS.into();
    redefine(&mut files, &mut guids, &args).unwrap();
    assert_source(&files, document, scalar_id(&files, document), "90 deg");
    assert_normal(&files, [1.0, 0.0, 0.0]);
}

fn assert_slots(plane: &WorkPlaneInfo) {
    let slots = plane.slots.clone().unwrap();
    let expected = [(0, "Line", "axis"), (1, "Plane", "plane")];
    assert_eq!(
        slots
            .iter()
            .map(|slot| (slot.index, slot.label.as_str(), slot.kind.as_str()))
            .collect::<Vec<_>>(),
        expected
    );
}

#[test]
fn identical_bare_text_in_new_units_is_a_physical_edit() {
    let (mut files, document, mut guids) = configured_files(DOCUMENT_PART, Default::default());
    angular_units(&mut files, document, "deg");
    create(&mut files, &mut guids, "90");
    angular_units(&mut files, document, "rad");
    let before = owner_revision(&files, document);
    let args = revision("90", WORK_REF_X_AXIS, WORK_REF_XY_PLANE);
    redefine(&mut files, &mut guids, &args).unwrap();
    assert!(owner_revision(&files, document).0 > before.0);
    assert_normal(&files, [0.0, -90_f64.sin(), 90_f64.cos()]);
}
