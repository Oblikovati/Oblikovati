// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    document_sketch::PlacedSketchHealth, parameter::ParameterCollection, sketch::SketchPlane,
    units::UnitsOfMeasure,
};
use oblikovati_solve::nonlinear::SolveOptions;
struct RecordedProjectionInputs {
    points: Vec<SketchPoint2D>,
}
impl SketchProjectionInputs for RecordedProjectionInputs {
    fn points(&self, _: &PlacedSketch) -> Result<Vec<SketchPoint2D>, SketchError> {
        Ok(self.points.clone())
    }
}
fn placed() -> PlacedSketch {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0.0; 2]).unwrap();
    let mut placed = PlacedSketch::new(
        DocumentSketchId(1),
        "Projected",
        super::super::SketchPlacement::Fixed(SketchPlane::xy()),
        geometry,
    );
    placed.point_projections = vec![SketchPointProjection {
        anchor: SketchEntityId(1),
        reference: WorkPointReference::Origin,
    }];
    placed
}
#[test]
fn injected_inputs_must_match_owned_bindings_and_finite_coordinate_shape() {
    let placed = placed();
    for points in [
        Vec::new(),
        vec![SketchPoint2D {
            id: SketchEntityId(2),
            position: [0.0; 2],
        }],
        vec![SketchPoint2D {
            id: SketchEntityId(1),
            position: [f64::NAN; 2],
        }],
    ] {
        let inputs = RecordedProjectionInputs { points };
        let error = placed.projection_inputs(&inputs).unwrap_err();
        let message = error.to_string();
        assert!(message.contains("expected finite points in binding order"));
    }
}
#[test]
fn signed_zero_changes_in_reference_inputs_invalidate_verification() {
    let mut placed = placed();
    let parameters = ParameterCollection::new(UnitsOfMeasure::default());
    let projections = origin_input(-0.0);
    record_projection(&mut placed, &parameters, &projections);
    assert!(matches!(
        placed.health_with_projection_inputs(&parameters, &projections),
        PlacedSketchHealth::Current(_)
    ));
    let changed = origin_input(0.0);
    let health = placed.health_with_projection_inputs(&parameters, &changed);
    assert!(matches!(
        health,
        PlacedSketchHealth::ProjectionInputsChanged
    ));
}
fn record_projection(
    placed: &mut PlacedSketch,
    parameters: &ParameterCollection<u64>,
    projections: &Result<Vec<SketchPoint2D>, SketchError>,
) {
    let result = placed.geometry.solve_with_grounded_points(
        parameters,
        projections.as_ref().unwrap(),
        SolveOptions::default(),
        || false,
    );
    placed.record_evaluation(parameters, projections.clone(), &result);
}
fn origin_input(x: f64) -> Result<Vec<SketchPoint2D>, SketchError> {
    Ok(vec![SketchPoint2D {
        id: SketchEntityId(1),
        position: [x, 0.0],
    }])
}
