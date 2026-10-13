// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{
    FeatureEvaluationStatus,
    part::{PartFeatureFailure, SketchPrismEvaluator},
    sketch_prism::ResolvedSketchPrism,
};
#[derive(Debug, PartialEq)]
struct FakePrismBody {
    plane: SketchPlane,
    displacement: [f64; 3],
}
#[derive(Default)]
struct FakePrismEvaluator {
    calls: usize,
    fail_first: bool,
}
impl SketchPrismEvaluator for FakePrismEvaluator {
    type Body = FakePrismBody;
    type Failure = &'static str;
    fn evaluate_prism(
        &mut self,
        _feature: oblikovati_model::feature::FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<Self::Body, Self::Failure> {
        self.calls += 1;
        if self.fail_first && self.calls == 1 {
            return Err("kernel refused first prism");
        }
        Ok(FakePrismBody {
            plane: input.sketch.plane,
            displacement: input.displacement,
        })
    }
}
fn replay_document() -> DocumentSnapshot {
    let mut owner = part_owner();
    append_prism(&mut owner);
    owner.snapshot().clone()
}
fn append_independent_prism(snapshot: &mut DocumentSnapshot) {
    let feature =
        AuthoredFeature::sketch_prism(FeatureId(3), "Independent prism", prism_source(), snapshot)
            .unwrap();
    snapshot.work_geometry.append(feature).unwrap();
}
fn append_independent_point(snapshot: &mut DocumentSnapshot) {
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(
            FeatureId(3),
            "Independent point",
            PartFeatureDefinition::Point([DatumLength::Constant(0.); 3]),
        ))
        .unwrap();
}
#[test]
fn common_replay_owns_current_results_and_one_ordered_health_report() {
    let document = replay_document();
    let mut backend = FakePrismEvaluator::default();
    let evaluated = document.evaluate_part_features(&mut backend).unwrap();
    assert_eq!(backend.calls, 1);
    assert_eq!(evaluated.plane(FeatureId(1)).unwrap(), SketchPlane::xy());
    assert!(evaluated.point(FeatureId(1)).is_err());
    assert!(evaluated.axis(FeatureId(1)).is_err());
    assert!(evaluated.body(FeatureId(1)).is_none());
    assert!(
        evaluated
            .health()
            .iter()
            .all(|entry| entry.status == FeatureEvaluationStatus::Healthy)
    );
    verify_fake_body(evaluated.into_body(FeatureId(2)).unwrap());
}
#[test]
fn failed_datum_quarantines_its_prism_reader_and_keeps_independent_datums() {
    let mut document = replay_document();
    fail_host_datum(&mut document);
    document
        .work_geometry
        .set_suppressed(FeatureId(2), true)
        .unwrap();
    append_independent_point(&mut document);
    let mut backend = FakePrismEvaluator::default();
    let evaluated = document.evaluate_part_features(&mut backend).unwrap();
    assert_eq!(backend.calls, 0);
    verify_blocked_prism(&evaluated);
    assert_eq!(evaluated.point(FeatureId(3)).unwrap(), [0.; 3]);
    assert!(evaluated.body(FeatureId(2)).is_none());
}
#[test]
fn kernel_failure_is_local_and_a_new_replay_can_recover_without_stale_outputs() {
    let mut document = replay_document();
    append_independent_prism(&mut document);
    let mut backend = FakePrismEvaluator {
        calls: 0,
        fail_first: true,
    };
    let failed = document.evaluate_part_features(&mut backend).unwrap();
    assert_eq!(backend.calls, 2);
    assert!(failed.body(FeatureId(2)).is_none());
    assert!(failed.body(FeatureId(3)).is_some());
    verify_kernel_failure(&failed);
    let recovered = document
        .evaluate_part_features(&mut FakePrismEvaluator::default())
        .unwrap();
    assert!(recovered.body(FeatureId(2)).is_some());
    assert!(failed.into_body(FeatureId(2)).is_none());
}
#[test]
fn unresolved_current_inputs_never_reach_the_kernel() {
    let mut document = replay_document();
    document
        .parameters
        .set_expression(3, "missing_depth")
        .unwrap();
    append_independent_point(&mut document);
    let mut backend = FakePrismEvaluator::default();
    let evaluated = document.evaluate_part_features(&mut backend).unwrap();
    assert_eq!(backend.calls, 0);
    assert!(matches!(
        evaluated.health()[1].status,
        FeatureEvaluationStatus::Failed(PartFeatureFailure::Inputs(_))
    ));
    assert!(evaluated.body(FeatureId(2)).is_none());
    assert_eq!(evaluated.point(FeatureId(3)).unwrap(), [0.; 3]);
}
#[test]
fn cutoff_and_empty_programs_do_not_call_the_kernel() {
    let mut document = replay_document();
    document.work_geometry.set_end_of_part(Some(1)).unwrap();
    let mut backend = FakePrismEvaluator::default();
    let evaluated = document.evaluate_part_features(&mut backend).unwrap();
    assert_eq!(evaluated.health().len(), 1);
    assert!(evaluated.body(FeatureId(2)).is_none());
    document.work_geometry.set_end_of_part(Some(0)).unwrap();
    assert!(
        document
            .evaluate_part_features(&mut backend)
            .unwrap()
            .health()
            .is_empty()
    );
    assert_eq!(backend.calls, 0);
}

