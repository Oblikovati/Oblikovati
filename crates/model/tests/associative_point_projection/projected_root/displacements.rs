// SPDX-License-Identifier: GPL-2.0-only
use super::*;

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
