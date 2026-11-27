// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    parameter::ParameterCollection,
    sketch::{DistanceMeasure, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId},
};

fn fixture() -> (Sketch2D, ParameterCollection<u64>) {
    let mut sketch = Sketch2D::default();
    let [origin, endpoint] = [SketchEntityId(1), SketchEntityId(2)];
    sketch.add_point(origin, [0., 0.]).unwrap();
    sketch.add_point(endpoint, [2., 0.]).unwrap();
    for (id, relation) in [
        (
            1,
            PlanarConstraint::Fix {
                point: origin,
                position: [0., 0.],
            },
        ),
        (
            2,
            PlanarConstraint::Distance {
                points: [origin, endpoint],
                measure: DistanceMeasure::Aligned,
                parameter: 1,
            },
        ),
        (
            3,
            PlanarConstraint::PointAngle {
                points: [origin, endpoint],
                parameter: 2,
            },
        ),
    ] {
        sketch
            .add_constraint(SketchConstraintId(id), relation)
            .unwrap();
    }
    let mut parameters = ParameterCollection::new(Default::default());
    parameters
        .insert_expression(1, "radius", USER_PARAM, "2 cm")
        .unwrap();
    parameters
        .insert_expression(2, "direction", USER_PARAM, "0 deg")
        .unwrap();
    (sketch, parameters)
}
#[test]
fn point_direction_drives_both_quadrants_without_changing_length() {
    let (mut sketch, mut parameters) = fixture();
    for angle in [0., 45., 120., 270., 359.] {
        parameters
            .set_expression(2, &format!("{angle} deg"))
            .unwrap();
        let result = sketch
            .solve(&parameters, Default::default(), || false)
            .unwrap();
        assert_eq!(
            result.numerical().termination,
            oblikovati_solve::nonlinear::SolveTermination::Converged
        );
        assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
        sketch = result.into_candidate();
        let point = sketch.point(SketchEntityId(2)).unwrap().position;
        let radians = f64::to_radians(angle);
        assert!(
            (point[0] - 2. * radians.cos()).abs() < 1e-7,
            "{angle}: {point:?}"
        );
        assert!(
            (point[1] - 2. * radians.sin()).abs() < 1e-7,
            "{angle}: {point:?}"
        );
        let measured = sketch
            .measure_dimension(SketchConstraintId(3))
            .unwrap()
            .value;
        assert!((measured - radians).abs() < 1e-7, "{angle}: {measured}");
    }
}
#[test]
fn point_direction_schema_round_trips_and_rejects_old_versions() {
    let (sketch, _) = fixture();
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 13);
    let bytes = serde_json::to_vec(&recipe).unwrap();
    let decoded = serde_json::from_slice(&bytes).unwrap();
    assert!(sketch.same_recipe(&Sketch2D::from_recipe(&decoded).unwrap()));
    let mut old = recipe;
    old.version = 12;
    assert!(
        Sketch2D::from_recipe(&old)
            .unwrap_err()
            .to_string()
            .contains("point angle recipe version 12")
    );
}
