// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketches, SketchPlacement},
    feature::{
        AuthoredFeature, FeatureId, FeatureReadKey,
        work_plane::{WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneOrigin, WorkPlaneReference},
    },
    history::{DocumentSketchPlacementError, DocumentSnapshot},
    parameter::ParameterCollection,
    sketch::{
        DistanceMeasure, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
        SketchPlane,
    },
    units::UnitsOfMeasure,
};
use oblikovati_solve::nonlinear::SolveOptions;
fn geometry() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    let point = SketchEntityId(1);
    geometry.add_point(point, [2.0, 3.0]).unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Fix {
                point,
                position: [2.0, 3.0],
            },
        )
        .unwrap();
    geometry
}
fn authored_snapshot() -> DocumentSnapshot {
    DocumentSnapshot {
        work_geometry: Default::default(),
        sketches: placed_sketches(),
        parameters: parameters(),
        properties: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
    }
}
fn parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "height", USER_PARAM, "4 cm")
        .unwrap();
    parameters
}
fn placed_sketches() -> DocumentSketches {
    let mut sketches = DocumentSketches::default();
    sketches
        .add_on_work_plane(DocumentSketchId(1), "Hosted", FeatureId(2), geometry())
        .unwrap();
    sketches
}
fn snapshot() -> DocumentSnapshot {
    let mut snapshot = authored_snapshot();
    define_plane(&mut snapshot);
    snapshot.sketches = snapshot
        .sketches
        .solve_pending(&snapshot.parameters, SolveOptions::default(), || false)
        .candidate;
    snapshot
}

fn define_plane(snapshot: &mut DocumentSnapshot) {
    let definition = WorkPlaneDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: WorkPlaneDistance::Parameter(1),
    };
    snapshot
        .work_geometry
        .append(AuthoredFeature::work_plane(
            FeatureId(2),
            "Height",
            definition,
        ))
        .unwrap();
}
fn projected(snapshot: &DocumentSnapshot) -> [f64; 3] {
    let view = snapshot
        .require_resolved_sketch(DocumentSketchId(1))
        .unwrap();
    assert_eq!(view.verification.structure.degrees_of_freedom, 0);
    view.plane
        .to_model(view.sketch.geometry().points()[0].position)
        .unwrap()
}
#[test]
fn linked_placement_uses_current_parameters_without_reauthoring_local_geometry() {
    let mut snapshot = snapshot();
    let recipe = snapshot.sketches.recipe();
    assert_eq!(projected(&snapshot), [2.0, 3.0, 4.0]);
    snapshot.parameters.set_expression(1, "7 cm").unwrap();
    assert_eq!(projected(&snapshot), [2.0, 3.0, 7.0]);
    assert_eq!(snapshot.sketches.recipe(), recipe);
    assert!(
        snapshot
            .sketches
            .get(DocumentSketchId(1))
            .unwrap()
            .plane(None)
            .is_err()
    );
}

