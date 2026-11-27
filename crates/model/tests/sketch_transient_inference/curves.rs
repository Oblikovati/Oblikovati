// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn on_circle_guidance_freezes_radius_while_moving_only_the_new_point() {
    let source = source_sketch();
    let freeze = SketchInferenceFreeze::capture(&source).unwrap();
    let mut candidate = source.clone();
    candidate.add_point(SketchEntityId(4), [24.0, 0.0]).unwrap();
    let relation = PlanarConstraint::PointOnCircularCurve {
        point: SketchEntityId(4),
        curve: SketchEntityId(20),
    };
    let context = SketchTransientInference {
        source: &freeze,
        durable_anchors: &[],
        temporary_anchors: &[],
        temporary_relations: &[relation],
    };
    let result = candidate
        .solve_transient_inference(
            &ParameterCollection::new(Default::default()),
            &context,
            SolveOptions::default(),
            || false,
        )
        .unwrap();
    let solved = result.candidate();
    assert_eq!(
        solved.circle(SketchEntityId(20)).unwrap().radius.to_bits(),
        3.0f64.to_bits()
    );
    assert_eq!(
        solved.point(SketchEntityId(3)).unwrap().position,
        [20.0, 0.0]
    );
    assert!((solved.point(SketchEntityId(4)).unwrap().position[0] - 23.0).abs() < 1e-8);
    assert!(solved.constraints().is_empty());
}

#[test]
fn temporary_alignment_cannot_rotate_an_existing_ellipse() {
    use oblikovati_model::sketch::SketchEllipse2D;
    let mut source = Sketch2D::default();
    source.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    source
        .add_ellipse(
            SketchEllipse2D::new(SketchEntityId(2), SketchEntityId(1), [1.0, 1.0], [4.0, 2.0])
                .unwrap(),
        )
        .unwrap();
    let original_axis = source.ellipse(SketchEntityId(2)).unwrap().major_axis();
    let freeze = SketchInferenceFreeze::capture(&source).unwrap();
    let relation = PlanarConstraint::EllipseAxisAlignment {
        ellipse: SketchEntityId(2),
        major_axis: true,
        vertical: false,
    };
    let context = SketchTransientInference {
        source: &freeze,
        durable_anchors: &[],
        temporary_anchors: &[],
        temporary_relations: &[relation],
    };
    assert!(
        source
            .solve_transient_inference(
                &ParameterCollection::new(Default::default()),
                &context,
                SolveOptions::default(),
                || false
            )
            .is_err()
    );
    assert_eq!(
        source.ellipse(SketchEntityId(2)).unwrap().major_axis(),
        original_axis
    );
}

#[test]
fn changed_existing_line_endpoints_are_not_accepted_as_an_append() {
    let source = source_sketch();
    let freeze = SketchInferenceFreeze::capture(&source).unwrap();
    let mut candidate = Sketch2D::default();
    for point in source.points() {
        candidate.add_point(point.id, point.position).unwrap();
    }
    candidate
        .add_line(SketchEntityId(10), [SketchEntityId(2), SketchEntityId(1)])
        .unwrap();
    candidate
        .add_circle(SketchEntityId(20), SketchEntityId(3), 3.0)
        .unwrap();
    let context = SketchTransientInference {
        source: &freeze,
        durable_anchors: &[],
        temporary_anchors: &[],
        temporary_relations: &[],
    };
    let error = candidate
        .solve_transient_inference(
            &ParameterCollection::new(Default::default()),
            &context,
            SolveOptions::default(),
            || false,
        )
        .unwrap_err()
        .to_string();
    assert!(
        error.contains("source line or arc topology changed"),
        "{error}"
    );
}
