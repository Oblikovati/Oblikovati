// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[path = "../common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_api::types::{MODEL_PARAM, USER_PARAM};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, PlacedSketchHealth, SketchEvaluation},
    history::{
        DocumentEditHistory, DocumentSnapshot, SketchDimensionCreation,
        SketchDimensionCreationReport,
    },
    identity::RecipeIdAllocator,
    parameter::{ParameterFailure, ParameterHealth},
};
fn snapshot() -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Created", SketchPlane::xy(), fixture())
        .unwrap();
    snapshot
}
fn history(snapshot: DocumentSnapshot) -> DocumentEditHistory {
    let mark = snapshot.parameters.ids().iter().copied().max().unwrap_or(0);
    DocumentEditHistory::new(
        snapshot,
        RecipeIdAllocator::restore(Some(mark), &[]).unwrap(),
    )
    .unwrap()
}
fn request(relation: PlanarConstraint, expected: Quantity) -> SketchDimensionCreation {
    let unit = if expected.dimension == Dimension::Angle {
        "rad"
    } else {
        "cm"
    };
    SketchDimensionCreation {
        operands: creation_operands::input(relation),
        expression: format!("{} {unit}", expected.value),
        reference: false,
        text_point: Some([2.0, -3.0]),
    }
}
fn create(
    history: &mut DocumentEditHistory,
    input: &SketchDimensionCreation,
) -> Result<SketchDimensionCreationReport, String> {
    let token = history.begin("Create dimension").unwrap();
    let result = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_dimension(DocumentSketchId(1), input, Default::default(), ids)
        })
        .map_err(|cause| cause.to_string());
    if result.is_ok() {
        history.commit(&token, false).unwrap();
    } else {
        history.abort(&token).unwrap();
    }
    result
}
#[test]
fn owner_creation_preserves_all_native_dimensional_forms_and_options() {
    for (relation, expected) in base_cases()
        .into_iter()
        .chain(angle_cases())
        .chain(offset_cases())
    {
        let mut history = history(snapshot());
        let input = request(relation, expected);
        let report = create(&mut history, &input).unwrap();
        assert_created_report(history.snapshot(), &input, &report, expected);
    }
}
fn assert_created_report(
    snapshot: &DocumentSnapshot,
    input: &SketchDimensionCreation,
    report: &SketchDimensionCreationReport,
    expected: Quantity,
) {
    let geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert_new_dimension_identity(report);
    assert_eq!(report.measured.dimension, expected.dimension);
    assert!(
        (report.measured.value - expected.value).abs() < 1e-12,
        "{report:?}"
    );
    assert_eq!(
        geometry.dimension_text_point(report.constraint).unwrap(),
        input.text_point
    );
    assert_owned_source(snapshot, input, report);
}
fn assert_owned_source(
    snapshot: &DocumentSnapshot,
    input: &SketchDimensionCreation,
    report: &SketchDimensionCreationReport,
) {
    let parameter = snapshot.parameters.get(report.parameter).unwrap();
    assert_eq!(
        (parameter.kind, parameter.source.as_str()),
        (MODEL_PARAM, input.expression.as_str())
    );
    assert_eq!(input.operands.dimension(), report.measured.dimension);
    assert!(report.unavailable_targets.is_empty());
}
fn distance_request(source: &str, reference: bool) -> SketchDimensionCreation {
    let mut input = request(distance_cases().remove(0).0, length(5.0));
    input.expression = source.into();
    input.reference = reference;
    input
}
#[test]
fn creation_retains_live_formula_and_updates_geometry_when_its_source_changes() {
    let mut snapshot = snapshot();
    snapshot
        .parameters
        .insert_expression(1, "width", USER_PARAM, "5 cm")
        .unwrap();
    let mut history = history(snapshot);
    let report = create(&mut history, &distance_request("width", false)).unwrap();
    assert_eq!(report.parameter, 2);
    assert_eq!(history.snapshot().parameters.drivers(2).unwrap(), [1]);
    change_width(&mut history);
    assert_measured(&history, report.constraint, 7.0);
}
fn assert_measured(history: &DocumentEditHistory, id: SketchConstraintId, expected: f64) {
    let geometry = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    let measured = geometry.measure_dimension(id).unwrap();
    assert!((measured.value - expected).abs() < 1e-8, "{measured:?}");
}
#[test]
fn reference_creation_does_not_drive_or_create_a_transient_conflict() {
    let mut history = history(fixed_distance_snapshot());
    let before = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    let report = create(&mut history, &distance_request("999 cm", true)).unwrap();
    assert_eq!(report.measured, length(5.0));
    assert_reference_geometry(&history, &before, &report);
}
fn assert_reference_geometry(
    history: &DocumentEditHistory,
    before: &Sketch2D,
    report: &SketchDimensionCreationReport,
) {
    let geometry = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert_eq!(geometry.points(), before.points());
    assert_eq!(
        geometry
            .analyze(&history.snapshot().parameters, 1e-9)
            .unwrap()
            .degrees_of_freedom,
        report.degrees_of_freedom
    );
    assert!(geometry.parameter_dependencies().is_empty());
}
#[test]
fn unresolved_driving_formula_retains_failed_health_and_available_dof_until_binding() {
    let mut history = history(snapshot());
    let report = create(&mut history, &distance_request("futureWidth", false)).unwrap();
    assert_eq!(report.measured, length(5.0));
    assert_eq!(report.unavailable_targets.len(), 1);
    assert!(matches!(
        history.snapshot().parameters.get(1).unwrap().health,
        ParameterHealth::Failed(ParameterFailure::Unresolved(_))
    ));
    assert_failed_sketch_health(&history);
    bind_future_width(&mut history);
    assert_measured(&history, report.constraint, 8.0);
}
fn bind_future_width(history: &mut DocumentEditHistory) {
    let token = history.begin("Bind width").unwrap();
    history
        .edit(&token, |snapshot, ids| -> Result<(), String> {
            let id = ids.allocate().unwrap();
            snapshot
                .parameters
                .insert_expression(id, "futureWidth", USER_PARAM, "8 cm")
                .unwrap();
            let solved = snapshot
                .solve_one_document_sketch(DocumentSketchId(1), Default::default(), || false)
                .unwrap();
            snapshot.sketches = solved.candidate;
            Ok(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
}
#[test]
fn unresolved_reference_source_does_not_block_its_measurement_or_solve() {
    let mut history = history(snapshot());
    let report = create(&mut history, &distance_request("futureWidth", true)).unwrap();
    assert_eq!(report.measured, length(5.0));
    assert!(report.unavailable_targets.is_empty());
    let placed = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap();
    assert!(matches!(
        placed.health(&history.snapshot().parameters),
        PlacedSketchHealth::Current(SketchEvaluation::Numerical(_))
    ));
}

fn change_width(history: &mut DocumentEditHistory) {
    let token = history.begin("Width").unwrap();
    history
        .edit(&token, |snapshot, _| -> Result<(), String> {
            snapshot.parameters.set_expression(1, "7 cm").unwrap();
            let solved = snapshot
                .solve_one_document_sketch(DocumentSketchId(1), Default::default(), || false)
                .unwrap();
            snapshot.sketches = solved.candidate;
            Ok(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
}

fn assert_failed_sketch_health(history: &DocumentEditHistory) {
    let placed = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap();
    assert!(matches!(
        placed.health(&history.snapshot().parameters),
        PlacedSketchHealth::Current(SketchEvaluation::Failed(_))
    ));
}

fn assert_new_dimension_identity(report: &SketchDimensionCreationReport) {
    assert_eq!(report.constraint, SketchConstraintId(1));
    assert_eq!(
        (
            report.parameter,
            report.parameter_name.as_str(),
            report.dimension_index
        ),
        (1, "d0", 0)
    );
}

#[path = "owner_creation/analysis.rs"]
mod analysis;
#[path = "owner_creation/failures.rs"]
mod failures;

fn fixed_distance_snapshot() -> DocumentSnapshot {
    let mut snapshot = snapshot();
    let mut geometry = fixture();
    for (id, point) in [(20, 1), (21, 2)] {
        let point = entity(point);
        let position = geometry.point(point).unwrap().position;
        geometry
            .add_constraint(
                SketchConstraintId(id),
                PlanarConstraint::Fix { point, position },
            )
            .unwrap();
    }
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), geometry)
        .unwrap();
    snapshot
}
#[path = "owner_creation/seeds.rs"]
mod seeds;
#[path = "owner_creation/units.rs"]
mod units;
