// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    parameter::ParameterCollection,
    sketch::{
        PlanarConstraint, Sketch2D, SketchEntityId, SketchInferenceFreeze, SketchPoint2D,
        SketchTransientInference,
    },
};
use oblikovati_solve::nonlinear::SolveOptions;

fn source_sketch() -> Sketch2D {
    let mut source = Sketch2D::default();
    for (id, point) in [(1, [0.0, 0.0]), (2, [10.0, 0.0]), (3, [20.0, 0.0])] {
        source.add_point(SketchEntityId(id), point).unwrap();
    }
    source
        .add_line(SketchEntityId(10), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    source
        .add_circle(SketchEntityId(20), SketchEntityId(3), 3.0)
        .unwrap();
    source
}

#[test]
fn midpoint_inference_freezes_source_and_reports_only_durable_structure() {
    let source = source_sketch();
    let freeze = SketchInferenceFreeze::capture(&source).unwrap();
    let mut candidate = source.clone();
    candidate.add_point(SketchEntityId(4), [4.9, 0.1]).unwrap();
    let relation = PlanarConstraint::Midpoint {
        point: SketchEntityId(4),
        line: SketchEntityId(10),
    };
    let context = SketchTransientInference {
        source: &freeze,
        durable_anchors: &[],
        temporary_anchors: &[],
        temporary_relations: &[relation],
    };
    let parameters = ParameterCollection::new(Default::default());
    let result = candidate
        .solve_transient_inference(&parameters, &context, SolveOptions::default(), || false)
        .unwrap();
    let solved = result.candidate();
    for point in source.points() {
        assert_eq!(
            solved.point(point.id).unwrap().position.map(f64::to_bits),
            point.position.map(f64::to_bits)
        );
    }
    assert_eq!(
        solved.circle(SketchEntityId(20)).unwrap().radius.to_bits(),
        3.0f64.to_bits()
    );
    let midpoint = solved.point(SketchEntityId(4)).unwrap().position;
    assert!((midpoint[0] - 5.0).abs() < 1e-8 && midpoint[1].abs() < 1e-8);
    assert_eq!(solved.constraints(), candidate.constraints());
    assert_eq!(result.durable_structure().degrees_of_freedom, 9);
    assert_eq!(result.durable_structure().equations, 0);
    assert_eq!(
        result.durable_structure(),
        &solved
            .analyze(&parameters, SolveOptions::default().rank_tolerance)
            .unwrap()
    );
    assert_eq!(
        candidate.point(SketchEntityId(4)).unwrap().position,
        [4.9, 0.1]
    );
}

#[test]
fn temporary_arc_anchor_survives_intrinsic_solve_without_a_saved_relation() {
    let mut source = Sketch2D::default();
    source.add_point(SketchEntityId(1), [0.0, 2.0005]).unwrap();
    let freeze = SketchInferenceFreeze::capture(&source).unwrap();
    let mut candidate = source.clone();
    for (id, point) in [(2, [0.0, 0.0]), (3, [2.0, 0.0]), (4, [0.0, 2.0005])] {
        candidate.add_point(SketchEntityId(id), point).unwrap();
    }
    candidate
        .add_arc(
            SketchEntityId(10),
            SketchEntityId(2),
            [SketchEntityId(3), SketchEntityId(4)],
            true,
        )
        .unwrap();
    let anchor = SketchPoint2D {
        id: SketchEntityId(4),
        position: [0.0, 2.0005],
    };
    let context = SketchTransientInference {
        source: &freeze,
        durable_anchors: &[],
        temporary_anchors: &[anchor],
        temporary_relations: &[],
    };
    let parameters = ParameterCollection::new(Default::default());
    let result = candidate
        .solve_transient_inference(&parameters, &context, SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(
        result
            .candidate()
            .point(anchor.id)
            .unwrap()
            .position
            .map(f64::to_bits),
        anchor.position.map(f64::to_bits)
    );
    assert!(result.candidate().constraints().is_empty());
    let durable = result
        .candidate()
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(
        durable
            .candidate()
            .point(anchor.id)
            .unwrap()
            .position
            .map(f64::to_bits),
        anchor.position.map(f64::to_bits)
    );
}

#[test]
fn conflicting_temporary_anchor_refuses_without_mutating_source() {
    let source = source_sketch();
    let freeze = SketchInferenceFreeze::capture(&source).unwrap();
    let anchor = SketchPoint2D {
        id: SketchEntityId(1),
        position: [1.0, 0.0],
    };
    let context = SketchTransientInference {
        source: &freeze,
        durable_anchors: &[],
        temporary_anchors: &[anchor],
        temporary_relations: &[],
    };
    let error = source
        .solve_transient_inference(
            &ParameterCollection::new(Default::default()),
            &context,
            SolveOptions::default(),
            || false,
        )
        .unwrap_err()
        .to_string();
    assert!(error.contains("conflicting"), "{error}");
    assert_eq!(
        source.point(SketchEntityId(1)).unwrap().position,
        [0.0, 0.0]
    );
}

#[path = "sketch_transient_inference/curves.rs"]
mod curves;
#[path = "sketch_transient_inference/performance.rs"]
mod performance;
#[path = "sketch_transient_inference/validation.rs"]
mod validation;
