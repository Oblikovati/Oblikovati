// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use num_rational::BigRational;
use oblikovati_model::{
    history::SketchFrameInputBounds,
    sketch::{DistanceMeasure, PlanarConstraint, SketchConstraintId},
};

fn source_snapshot() -> (
    DocumentSnapshot,
    oblikovati_model::sketch::SketchSolveOutcome,
) {
    let mut snapshot = snapshot();
    snapshot
        .sketches
        .set_point_projections(
            SKETCH,
            vec![SketchPointProjection {
                anchor: ANCHOR,
                reference: WorkPointReference::Origin,
            }],
        )
        .unwrap();
    let mut geometry = geometry();
    geometry.add_point(SketchEntityId(12), [4., 0.]).unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Distance {
                points: [ANCHOR, SketchEntityId(12)],
                measure: DistanceMeasure::Horizontal,
                parameter: 1,
            },
        )
        .unwrap();
    snapshot
        .sketches
        .replace_geometry(SKETCH, geometry)
        .unwrap();
    let batch = snapshot
        .solve_one_document_sketch(SKETCH, SolveOptions::default(), || false)
        .unwrap();
    let outcome = batch.report.reports[0].result.as_ref().unwrap().clone();
    snapshot.sketches = batch.candidate;
    (snapshot, outcome)
}
fn assumptions() -> SketchFrameInputBounds {
    SketchFrameInputBounds {
        origin: [[-0.1, 0.2], [-0.3, 0.4], [0., 0.]],
        x_axis: [[1., 1.], [0., 0.], [0., 0.]],
        y_axis: [[0., 0.], [1., 1.], [0., 0.]],
    }
}
fn q(value: f64) -> BigRational {
    BigRational::from_float(value).unwrap()
}
#[test]
fn origin_projection_contains_exact_frame_assumption_corners() {
    let (snapshot, outcome) = source_snapshot();
    let evidence = snapshot
        .conditional_origin_projection(SKETCH, ANCHOR, &outcome, assumptions())
        .unwrap();
    assert_eq!(evidence.sketch(), SKETCH);
    assert_eq!(evidence.anchor(), ANCHOR);
    assert_eq!(evidence.source(), WorkPointReference::Origin);
    assert_eq!(evidence.assumptions(), assumptions());
    let bounds = evidence.local_bounds();
    for origin in [-0.1, 0.2] {
        assert!(q(bounds[0][0]) <= -q(origin) && -q(origin) <= q(bounds[0][1]));
    }
    for origin in [-0.3, 0.4] {
        assert!(q(bounds[1][0]) <= -q(origin) && -q(origin) <= q(bounds[1][1]));
    }
    assert!(
        bounds
            .iter()
            .all(|interval| interval[0] <= 0. && 0. <= interval[1])
    );
}
#[test]
fn same_value_parameter_source_change_is_rejected() {
    let (mut snapshot, outcome) = source_snapshot();
    snapshot.parameters.set_expression(1, "40 mm").unwrap();
    assert_eq!(snapshot.parameters.model_value(1).unwrap().value, 4.);
    assert!(
        snapshot
            .conditional_origin_projection(SKETCH, ANCHOR, &outcome, assumptions())
            .is_err()
    );
}
#[test]
fn frame_edit_and_unproven_frame_boxes_are_rejected() {
    let (mut snapshot, outcome) = source_snapshot();
    let mut bad = assumptions();
    bad.x_axis[0] = [2., 2.];
    assert!(
        snapshot
            .conditional_origin_projection(SKETCH, ANCHOR, &outcome, bad)
            .is_err()
    );
    let rotated =
        SketchPlane::new([0.; 3], [0., 1., 0.], [-1., 0., 0.], Default::default()).unwrap();
    snapshot.sketches.replace_plane(SKETCH, rotated).unwrap();
    assert_eq!(
        snapshot.resolve_sketch_projected_points(SKETCH).unwrap()[0].position,
        [0., 0.]
    );
    assert!(
        snapshot
            .conditional_origin_projection(SKETCH, ANCHOR, &outcome, assumptions())
            .is_err()
    );
}
#[test]
fn nonorigin_projection_is_refused() {
    let mut snapshot = snapshot();
    let batch = snapshot
        .solve_one_document_sketch(SKETCH, SolveOptions::default(), || false)
        .unwrap();
    let outcome = batch.report.reports[0].result.as_ref().unwrap().clone();
    snapshot.sketches = batch.candidate;
    assert!(
        snapshot
            .conditional_origin_projection(SKETCH, ANCHOR, &outcome, assumptions())
            .is_err()
    );
}

#[test]
fn moved_frame_invalidates_the_stored_projected_point() {
    let (mut snapshot, outcome) = source_snapshot();
    let moved =
        SketchPlane::new([1., 0., 0.], [1., 0., 0.], [0., 1., 0.], Default::default()).unwrap();
    snapshot.sketches.replace_plane(SKETCH, moved).unwrap();
    assert_eq!(
        snapshot.resolve_sketch_projected_points(SKETCH).unwrap()[0].position,
        [-1., 0.]
    );
    assert!(
        snapshot
            .conditional_origin_projection(SKETCH, ANCHOR, &outcome, assumptions())
            .is_err()
    );
}
#[test]
fn nonfinite_or_misidentified_frame_assumptions_are_refused() {
    let (snapshot, outcome) = source_snapshot();
    let mut invalid = assumptions();
    invalid.origin[0] = [f64::NEG_INFINITY, 0.];
    assert!(
        snapshot
            .conditional_origin_projection(SKETCH, ANCHOR, &outcome, invalid)
            .is_err()
    );
    assert!(
        snapshot
            .conditional_origin_projection(SKETCH, SketchEntityId(999), &outcome, assumptions())
            .is_err()
    );
}
#[test]
fn nonconstant_axis_assumptions_enclose_exact_corner_products() {
    let (snapshot, outcome) = source_snapshot();
    let mut frame = assumptions();
    frame.x_axis[0] = [0.9, 1.1];
    let evidence = snapshot
        .conditional_origin_projection(SKETCH, ANCHOR, &outcome, frame)
        .unwrap();
    let bounds = evidence.local_bounds()[0];
    for origin in frame.origin[0] {
        for direction in frame.x_axis[0] {
            let exact = -q(origin) * q(direction);
            assert!(q(bounds[0]) <= exact && exact <= q(bounds[1]));
        }
    }
}