fn fail_host_datum(document: &mut DocumentSnapshot) {
    let failed = PartFeatureDefinition::Offset {
        base: oblikovati_model::feature::work_plane::WorkPlaneReference::Origin(
            oblikovati_model::feature::work_plane::WorkPlaneOrigin::XY,
        ),
        distance: DatumLength::Parameter(99),
    };
    document
        .work_geometry
        .replace(FeatureId(1), failed.clone(), failed.reads())
        .unwrap();
}

fn verify_fake_body(body: FakePrismBody) {
    assert_eq!(
        body,
        FakePrismBody {
            plane: SketchPlane::xy(),
            displacement: [0., 0., 2.]
        }
    );
}

fn verify_kernel_failure(
    evaluation: &oblikovati_model::feature::part::PartFeatureEvaluation<
        FakePrismBody,
        &'static str,
    >,
) {
    assert_eq!(
        evaluation.health()[1].status,
        FeatureEvaluationStatus::Failed(PartFeatureFailure::Kernel("kernel refused first prism"))
    );
}

fn verify_blocked_prism(
    evaluated: &oblikovati_model::feature::part::PartFeatureEvaluation<FakePrismBody, &'static str>,
) {
    assert!(matches!(
        evaluated.health()[0].status,
        FeatureEvaluationStatus::Failed(PartFeatureFailure::Datum(_))
    ));
    assert_eq!(
        evaluated.health()[1].status,
        FeatureEvaluationStatus::Blocked {
            drivers: vec![FeatureId(1)]
        }
    );
}

fn projected_replay_document() -> DocumentSnapshot {
    let mut document = replay_document();
    append_independent_point(&mut document);
    document.work_geometry.reorder(FeatureId(3), 1).unwrap();
    bind_projected_origin(&mut document);
    document.refresh_prism_reads().unwrap();
    document.sketches = document
        .solve_pending_document_sketches(Default::default(), || false)
        .candidate;
    document
}
#[test]
fn source_point_projection_and_host_frame_share_the_current_replay_catalogue() {
    let document = projected_replay_document();
    let evaluated = document
        .evaluate_part_features(&mut FakePrismEvaluator::default())
        .unwrap();
    assert_eq!(evaluated.point(FeatureId(3)).unwrap(), [0.; 3]);
    verify_fake_body(evaluated.into_body(FeatureId(2)).unwrap());
}

fn bind_projected_origin(document: &mut DocumentSnapshot) {
    use oblikovati_model::{
        document_sketch::SketchPointProjection, feature::work_geometry::WorkPointReference,
        sketch::SketchEntityId,
    };
    document
        .sketches
        .set_point_projections(
            SKETCH,
            vec![SketchPointProjection {
                anchor: SketchEntityId(1),
                reference: WorkPointReference::Feature(FeatureId(3)),
            }],
        )
        .unwrap();
}

#[test]
fn incomplete_saved_dependency_coverage_refuses_replay_before_kernel_calls() {
    let mut document = replay_document();
    let definition = PartFeatureDefinition::SketchPrism(Box::new(prism_source()));
    document
        .work_geometry
        .replace(FeatureId(2), definition, Default::default())
        .unwrap();
    let mut backend = FakePrismEvaluator::default();
    let error = document.evaluate_part_features(&mut backend).err().unwrap();
    assert!(error.to_string().contains("FeatureId(2)"));
    assert_eq!(backend.calls, 0);
}

#[test]
fn body_metadata_is_refused_before_kernel_construction() {
    let mut document = replay_document();
    document
        .work_geometry
        .set_work_plane_construction(FeatureId(2), true)
        .unwrap();
    let mut backend = FakePrismEvaluator::default();
    let evaluated = document.evaluate_part_features(&mut backend).unwrap();
    assert_eq!(backend.calls, 0);
    assert!(evaluated.body(FeatureId(2)).is_none());
    assert!(matches!(
        evaluated.health()[1].status,
        FeatureEvaluationStatus::Failed(PartFeatureFailure::Metadata(_))
    ));
}
