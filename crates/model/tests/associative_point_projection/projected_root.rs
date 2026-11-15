// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use num_rational::BigRational;
use oblikovati_model::{
    history::SketchFrameInputBounds,
    sketch::{DistanceMeasure, PlanarConstraint, SketchConstraintId, SketchSolveOutcome},
};

fn rectangle_document(omit: bool, duplicate: bool) -> (DocumentSnapshot, SketchSolveOutcome) {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .parameters
        .insert_expression(1, "size", USER_PARAM, "10 mm")
        .unwrap();
    let mut geometry = Sketch2D::default();
    for (id, point) in [
        (11, [0., 0.]),
        (1, [0., 0.]),
        (2, [1., 0.]),
        (3, [1., 1.]),
        (4, [0., 1.]),
    ] {
        geometry.add_point(SketchEntityId(id), point).unwrap();
    }
    let mut add = |id, relation| {
        geometry
            .add_constraint(SketchConstraintId(id), relation)
            .unwrap()
    };
    add(
        1,
        PlanarConstraint::Coincident {
            points: [ANCHOR, SketchEntityId(1)],
        },
    );
    add(
        2,
        PlanarConstraint::Horizontal {
            points: [SketchEntityId(1), SketchEntityId(2)],
        },
    );
    add(
        3,
        PlanarConstraint::Horizontal {
            points: [SketchEntityId(3), SketchEntityId(4)],
        },
    );
    add(
        4,
        PlanarConstraint::Vertical {
            points: [SketchEntityId(2), SketchEntityId(3)],
        },
    );
    add(
        5,
        PlanarConstraint::Vertical {
            points: [SketchEntityId(4), SketchEntityId(1)],
        },
    );
    add(
        6,
        PlanarConstraint::Distance {
            points: [SketchEntityId(1), SketchEntityId(2)],
            measure: DistanceMeasure::Horizontal,
            parameter: 1,
        },
    );
    if !omit {
        add(
            7,
            PlanarConstraint::Distance {
                points: [SketchEntityId(1), SketchEntityId(4)],
                measure: DistanceMeasure::Vertical,
                parameter: 1,
            },
        );
    }
    if duplicate {
        add(
            8,
            PlanarConstraint::Horizontal {
                points: [SketchEntityId(1), SketchEntityId(2)],
            },
        );
    }
    for (id, endpoints) in [(101, [1, 2]), (102, [2, 3]), (103, [3, 4]), (104, [4, 1])] {
        geometry
            .add_line(SketchEntityId(id), endpoints.map(SketchEntityId))
            .unwrap();
    }
    snapshot
        .sketches
        .add_on_origin_plane(
            SKETCH,
            "Rectangle",
            oblikovati_model::feature::work_plane::WorkPlaneOrigin::XY,
            geometry,
        )
        .unwrap();
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
    let batch = snapshot
        .solve_one_document_sketch(SKETCH, SolveOptions::default(), || false)
        .unwrap();
    let outcome = batch.report.reports[0].result.as_ref().unwrap().clone();
    snapshot.sketches = batch.candidate;
    (snapshot, outcome)
}
fn inverse() -> Vec<Vec<f64>> {
    vec![
        vec![0., 0., 0., 0., 0., 0., 0., 0., 1., 0.],
        vec![0., 0., 0., 0., 0., 0., 0., 0., 0., 1.],
        vec![-1., 0., 0., 0., 0., 0., 0., 0., 1., 0.],
        vec![0., -1., 0., 0., 0., 0., 0., 0., 0., 1.],
        vec![-1., 0., 0., 0., 0., 0., 1., 0., 1., 0.],
        vec![0., -1., -1., 0., 0., 0., 0., 0., 0., 1.],
        vec![-1., 0., 0., 0., -1., 0., 1., 0., 1., 0.],
        vec![0., -1., 0., 1., 0., 0., 0., 1., 0., 1.],
        vec![-1., 0., 0., 0., 0., 1., 0., 0., 1., 0.],
        vec![0., -1., 0., 0., 0., 0., 0., 1., 0., 1.],
    ]
}
fn assumptions() -> SketchFrameInputBounds {
    SketchFrameInputBounds {
        origin: [[-0.0001, 0.0001], [-0.0001, 0.0001], [0., 0.]],
        x_axis: [[1., 1.], [0., 0.], [0., 0.]],
        y_axis: [[0., 0.], [1., 1.], [0., 0.]],
    }
}
fn full_box(outcome: &SketchSolveOutcome) -> Vec<[f64; 2]> {
    outcome
        .candidate()
        .points()
        .iter()
        .flat_map(|point| point.position)
        .map(|value| [value - 0.01, value + 0.01])
        .collect()
}
#[test]
fn projected_rectangle_root_encloses_independent_origin_and_size_corners() {
    let (snapshot, outcome) = rectangle_document(false, false);
    assert_eq!(outcome.numerical().values.len(), 8);
    assert_eq!(outcome.numerical().evaluation.values().len(), 8);
    let projection = snapshot
        .conditional_origin_projection(SKETCH, ANCHOR, &outcome, assumptions())
        .unwrap();
    let root = projection
        .conditional_root(&full_box(&outcome), &[], &inverse())
        .unwrap();
    assert_eq!(root.root().relations().len(), 8);
    assert_eq!(
        root.full_to_active(),
        &[
            None,
            None,
            Some(0),
            Some(1),
            Some(2),
            Some(3),
            Some(4),
            Some(5),
            Some(6),
            Some(7)
        ]
    );
    let (anchor, source_bounds) = root.root().projected_anchor().unwrap();
    assert_eq!(anchor.id, ANCHOR);
    assert_eq!(source_bounds, projection.local_bounds());
    assert_eq!(root.projection().source(), WorkPointReference::Origin);
    assert_eq!(root.root().source_rows().len(), 8);
    for x in [-0.0001, 0.0001] {
        for y in [-0.0001, 0.0001] {
            let q = |value| BigRational::from_float(value).unwrap();
            let (x, y, one) = (q(x), q(y), q(1.));
            let exact = [
                x.clone(),
                y.clone(),
                x.clone(),
                y.clone(),
                &x + &one,
                y.clone(),
                &x + &one,
                &y + &one,
                x,
                &y + &one,
            ];
            for (value, interval) in exact.into_iter().zip(root.root().enclosure()) {
                assert!(q(interval[0]) <= value && value <= q(interval[1]));
            }
        }
    }
}
#[test]
fn projected_system_refuses_non_square_and_redundant_equations() {
    for (omit, duplicate) in [(true, false), (false, true), (true, true)] {
        let (snapshot, outcome) = rectangle_document(omit, duplicate);
        let projection = snapshot
            .conditional_origin_projection(SKETCH, ANCHOR, &outcome, assumptions())
            .unwrap();
        assert!(
            projection
                .conditional_root(&full_box(&outcome), &[], &inverse())
                .is_err()
        );
    }
}
#[test]
fn projected_origin_bound_must_fit_the_variable_box() {
    let (snapshot, outcome) = rectangle_document(false, false);
    let projection = snapshot
        .conditional_origin_projection(SKETCH, ANCHOR, &outcome, assumptions())
        .unwrap();
    let mut bounds = full_box(&outcome);
    bounds[0] = [-0.00001, 0.00001];
    assert!(
        projection
            .conditional_root(&bounds, &[], &inverse())
            .is_err()
    );
}

