// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn changed_source_radius_is_refused_before_transient_solve() {
    let source = source_sketch();
    let freeze = SketchInferenceFreeze::capture(&source).unwrap();
    let mut candidate = source.clone();
    candidate.set_radius(SketchEntityId(20), 4.0).unwrap();
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
        error.contains("source scalar") && error.contains("20"),
        "{error}"
    );
    assert_eq!(candidate.circle(SketchEntityId(20)).unwrap().radius, 4.0);
    assert_eq!(source.circle(SketchEntityId(20)).unwrap().radius, 3.0);
}

#[test]
fn temporary_relation_ids_do_not_replace_existing_high_id_relations() {
    use oblikovati_model::sketch::SketchConstraintId;
    let mut source = source_sketch();
    source
        .add_constraint(
            SketchConstraintId(u64::MAX),
            PlanarConstraint::LineHorizontal {
                line: SketchEntityId(10),
            },
        )
        .unwrap();
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
    let result = candidate
        .solve_transient_inference(
            &ParameterCollection::new(Default::default()),
            &context,
            SolveOptions::default(),
            || false,
        )
        .unwrap();
    assert_eq!(result.candidate().constraints(), source.constraints());
    assert_eq!(
        result.candidate().constraints()[0].id,
        SketchConstraintId(u64::MAX)
    );
}

#[test]
fn cancelled_transient_solve_does_not_publish_guidance() {
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
    let error = candidate
        .solve_transient_inference(
            &ParameterCollection::new(Default::default()),
            &context,
            SolveOptions::default(),
            || true,
        )
        .unwrap_err()
        .to_string();
    assert!(error.contains("Cancelled"), "{error}");
    assert!(candidate.constraints().is_empty());
    assert_eq!(
        candidate.point(SketchEntityId(4)).unwrap().position,
        [4.9, 0.1]
    );
}

#[test]
fn stale_existing_point_is_rejected_before_guidance_is_applied() {
    let source = source_sketch();
    let freeze = SketchInferenceFreeze::capture(&source).unwrap();
    let mut candidate = source.clone();
    candidate
        .set_position(SketchEntityId(1), [0.1, 0.0])
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
        error.contains("source point") && error.contains("unchanged"),
        "{error}"
    );
    assert_eq!(
        candidate.point(SketchEntityId(1)).unwrap().position,
        [0.1, 0.0]
    );
}

#[test]
fn duplicate_identical_anchors_are_merged_without_changing_the_durable_graph() {
    let source = source_sketch();
    let freeze = SketchInferenceFreeze::capture(&source).unwrap();
    let same = SketchPoint2D {
        id: SketchEntityId(1),
        position: [0.0, 0.0],
    };
    let context = SketchTransientInference {
        source: &freeze,
        durable_anchors: &[same],
        temporary_anchors: &[same],
        temporary_relations: &[],
    };
    let result = source
        .solve_transient_inference(
            &ParameterCollection::new(Default::default()),
            &context,
            SolveOptions::default(),
            || false,
        )
        .unwrap();
    assert_eq!(
        result.candidate().point(same.id).unwrap().position,
        same.position
    );
    assert!(result.candidate().constraints().is_empty());
}

#[test]
fn a_temporary_dimension_is_refused_before_identity_allocation() {
    use oblikovati_model::sketch::DistanceMeasure;
    let source = source_sketch();
    let freeze = SketchInferenceFreeze::capture(&source).unwrap();
    let relation = PlanarConstraint::Distance {
        points: [SketchEntityId(1), SketchEntityId(2)],
        measure: DistanceMeasure::Aligned,
        parameter: 1,
    };
    let context = SketchTransientInference {
        source: &freeze,
        durable_anchors: &[],
        temporary_anchors: &[],
        temporary_relations: &[relation],
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
    assert!(error.contains("expected a geometric relation"), "{error}");
    assert!(source.constraints().is_empty());
}

#[test]
fn removing_an_existing_relation_refuses_the_candidate() {
    use oblikovati_model::sketch::SketchConstraintId;
    let mut source = source_sketch();
    let original = SketchConstraintId(7);
    source
        .add_constraint(
            original,
            PlanarConstraint::LineHorizontal {
                line: SketchEntityId(10),
            },
        )
        .unwrap();
    let freeze = SketchInferenceFreeze::capture(&source).unwrap();
    let mut candidate = source.clone();
    candidate.remove_geometric_constraint(original).unwrap();
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
    assert!(error.contains("source constraints changed"), "{error}");
    assert_eq!(source.constraints()[0].id, original);
}

#[test]
fn an_unknown_temporary_anchor_preserves_the_original_typed_cause() {
    use std::error::Error;
    let source = source_sketch();
    let freeze = SketchInferenceFreeze::capture(&source).unwrap();
    let unknown = SketchPoint2D {
        id: SketchEntityId(99),
        position: [0.0, 0.0],
    };
    let context = SketchTransientInference {
        source: &freeze,
        durable_anchors: &[],
        temporary_anchors: &[unknown],
        temporary_relations: &[],
    };
    let error = source
        .solve_transient_inference(
            &ParameterCollection::new(Default::default()),
            &context,
            SolveOptions::default(),
            || false,
        )
        .unwrap_err();
    assert!(error.to_string().contains("99"));
    assert!(error.source().is_some());
}