#[test]
fn normal_flip_changes_hosted_projection_without_changing_sketch_source() {
    let mut snapshot = snapshot();
    let source = snapshot.sketches.recipe();
    snapshot
        .work_geometry
        .flip_plane_normal(FeatureId(2))
        .unwrap();
    assert_eq!(projected(&snapshot), [2.0, -3.0, 4.0]);
    assert_eq!(snapshot.sketches.recipe(), source);
    snapshot.parameters.set_expression(1, "7 cm").unwrap();
    assert_eq!(projected(&snapshot), [2.0, -3.0, 7.0]);
}
fn assert_unavailable(snapshot: &DocumentSnapshot) {
    let error = snapshot
        .require_resolved_sketch(DocumentSketchId(1))
        .unwrap_err();
    assert!(matches!(
        &error,
        DocumentSketchPlacementError::Plane {
            sketch: DocumentSketchId(1),
            ..
        }
    ));
    assert!(error.to_string().contains("FeatureId(2)"));
    assert!(std::error::Error::source(&error).is_some());
}
#[test]
fn failure_suppression_cutoff_and_missing_features_have_no_last_good_frame() {
    let original = snapshot();
    assert_eq!(projected(&original), [2.0, 3.0, 4.0]);
    let mut failed = original.clone();
    failed.parameters.set_expression(1, "missing").unwrap();
    assert_unavailable(&failed);
    let mut suppressed = original.clone();
    suppressed
        .work_geometry
        .set_suppressed(FeatureId(2), true)
        .unwrap();
    assert_unavailable(&suppressed);
    let mut rolled_back = original.clone();
    rolled_back.work_geometry.set_end_of_part(Some(0)).unwrap();
    assert_unavailable(&rolled_back);
    let mut removed = original.clone();
    removed.work_geometry.remove(FeatureId(2)).unwrap();
    assert_unavailable(&removed);
    assert_eq!(projected(&original), [2.0, 3.0, 4.0]);
}
#[test]
fn fixed_placement_is_independent_of_a_failed_work_plane_and_detaches_the_reference() {
    let mut snapshot = snapshot();
    let original = snapshot.sketches.clone();
    snapshot.parameters.set_expression(1, "missing").unwrap();
    snapshot
        .sketches
        .replace_plane(DocumentSketchId(1), SketchPlane::yz())
        .unwrap();
    assert_eq!(projected(&snapshot), [0.0, 2.0, 3.0]);
    assert!(!snapshot.sketches.same_recipe(&original));
    assert_eq!(
        snapshot
            .sketches
            .get(DocumentSketchId(1))
            .unwrap()
            .placement(),
        SketchPlacement::Fixed(SketchPlane::yz())
    );
    assert_eq!(snapshot.sketches.recipe().version, 1);
}
#[test]
fn linked_recipe_saves_only_the_reference_and_restores_exact_authored_state() {
    let snapshot = snapshot();
    let recipe = snapshot.sketches.recipe();
    assert_eq!(recipe.version, 2);
    assert_eq!(recipe.sketches[0].plane_feature, Some(FeatureId(2)));
    assert!(recipe.sketches[0].plane.is_none());
    assert!(recipe.sketches[0].plane_options.is_none());
    let encoded = serde_json::to_string(&recipe).unwrap();
    let recipe = serde_json::from_str(&encoded).unwrap();
    let restored = DocumentSketches::from_recipe(&recipe).unwrap();
    assert!(restored.same_recipe(&snapshot.sketches));
    assert!(
        restored
            .get(DocumentSketchId(1))
            .unwrap()
            .require_fully_constrained(&snapshot.parameters)
            .is_err()
    );
}
#[test]
fn mixed_ambiguous_incomplete_and_legacy_reference_shapes_are_refused() {
    let original = snapshot().sketches.recipe();
    for kind in 0..5 {
        let mut recipe = original.clone();
        match kind {
            0 => recipe.version = 1,
            1 => recipe.sketches[0].plane_feature = Some(FeatureId(0)),
            2 => recipe.sketches[0].plane_feature = None,
            3 => recipe.sketches[0].plane = Some(SketchPlane::xy().recipe()),
            _ => recipe.sketches[0].plane_options = Some(Default::default()),
        }
        let error = DocumentSketches::from_recipe(&recipe).unwrap_err();
        assert!(error.to_string().contains("expected a fixed frame"));
    }
}
#[test]
fn checked_consumer_refuses_missing_and_unverified_geometry() {
    let mut snapshot = snapshot();
    let missing = snapshot
        .require_resolved_sketch(DocumentSketchId(9))
        .unwrap_err();
    assert!(matches!(missing, DocumentSketchPlacementError::Sketch(_)));
    assert!(std::error::Error::source(&missing).is_some());
    assert!(missing.to_string().contains("DocumentSketchId(9)"));
    assert_unverified(&mut snapshot);
}
fn assert_unverified(snapshot: &mut DocumentSnapshot) {
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), Sketch2D::default())
        .unwrap();
    let error = snapshot
        .require_resolved_sketch(DocumentSketchId(1))
        .unwrap_err();
    assert!(matches!(
        error,
        DocumentSketchPlacementError::Verification(_)
    ));
    assert!(std::error::Error::source(&error).is_some());
    assert!(error.to_string().contains("verification absent"));
}
#[test]
fn attachment_errors_leave_authored_state_unchanged() {
    let mut snapshot = snapshot();
    let original = snapshot.sketches.clone();
    for (sketch, feature) in [(1, 0), (99, 2)] {
        assert!(
            snapshot
                .sketches
                .attach_work_plane(DocumentSketchId(sketch), FeatureId(feature))
                .is_err()
        );
        assert!(snapshot.sketches.same_recipe(&original));
    }
    snapshot
        .sketches
        .attach_work_plane(DocumentSketchId(1), FeatureId(3))
        .unwrap();
    assert!(!snapshot.sketches.same_recipe(&original));
}
#[test]
fn creating_directly_on_a_reference_checks_identity_and_geometry_atomically() {
    let mut sketches = DocumentSketches::default();
    assert!(
        sketches
            .add_on_work_plane(DocumentSketchId(1), "Invalid", FeatureId(0), geometry())
            .is_err()
    );
    assert!(sketches.sketches().is_empty());
    sketches
        .add_on_work_plane(DocumentSketchId(1), "Linked", FeatureId(2), geometry())
        .unwrap();
    assert_duplicate_creation_refused(&mut sketches);
}
fn assert_duplicate_creation_refused(sketches: &mut DocumentSketches) {
    let before = sketches.clone();
    assert!(
        sketches
            .add_on_work_plane(DocumentSketchId(1), "Duplicate", FeatureId(3), geometry())
            .is_err()
    );
    assert!(
        sketches
            .add_on_work_plane(
                DocumentSketchId(2),
                "Repeated entity",
                FeatureId(3),
                geometry()
            )
            .is_err()
    );
    assert!(sketches.same_recipe(&before));
}
#[test]
fn checked_consumer_refuses_free_geometry_after_a_converged_solve() {
    let mut snapshot = snapshot();
    let mut free = Sketch2D::default();
    free.add_point(SketchEntityId(1), [2.0, 3.0]).unwrap();
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), free)
        .unwrap();
    snapshot.sketches = snapshot
        .sketches
        .solve_pending(&snapshot.parameters, SolveOptions::default(), || false)
        .candidate;
    assert_free_geometry_refused(&snapshot);
}
fn assert_free_geometry_refused(snapshot: &DocumentSnapshot) {
    let error = snapshot
        .require_resolved_sketch(DocumentSketchId(1))
        .unwrap_err();
    assert!(matches!(
        error,
        DocumentSketchPlacementError::Verification(_)
    ));
    assert!(error.to_string().contains("degrees of freedom 2"));
}
#[test]
fn malformed_work_plane_reads_are_reported_and_fixed_sketches_stay_independent() {
    let mut snapshot = snapshot();
    let definition = *snapshot.work_geometry.features()[0].definition();
    snapshot
        .work_geometry
        .replace(FeatureId(2), definition, Default::default())
        .unwrap();
    let error = snapshot
        .resolve_sketch_plane(DocumentSketchId(1))
        .unwrap_err();
    assert!(matches!(error, DocumentSketchPlacementError::Replay(_)));
    assert!(error.to_string().contains("Parameter(1)"));
    assert!(std::error::Error::source(&error).is_some());
    snapshot
        .sketches
        .replace_plane(DocumentSketchId(1), SketchPlane::xy())
        .unwrap();
    assert_eq!(projected(&snapshot), [2.0, 3.0, 0.0]);
}
fn dimension_geometry() -> Sketch2D {
    let mut geometry = geometry();
    geometry.add_point(SketchEntityId(2), [3.0, 3.0]).unwrap();
    let relation = PlanarConstraint::Distance {
        points: [SketchEntityId(1), SketchEntityId(2)],
        measure: DistanceMeasure::Horizontal,
        parameter: 1,
    };
    geometry
        .add_constraint(SketchConstraintId(2), relation)
        .unwrap();
    geometry
}
fn footprint_keys(snapshot: &DocumentSnapshot) -> Vec<FeatureReadKey> {
    let reads = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .read_footprint();
    assert!(!reads.has_unattributed_reads());
    reads.keys().iter().copied().collect()
}
#[test]
fn sketch_footprint_attributes_dimensions_and_the_host_in_separate_namespaces() {
    let mut snapshot = snapshot();
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), dimension_geometry())
        .unwrap();
    let expected = [
        FeatureReadKey::Feature(FeatureId(2)),
        FeatureReadKey::Parameter(1),
    ];
    assert_eq!(footprint_keys(&snapshot), expected);
    snapshot
        .sketches
        .replace_plane(DocumentSketchId(1), SketchPlane::xy())
        .unwrap();
    assert_eq!(footprint_keys(&snapshot), [FeatureReadKey::Parameter(1)]);
}
