// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn source_failure_and_recovery_are_local_and_never_return_last_good_geometry_as_verified() {
    let mut snapshot = snapshot();
    recompute(&mut snapshot);
    let before = position(&snapshot);
    snapshot.parameters.set_expression(1, "missing").unwrap();
    let outcome = snapshot.solve_document_sketches(SolveOptions::default(), || false);
    assert_source_failure(&outcome.report.reports[0].result);
    snapshot.sketches = outcome.candidate;
    assert_eq!(position(&snapshot), before);
    assert!(snapshot.require_resolved_sketch(SKETCH).is_err());
    assert_eq!(recompute(&mut snapshot), 0);
    snapshot.parameters.set_expression(1, "6 cm").unwrap();
    assert_eq!(recompute(&mut snapshot), 1);
    assert_ready(&snapshot, [2.0, 6.0]);
}
fn assert_source_failure(
    result: &Result<oblikovati_model::sketch::SketchSolveOutcome, SketchError>,
) {
    let error = result.as_ref().unwrap_err();
    assert!(matches!(
        error,
        SketchError::ProjectionSource { point: ANCHOR, .. }
    ));
    assert!(error.to_string().contains("FeatureId(1)"));
    assert!(
        error
            .source()
            .unwrap()
            .downcast_ref::<DocumentSketchPlacementError>()
            .is_some()
    );
}
#[test]
fn suppressed_source_does_not_stop_an_independent_sketch_and_selected_solve_is_local() {
    let mut snapshot = snapshot();
    add_independent(&mut snapshot);
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(1), true)
        .unwrap();
    let outcome = snapshot.solve_document_sketches(SolveOptions::default(), || false);
    assert_source_failure(&outcome.report.reports[0].result);
    assert!(outcome.report.reports[1].result.is_ok());
    let selected = snapshot
        .solve_one_document_sketch(DocumentSketchId(2), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(selected.report.reports.len(), 1);
    assert_eq!(selected.report.reports[0].sketch, DocumentSketchId(2));
}
#[test]
fn unknown_selected_sketch_is_declined_before_solving() {
    let snapshot = snapshot();
    assert!(
        snapshot
            .solve_one_document_sketch(DocumentSketchId(99), SolveOptions::default(), || false)
            .is_err()
    );
}
fn add_independent(snapshot: &mut DocumentSnapshot) {
    snapshot
        .sketches
        .add(
            DocumentSketchId(2),
            "Independent",
            SketchPlane::xy(),
            Sketch2D::default(),
        )
        .unwrap();
}
#[test]
fn cancellation_discards_grounded_coordinates_and_projection_verification_as_one_batch() {
    let snapshot = snapshot();
    let outcome = snapshot.solve_document_sketches(SolveOptions::default(), || true);
    assert!(outcome.report.cancelled);
    assert!(outcome.candidate.same_evaluated_state(&snapshot.sketches));
}
#[test]
fn analysis_reads_current_projected_sources_and_refuses_source_loss() {
    let mut snapshot = snapshot();
    let before = snapshot.sketches.clone();
    let structure = snapshot.analyze_document_sketch(SKETCH, 1e-9).unwrap();
    assert_eq!(structure.degrees_of_freedom, 0);
    assert_eq!(structure.rank, 0);
    assert!(before.same_evaluated_state(&snapshot.sketches));
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(1), true)
        .unwrap();
    assert!(matches!(
        snapshot.analyze_document_sketch(SKETCH, 1e-9),
        Err(SketchError::ProjectionSource { .. })
    ));
}
#[test]
fn same_bindings_are_a_true_noop() {
    let mut snapshot = snapshot();
    recompute(&mut snapshot);
    let before = snapshot.sketches.clone();
    snapshot
        .sketches
        .set_point_projections(SKETCH, vec![binding()])
        .unwrap();
    assert!(before.same_evaluated_state(&snapshot.sketches));
}
#[test]
fn source_redefinition_requires_a_new_solve_and_refreshes_dependencies() {
    let mut snapshot = snapshot();
    recompute(&mut snapshot);
    let origin = SketchPointProjection {
        anchor: ANCHOR,
        reference: WorkPointReference::Origin,
    };
    snapshot
        .sketches
        .set_point_projections(SKETCH, vec![origin])
        .unwrap();
    assert!(snapshot.require_resolved_sketch(SKETCH).is_err());
    recompute(&mut snapshot);
    assert_ready(&snapshot, [0.0; 2]);
    let reads = snapshot.sketches.get(SKETCH).unwrap().read_footprint();
    assert!(reads.keys().is_empty());
}
#[test]
fn removing_a_bound_anchor_or_malformed_bindings_is_refused_atomically() {
    let mut snapshot = snapshot();
    let before = snapshot.sketches.clone();
    for bindings in invalid_bindings() {
        assert!(
            snapshot
                .sketches
                .set_point_projections(SKETCH, bindings)
                .is_err()
        );
        assert!(before.same_evaluated_state(&snapshot.sketches));
    }
}
fn invalid_bindings() -> [Vec<SketchPointProjection>; 3] {
    let missing = SketchPointProjection {
        anchor: SketchEntityId(99),
        ..binding()
    };
    let zero = SketchPointProjection {
        reference: WorkPointReference::Feature(FeatureId(0)),
        ..binding()
    };
    [vec![binding(); 2], vec![missing], vec![zero]]
}
#[test]
fn removal_of_a_bound_anchor_is_refused_atomically() {
    let mut snapshot = snapshot();
    let before = snapshot.sketches.clone();
    assert!(
        snapshot
            .sketches
            .replace_geometry(SKETCH, Sketch2D::default())
            .is_err()
    );
    assert!(before.same_evaluated_state(&snapshot.sketches));
}
#[test]
fn structural_source_replay_failure_is_retained_in_the_projection_report() {
    let mut snapshot = snapshot();
    let definition = source_definition();
    let feature = AuthoredFeature::new(FeatureId(1), "Undeclared", definition, Default::default());
    snapshot.work_geometry = Default::default();
    snapshot.work_geometry.append(feature).unwrap();
    let outcome = snapshot.solve_document_sketches(SolveOptions::default(), || false);
    let error = outcome.report.reports[0].result.as_ref().unwrap_err();
    assert!(matches!(
        error
            .source()
            .unwrap()
            .downcast_ref::<DocumentSketchPlacementError>()
            .unwrap(),
        DocumentSketchPlacementError::Replay(_)
    ));
}
#[test]
fn deleting_the_last_projection_consumer_prunes_its_construction_source() {
    let mut snapshot = snapshot();
    snapshot
        .work_geometry
        .set_work_plane_construction(FeatureId(1), true)
        .unwrap();
    let report = snapshot.delete_sketch(SKETCH).unwrap();
    assert_eq!(report.construction_datums, [FeatureId(1)]);
    assert!(snapshot.work_geometry.get(FeatureId(1)).is_err());
    assert!(snapshot.parameters.get(1).is_some());
}
fn add_second_anchor(snapshot: &mut DocumentSnapshot) {
    let mut geometry = geometry();
    geometry.add_point(SketchEntityId(12), [99.0; 2]).unwrap();
    snapshot
        .sketches
        .replace_geometry(SKETCH, geometry)
        .unwrap();
}
#[test]
fn a_repeated_source_with_a_distinct_anchor_is_refused_atomically() {
    let mut snapshot = snapshot();
    add_second_anchor(&mut snapshot);
    let before = snapshot.sketches.clone();
    let repeated = SketchPointProjection {
        anchor: SketchEntityId(12),
        ..binding()
    };
    let error = snapshot
        .sketches
        .set_point_projections(SKETCH, vec![binding(), repeated])
        .unwrap_err();
    assert!(error.to_string().contains("unique anchor and source"));
    assert!(before.same_evaluated_state(&snapshot.sketches));
}
#[test]
fn distinct_sources_at_equal_coordinates_keep_both_anchor_identities() {
    let mut snapshot = snapshot();
    add_second_anchor(&mut snapshot);
    append_second_source(&mut snapshot);
    let second = SketchPointProjection {
        anchor: SketchEntityId(12),
        reference: WorkPointReference::Feature(FeatureId(2)),
    };
    snapshot
        .sketches
        .set_point_projections(SKETCH, vec![binding(), second])
        .unwrap();
    recompute(&mut snapshot);
    assert_ready(&snapshot, [2.0, 4.0]);
    let placed = snapshot.sketches.get(SKETCH).unwrap();
    assert_second_anchor_position(placed, second.anchor);
    assert_eq!(placed.point_projections(), [binding(), second]);
}

fn append_second_source(snapshot: &mut DocumentSnapshot) {
    let definition = source_definition();
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(
            FeatureId(2),
            "Second source",
            definition,
        ))
        .unwrap();
}

fn assert_second_anchor_position(placed: &PlacedSketch, anchor: SketchEntityId) {
    assert_eq!(
        placed.geometry().point(anchor).unwrap().position,
        [2.0, 4.0]
    );
}
