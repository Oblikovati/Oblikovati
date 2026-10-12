// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::MartensitePrismEvaluator;
use oblikovati_model::feature::{
    AuthoredFeature, FeatureEvaluationStatus,
    part::{PartFeatureDefinition, PartFeatureEvaluation, PartFeatureFailure},
    work_plane::{WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneOrigin, WorkPlaneReference},
};
type NativePartEvaluation =
    PartFeatureEvaluation<oblikovati_martensite::PrismProbeSnapshot, String>;
fn replace_prism_source(document: &mut DocumentSnapshot, definition: SketchPrismDefinition) {
    let reads = definition.reads(document).unwrap();
    document
        .work_geometry
        .replace(
            FeatureId(1),
            PartFeatureDefinition::SketchPrism(Box::new(definition)),
            reads,
        )
        .unwrap();
}
fn broken_boundary_document() -> DocumentSnapshot {
    let mut document = stored_owner().snapshot().clone();
    let mut recipe = stored_definition().recipe();
    recipe.selection.outer[0].reversed = true;
    replace_prism_source(
        &mut document,
        SketchPrismDefinition::from_recipe(&recipe).unwrap(),
    );
    let independent = AuthoredFeature::sketch_prism(
        FeatureId(2),
        "Independent prism",
        stored_definition(),
        &document,
    )
    .unwrap();
    document.work_geometry.append(independent).unwrap();
    document
}
fn assert_current_volume(evaluated: &NativePartEvaluation, id: u64) {
    let body = evaluated.body(FeatureId(id)).unwrap();
    assert_volume(body.report().properties.volume, 24.);
}
#[test]
fn kernel_boundary_decline_keeps_independent_current_prism_and_recovers() {
    let mut document = broken_boundary_document();
    let failed = document
        .evaluate_part_features(&mut MartensitePrismEvaluator)
        .unwrap();
    assert!(failed.body(FeatureId(1)).is_none());
    assert!(matches!(
        failed.health()[0].status,
        FeatureEvaluationStatus::Failed(PartFeatureFailure::Kernel(_))
    ));
    assert_current_volume(&failed, 2);
    replace_prism_source(&mut document, stored_definition());
    let recovered = document
        .evaluate_part_features(&mut MartensitePrismEvaluator)
        .unwrap();
    assert_current_volume(&recovered, 1);
    assert!(failed.into_body(FeatureId(1)).is_none());
}
fn append_failed_host(document: &mut DocumentSnapshot) {
    let definition = WorkPlaneDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: WorkPlaneDistance::Parameter(99),
    };
    document
        .work_geometry
        .append(AuthoredFeature::work_plane(
            FeatureId(2),
            "Host plane",
            definition,
        ))
        .unwrap();
}
fn hosted_document() -> DocumentSnapshot {
    let mut document = stored_owner().snapshot().clone();
    append_failed_host(&mut document);
    document.work_geometry.reorder(FeatureId(2), 0).unwrap();
    document
        .sketches
        .attach_work_plane(SKETCH, FeatureId(2))
        .unwrap();
    document.refresh_prism_reads().unwrap();
    document
}
fn recover_host(document: &mut DocumentSnapshot) {
    let definition = WorkPlaneDefinition::Fixed(oblikovati_model::sketch::SketchPlane::xy());
    document
        .work_geometry
        .replace(FeatureId(2), definition, Default::default())
        .unwrap();
}
#[test]
fn failed_host_plane_blocks_native_prism_until_current_datum_recovers() {
    let mut document = hosted_document();
    let failed = document
        .evaluate_part_features(&mut MartensitePrismEvaluator)
        .unwrap();
    assert_eq!(
        failed.health()[1].status,
        FeatureEvaluationStatus::Blocked {
            drivers: vec![FeatureId(2)]
        }
    );
    assert!(failed.body(FeatureId(1)).is_none());
    recover_host(&mut document);
    let recovered = document
        .evaluate_part_features(&mut MartensitePrismEvaluator)
        .unwrap();
    assert_current_volume(&recovered, 1);
    assert!(failed.into_body(FeatureId(1)).is_none());
}
