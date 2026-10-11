// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    document_sketch::{
        DocumentSketchId, DocumentSketches, PlacedSketch, PlacedSketchHealth,
        SketchPointProjection, SketchReadinessCause,
    },
    feature::{
        AuthoredFeature, FeatureId, FeatureReadKey,
        work_geometry::{DatumLength, WorkGeometryDefinition, WorkPointReference},
    },
    history::{DocumentSketchPlacementError, DocumentSnapshot},
    sketch::{Sketch2D, SketchEntityId, SketchError, SketchPlane},
};
use oblikovati_solve::nonlinear::{SolveOptions, SolveTermination};
use std::error::Error;
const SKETCH: DocumentSketchId = DocumentSketchId(1);
const ANCHOR: SketchEntityId = SketchEntityId(11);
fn snapshot() -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .parameters
        .insert_expression(1, "height", USER_PARAM, "4 cm")
        .unwrap();
    append_source(&mut snapshot);
    snapshot
        .sketches
        .add(SKETCH, "Projected", SketchPlane::xy(), geometry())
        .unwrap();
    snapshot
        .sketches
        .set_point_projections(SKETCH, vec![binding()])
        .unwrap();
    snapshot
}
fn append_source(snapshot: &mut DocumentSnapshot) {
    let definition = source_definition();
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(1), "Source", definition))
        .unwrap();
}
fn source_definition() -> WorkGeometryDefinition {
    WorkGeometryDefinition::Point([
        DatumLength::Constant(2.0),
        DatumLength::Parameter(1),
        DatumLength::Constant(9.0),
    ])
}
fn binding() -> SketchPointProjection {
    SketchPointProjection {
        anchor: ANCHOR,
        reference: WorkPointReference::Feature(FeatureId(1)),
    }
}
fn geometry() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    geometry.add_point(ANCHOR, [99.0, 88.0]).unwrap();
    geometry
}
fn recompute(snapshot: &mut DocumentSnapshot) -> usize {
    let outcome = snapshot.solve_pending_document_sketches(SolveOptions::default(), || false);
    let count = outcome.report.reports.len();
    assert!(!outcome.report.cancelled);
    snapshot.sketches = outcome.candidate;
    count
}
fn position(snapshot: &DocumentSnapshot) -> [f64; 2] {
    snapshot
        .sketches
        .get(SKETCH)
        .unwrap()
        .geometry()
        .point(ANCHOR)
        .unwrap()
        .position
}
fn assert_ready(snapshot: &DocumentSnapshot, expected: [f64; 2]) {
    let view = snapshot.require_resolved_sketch(SKETCH).unwrap();
    assert_eq!(view.verification.termination, SolveTermination::Converged);
    assert_eq!(view.verification.structure.degrees_of_freedom, 0);
    assert_eq!(position(snapshot), expected);
}

#[test]
fn current_source_drives_grounding_and_pending_recompute_without_direct_dimension_reads() {
    let mut snapshot = snapshot();
    assert!(
        snapshot
            .sketches
            .get(SKETCH)
            .unwrap()
            .geometry()
            .parameter_dependencies()
            .is_empty()
    );
    assert_eq!(recompute(&mut snapshot), 1);
    assert_ready(&snapshot, [2.0, 4.0]);
    assert_eq!(recompute(&mut snapshot), 0);
    snapshot.parameters.set_expression(1, "7 cm").unwrap();
    assert_projection_changed(&snapshot);
    assert_eq!(recompute(&mut snapshot), 1);
    assert_ready(&snapshot, [2.0, 7.0]);
}
fn assert_projection_changed(snapshot: &DocumentSnapshot) {
    let error = snapshot.require_resolved_sketch(SKETCH).unwrap_err();
    assert!(
        matches!(error, DocumentSketchPlacementError::Verification(ref error) if error.cause == SketchReadinessCause::ProjectionInputsChanged)
    );
    assert!(error.to_string().contains("expected a current solve"));
}
#[test]
fn host_frame_edit_changes_local_anchor_and_undo_seed_cannot_supply_a_source() {
    let mut snapshot = snapshot();
    recompute(&mut snapshot);
    let plane = SketchPlane::new(
        [10.0, 20.0, 30.0],
        [1.0, 0.0, 0.0],
        [0.0, 1.0, 0.0],
        Default::default(),
    )
    .unwrap();
    snapshot.sketches.replace_plane(SKETCH, plane).unwrap();
    assert_projection_changed(&snapshot);
    recompute(&mut snapshot);
    assert_ready(&snapshot, [-8.0, -16.0]);
}
#[test]
fn standalone_parameter_health_and_solve_refuse_unresolved_point_sources() {
    let mut snapshot = snapshot();
    recompute(&mut snapshot);
    let placed = snapshot.sketches.get(SKETCH).unwrap();
    assert!(matches!(
        placed.health(&snapshot.parameters),
        PlacedSketchHealth::ProjectionInputsRequired
    ));
    let error = placed
        .require_fully_constrained(&snapshot.parameters)
        .unwrap_err();
    assert_eq!(error.cause, SketchReadinessCause::ProjectionInputsRequired);
    let message = error.to_string();
    assert!(message.contains("expected current document resolution"));
    let outcome = snapshot
        .sketches
        .solve(&snapshot.parameters, SolveOptions::default(), || false);
    assert!(outcome.report.reports[0].result.is_err());
    assert!(outcome.candidate.same_recipe(&snapshot.sketches));
}
#[test]
fn source_reads_join_deletion_dependencies_and_source_lookup_uses_identity() {
    let mut snapshot = snapshot();
    let placed = snapshot.sketches.get(SKETCH).unwrap();
    assert_eq!(
        placed.projected_point_for_source(binding().reference),
        Some(ANCHOR)
    );
    assert_eq!(
        placed.projected_point_for_source(WorkPointReference::Origin),
        None
    );
    assert!(
        placed
            .read_footprint()
            .keys()
            .contains(&FeatureReadKey::Feature(FeatureId(1)))
    );
    assert!(snapshot.delete_datum(FeatureId(1)).is_err());
}
#[path = "associative_point_projection/guards.rs"]
mod guards;
#[path = "associative_point_projection/placement.rs"]
mod placement;
#[path = "associative_point_projection/recipes.rs"]
mod recipes;
