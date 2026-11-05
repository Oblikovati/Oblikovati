// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{parameter::ParameterCollection, units::UnitsOfMeasure};
use std::f64::consts::{FRAC_PI_3, FRAC_PI_4, FRAC_PI_6};
fn angular_snapshot(scale: f64) -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    let units = UnitsOfMeasure::default().with_working_scale(scale).unwrap();
    snapshot.parameters = ParameterCollection::new(units);
    snapshot
        .parameters
        .insert_expression(1, "angle", MODEL_PARAM, "90 deg")
        .unwrap();
    snapshot
        .sketches
        .add(
            DocumentSketchId(1),
            "Angle",
            SketchPlane::xy(),
            angle_geometry(),
        )
        .unwrap();
    snapshot
}
fn angle_geometry() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    for (id, position) in [(1, [0.0, 0.0]), (2, [1.0, 0.0]), (3, [0.0, 1.0])] {
        geometry.add_point(SketchEntityId(id), position).unwrap();
    }
    append_angle(&mut geometry);
    geometry
}
fn append_angle(geometry: &mut Sketch2D) {
    let relation = PlanarConstraint::ThreePointAngle {
        vertex: SketchEntityId(1),
        points: [SketchEntityId(2), SketchEntityId(3)],
        turn: AngleTurn::CounterClockwise,
        parameter: 1,
    };
    geometry
        .add_constraint(SketchConstraintId(1), relation)
        .unwrap();
}
#[test]
fn angular_display_drives_and_radian_limits_are_independent_of_length_scale() {
    for scale in [1e-6, 1.0, 1e6] {
        let mut snapshot = angular_snapshot(scale);
        drive(&mut snapshot, "45").unwrap();
        assert!((target(&snapshot) - FRAC_PI_4).abs() < 1e-14);
        set_limits(&mut snapshot, FRAC_PI_6, FRAC_PI_3);
        assert!((target(&snapshot) - FRAC_PI_4).abs() < 1e-14);
        drive(&mut snapshot, "75 deg").unwrap();
        assert_eq!(target(&snapshot), FRAC_PI_3);
        drive(&mut snapshot, "0 deg").unwrap();
        assert_eq!(target(&snapshot), FRAC_PI_6);
        drive(&mut snapshot, "1 cm").unwrap_err();
        assert_eq!(target(&snapshot), FRAC_PI_6);
    }
}