#[test]
fn projected_root_rejects_same_value_parameter_source_edits() {
    let (mut snapshot, outcome) = rectangle_document(false, false);
    snapshot.parameters.set_expression(1, "1 cm").unwrap();
    assert_eq!(snapshot.parameters.model_value(1).unwrap().value, 1.);
    assert!(
        snapshot
            .conditional_origin_projection(SKETCH, ANCHOR, &outcome, assumptions())
            .is_err()
    );
}
#[test]
fn projected_root_rejects_frame_change_with_same_rounded_anchor() {
    let (mut snapshot, outcome) = rectangle_document(false, false);
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
fn changed_projection_source_is_rejected_even_when_local_coordinates_match() {
    let (mut snapshot, outcome) = rectangle_document(false, false);
    let source = WorkGeometryDefinition::Point([DatumLength::Constant(0.0); 3]);
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(1), "Same point", source))
        .unwrap();
    snapshot
        .sketches
        .set_point_projections(
            SKETCH,
            vec![SketchPointProjection {
                anchor: ANCHOR,
                reference: WorkPointReference::Feature(FeatureId(1)),
            }],
        )
        .unwrap();
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
fn canonical_xy_identity_supplies_exact_origin_bounds_for_rectangle() {
    use oblikovati_model::{feature::work_plane::WorkPlaneOrigin, history::ConditionalFrameSource};
    let (snapshot, outcome) = rectangle_document(false, false);
    let projection = snapshot
        .canonical_origin_projection(SKETCH, ANCHOR, &outcome)
        .unwrap();
    assert_eq!(
        projection.frame_source(),
        ConditionalFrameSource::CanonicalOrigin(WorkPlaneOrigin::XY)
    );
    assert_eq!(projection.local_bounds(), [[0., 0.], [0., 0.]]);
    let result = projection
        .conditional_root(&full_box(&outcome), &[], &inverse())
        .unwrap();
    for (exact, bounds) in [0., 0., 0., 0., 1., 0., 1., 1., 0., 1.]
        .into_iter()
        .zip(result.root().enclosure())
    {
        assert!(bounds[0] <= exact && exact <= bounds[1]);
    }
}
#[test]
fn fixed_lookalike_and_edited_plane_cannot_claim_canonical_identity() {
    let (mut snapshot, outcome) = rectangle_document(false, false);
    snapshot
        .sketches
        .replace_plane(SKETCH, SketchPlane::xy())
        .unwrap();
    assert!(
        snapshot
            .canonical_origin_projection(SKETCH, ANCHOR, &outcome)
            .is_err()
    );
}

