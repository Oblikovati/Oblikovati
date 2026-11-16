// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::prism_test_fixture as fixture;
use oblikovati_api::wire::TransactionEventPayload;
use oblikovati_model::{
    document_sketch::{DocumentSketches, SketchPointProjection},
    feature::{
        sketch_prism::SketchPrismDefinition,
        work_geometry::{DatumLength, WorkPointReference},
        work_plane::WorkPlaneOrigin,
    },
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::RecipeIdAllocator,
    sketch::{SketchConstraintId, SketchEntityId},
};
use oblikovati_session::DocumentSession;

struct RecordingObserver;
impl TransactionObserver for RecordingObserver {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {}
    fn changed(&mut self, _: u64, _: u64) {}
}
fn document() -> DocumentSnapshot {
    let mut document = fixture::selected_sketch_document();
    let mut geometry = document
        .sketches
        .get(fixture::SKETCH)
        .unwrap()
        .geometry()
        .clone();
    geometry
        .remove_geometric_constraint(SketchConstraintId(1))
        .unwrap();
    let mut sketches = DocumentSketches::default();
    sketches
        .add_on_origin_plane(fixture::SKETCH, "Rectangle", WorkPlaneOrigin::XY, geometry)
        .unwrap();
    sketches
        .set_point_projections(
            fixture::SKETCH,
            vec![SketchPointProjection {
                anchor: SketchEntityId(1),
                reference: WorkPointReference::Origin,
            }],
        )
        .unwrap();
    document.sketches = sketches;
    document.sketches = document
        .solve_one_document_sketch(fixture::SKETCH, Default::default(), || false)
        .unwrap()
        .candidate;
    document
}
fn session() -> (DocumentSession<RecordingObserver>, FeatureId) {
    session_with_document(document())
}
fn session_with_document(
    document: DocumentSnapshot,
) -> (DocumentSession<RecordingObserver>, FeatureId) {
    let history = DocumentEditHistory::new(
        document,
        RecipeIdAllocator::restore(Some(2), &[1, 2]).unwrap(),
    )
    .unwrap();
    let mut session = DocumentSession::new(1, history, RecordingObserver);
    let definition = SketchPrismDefinition::new(
        fixture::SKETCH,
        fixture::selected_sketch_boundary(),
        [
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
            DatumLength::Parameter(1),
        ],
    )
    .unwrap();
    let feature = session.create_sketch_prism("Prism", definition).unwrap();
    (session, feature)
}
fn sketch_box(session: &DocumentSession<RecordingObserver>) -> Vec<[f64; 2]> {
    session
        .snapshot()
        .sketches
        .get(fixture::SKETCH)
        .unwrap()
        .geometry()
        .points()
        .iter()
        .flat_map(|point| point.position)
        .map(|value| [value - 0.01, value + 0.01])
        .collect()
}
fn check(session: &DocumentSession<RecordingObserver>, feature: FeatureId, width: f64) {
    let evidence = session
        .conditional_canonical_sketch(fixture::SKETCH, SketchEntityId(1))
        .unwrap();
    let result = conditional_prism_inputs(
        &evidence,
        feature,
        &sketch_box(session),
        [Some([0., 0.]), Some([0., 0.]), None],
    )
    .unwrap();
    assert_eq!(result.feature(), feature);
    assert_eq!(
        result
            .lines()
            .iter()
            .map(|line| line.curve.0)
            .collect::<Vec<_>>(),
        [101, 102, 103, 104]
    );
    assert_eq!(
        result
            .vertices()
            .iter()
            .map(|vertex| vertex.vertex.0)
            .collect::<Vec<_>>(),
        [1, 2, 3, 4]
    );
    assert_eq!(
        result
            .swept_points()
            .iter()
            .map(|point| point.point.0)
            .collect::<Vec<_>>(),
        [1, 2, 3, 4]
    );
    assert!(result.axes()[2].bounds()[0] <= width && width <= result.axes()[2].bounds()[1]);
    assert_eq!(result.axes()[2].parameter().unwrap().id(), 1);
    assert!(!result.root().source_rows().is_empty());
    use PrismPrecisionObligation::*;
    assert_eq!(
        result.obligations(),
        [
            ArithmeticPlatform,
            ExtrusionSource,
            ProfileIdentityMapping,
            ExactCoincidenceAdmission,
            OperandToleranceBudget,
            AdmissionCoverage
        ]
    );
}
#[test]
fn one_session_root_drives_the_complete_conditional_input_report() {
    let (mut session, feature) = session();
    let generation = session.generation();
    check(&session, feature, 4.);
    assert_eq!(session.generation(), generation);
    session
        .run("Change width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "6 cm")
        })
        .unwrap();
    check(&session, feature, 6.);
    session.undo().unwrap();
    check(&session, feature, 4.);
    session.redo().unwrap();
    check(&session, feature, 6.);
}
#[test]
fn conditional_report_declines_wrong_feature_and_unproven_constants() {
    let (session, feature) = session();
    let evidence = session
        .conditional_canonical_sketch(fixture::SKETCH, SketchEntityId(1))
        .unwrap();
    let bounds = sketch_box(&session);
    assert!(
        conditional_prism_inputs(
            &evidence,
            FeatureId(u64::MAX),
            &bounds,
            [Some([0., 0.]), Some([0., 0.]), None]
        )
        .is_err()
    );
    assert!(
        conditional_prism_inputs(&evidence, feature, &bounds, [None, Some([0., 0.]), None])
            .is_err()
    );
}

#[test]
fn selected_construction_line_is_not_a_converted_profile_boundary() {
    let mut document = document();
    let mut geometry = document
        .sketches
        .get(fixture::SKETCH)
        .unwrap()
        .geometry()
        .clone();
    geometry
        .set_curve_construction(&[SketchEntityId(101)], true)
        .unwrap();
    document
        .sketches
        .replace_geometry(fixture::SKETCH, geometry)
        .unwrap();
    document.sketches = document
        .solve_one_document_sketch(fixture::SKETCH, Default::default(), || false)
        .unwrap()
        .candidate;
    let (session, feature) = session_with_document(document);
    let evidence = session
        .conditional_canonical_sketch(fixture::SKETCH, SketchEntityId(1))
        .unwrap();
    let error = conditional_prism_inputs(
        &evidence,
        feature,
        &sketch_box(&session),
        [Some([0., 0.]), Some([0., 0.]), None],
    )
    .err()
    .unwrap();
    assert!(
        error.contains("expected one distinct converted profile line"),
        "{error}"
    );
}