#[test]
fn all_canonical_origin_plane_identities_bound_the_projected_origin() {
    use oblikovati_model::{
        document_sketch::DocumentSketches, feature::work_plane::WorkPlaneOrigin,
        history::ConditionalFrameSource,
    };
    for origin in [
        WorkPlaneOrigin::XY,
        WorkPlaneOrigin::XZ,
        WorkPlaneOrigin::YZ,
    ] {
        let (mut snapshot, _) = rectangle_document(false, false);
        let mut recipe = snapshot.sketches.recipe();
        recipe.sketches[0].plane_origin = Some(origin);
        snapshot.sketches = DocumentSketches::from_recipe(&recipe).unwrap();
        let batch = snapshot
            .solve_one_document_sketch(SKETCH, SolveOptions::default(), || false)
            .unwrap();
        let outcome = batch.report.reports[0].result.as_ref().unwrap().clone();
        snapshot.sketches = batch.candidate;
        let projection = snapshot
            .canonical_origin_projection(SKETCH, ANCHOR, &outcome)
            .unwrap();
        assert_eq!(
            projection.frame_source(),
            ConditionalFrameSource::CanonicalOrigin(origin)
        );
        assert_eq!(projection.local_bounds(), [[0., 0.], [0., 0.]]);
    }
}
#[test]
fn canonical_origin_rejects_stale_same_value_parameter_source() {
    let (mut snapshot, outcome) = rectangle_document(false, false);
    snapshot.parameters.set_expression(1, "1 cm").unwrap();
    assert_eq!(snapshot.parameters.model_value(1).unwrap().value, 1.);
    assert!(
        snapshot
            .canonical_origin_projection(SKETCH, ANCHOR, &outcome)
            .is_err()
    );
}

#[test]
fn generated_preconditioner_checks_the_actual_canonical_rectangle() {
    let (snapshot, outcome) = rectangle_document(false, false);
    let projection = snapshot
        .canonical_origin_projection(SKETCH, ANCHOR, &outcome)
        .unwrap();
    let result = projection
        .conditional_root_auto(&full_box(&outcome), &[])
        .unwrap();
    let q = |v| BigRational::from_float(v).unwrap();
    for (exact, bounds) in [0., 0., 0., 0., 1., 0., 1., 1., 0., 1.]
        .into_iter()
        .zip(result.root().enclosure())
    {
        assert!(q(bounds[0]) <= q(exact) && q(exact) <= q(bounds[1]));
    }
    assert!(result.root().contraction_upper() < 1.0);
}
#[test]
fn generated_preconditioner_rejects_non_square_rectangle() {
    for (omit, duplicate) in [(true, false), (false, true), (true, true)] {
        let (snapshot, outcome) = rectangle_document(omit, duplicate);
        let projection = snapshot
            .canonical_origin_projection(SKETCH, ANCHOR, &outcome)
            .unwrap();
        assert!(
            projection
                .conditional_root_auto(&full_box(&outcome), &[])
                .is_err()
        );
    }
}

#[test]
fn canonical_point_displacements_cover_exact_rectangle_corners_in_all_origin_frames() {
    use oblikovati_model::{
        document_sketch::DocumentSketches, feature::work_plane::WorkPlaneOrigin,
    };
    let q = |v| BigRational::from_float(v).unwrap();
    for origin in [
        WorkPlaneOrigin::XY,
        WorkPlaneOrigin::XZ,
        WorkPlaneOrigin::YZ,
    ] {
        let (mut snapshot, _) = rectangle_document(false, false);
        let mut recipe = snapshot.sketches.recipe();
        recipe.sketches[0].plane_origin = Some(origin);
        snapshot.sketches = DocumentSketches::from_recipe(&recipe).unwrap();
        let batch = snapshot
            .solve_one_document_sketch(SKETCH, SolveOptions::default(), || false)
            .unwrap();
        let outcome = batch.report.reports[0].result.as_ref().unwrap().clone();
        snapshot.sketches = batch.candidate;
        let projection = snapshot
            .canonical_origin_projection(SKETCH, ANCHOR, &outcome)
            .unwrap();
        let result = projection
            .conditional_root_auto(&full_box(&outcome), &[])
            .unwrap();
        let distances = result.canonical_point_displacements().unwrap();
        for (index, (point, distance)) in outcome
            .candidate()
            .points()
            .iter()
            .zip(distances)
            .enumerate()
        {
            assert_eq!(point.id, distance.point);
            let exact = [[0., 0.], [0., 0.], [1., 0.], [1., 1.], [0., 1.]][index];
            let dx = q(point.position[0]) - q(exact[0]);
            let dy = q(point.position[1]) - q(exact[1]);
            assert!(&dx * &dx + &dy * &dy <= q(distance.upper) * q(distance.upper));
        }
    }
}
#[test]
fn assumed_frame_cannot_claim_canonical_point_displacements() {
    let (snapshot, outcome) = rectangle_document(false, false);
    let projection = snapshot
        .conditional_origin_projection(SKETCH, ANCHOR, &outcome, assumptions())
        .unwrap();
    let result = projection
        .conditional_root_auto(&full_box(&outcome), &[])
        .unwrap();
    assert!(
        result
            .canonical_point_displacements()
            .unwrap_err()
            .contains("canonical origin plane")
    );
}

#[test]
fn rectangle_segments_have_source_attributed_conditional_bounds() {
    let (snapshot, outcome) = rectangle_document(false, false);
    let projection = snapshot
        .canonical_origin_projection(SKETCH, ANCHOR, &outcome)
        .unwrap();
    let root = projection
        .conditional_root_auto(&full_box(&outcome), &[])
        .unwrap();
    let bounds = root.canonical_segment_errors().unwrap();
    assert_eq!(bounds.len(), 4);
    let points = root.root().sketch().points();
    let q = |x| BigRational::from_float(x).unwrap();
    for (line, bound) in root.root().sketch().lines().iter().zip(bounds) {
        assert_eq!((line.id, line.endpoints), (bound.line, bound.endpoints));
        let exact = |id: SketchEntityId| match id.0 {
            1 => [0., 0.],
            2 => [1., 0.],
            3 => [1., 1.],
            4 => [0., 1.],
            _ => unreachable!(),
        };
        let p = line
            .endpoints
            .map(|id| points.iter().find(|p| p.id == id).unwrap().position);
        let e = line.endpoints.map(exact);
        for t in [0., 0.25, 0.5, 0.75, 1.] {
            let t = q(t);
            let one = q(1.);
            let d = |axis: usize| {
                (&one - &t) * (q(p[0][axis]) - q(e[0][axis])) + &t * (q(p[1][axis]) - q(e[1][axis]))
            };
            let (dx, dy) = (d(0), d(1));
            assert!(
                &dx * &dx + &dy * &dy <= q(bound.displacement_upper) * q(bound.displacement_upper)
            );
        }
        let accepted = [q(p[1][0]) - q(p[0][0]), q(p[1][1]) - q(p[0][1])];
        let exact = [q(e[1][0]) - q(e[0][0]), q(e[1][1]) - q(e[0][1])];
        let cross = &accepted[0] * &exact[1] - &accepted[1] * &exact[0];
        let dot = &accepted[0] * &exact[0] + &accepted[1] * &exact[1];
        assert!(dot > q(0.));
        let ratio = cross / dot;
        let angle_bound = q(bound.angle_upper_radians);
        assert!(-angle_bound.clone() <= ratio && ratio <= angle_bound);
    }
}
#[test]
fn segment_error_declines_unproven_frame() {
    let (snapshot, outcome) = rectangle_document(false, false);
    let projection = snapshot
        .conditional_origin_projection(SKETCH, ANCHOR, &outcome, assumptions())
        .unwrap();
    let root = projection
        .conditional_root_auto(&full_box(&outcome), &[])
        .unwrap();
    assert!(root.canonical_segment_errors().is_err());
}
