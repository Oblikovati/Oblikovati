// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::{SketchDimensionCreationError, SketchReservationMarks};
#[test]
fn malformed_source_consumes_names_and_ids_without_publishing_a_partial_dimension() {
    let mut history = history(snapshot());
    let before = history.snapshot().sketches.recipe();
    for source in ["", "2 *", "d2"] {
        assert!(create(&mut history, &distance_request(source, false)).is_err());
        assert_eq!(history.snapshot().sketches.recipe(), before);
        assert!(history.snapshot().parameters.ids().is_empty());
    }
    let report = create(&mut history, &distance_request("5 cm", false)).unwrap();
    assert_eq!(
        (
            report.parameter,
            report.constraint.0,
            report.parameter_name.as_str()
        ),
        (4, 4, "d3")
    );
}
#[test]
fn invalid_operands_and_annotation_are_refused_before_any_reservation() {
    let mut history = history(snapshot());
    let before = history.sketch_reservations();
    let mut input = distance_request("5 cm", false);
    input.operands =
        SketchDimensionalConstraintCreation::Measured(MeasuredDimensionOperands::Radius {
            curve: entity(8),
        });
    assert!(create(&mut history, &input).is_err());
    input = distance_request("5 cm", false);
    input.text_point = Some([f64::NAN, 2.0]);
    assert!(create(&mut history, &input).is_err());
    assert_eq!(history.sketch_reservations(), before);
    assert_eq!(history.high_water(), 0);
}
#[test]
fn invalid_metric_domains_are_atomic_and_do_not_become_target_quarantines() {
    for (source, expected) in [("0 cm", "strictly positive"), ("-1 cm", "nonnegative")] {
        let mut history = history(snapshot());
        let mut input = distance_request(source, false);
        input.operands =
            SketchDimensionalConstraintCreation::Measured(MeasuredDimensionOperands::Radius {
                curve: entity(6),
            });
        let cause = create(&mut history, &input).unwrap_err();
        assert!(cause.contains(expected), "{cause}");
        assert_empty_creation(&history);
    }
}
fn assert_empty_creation(history: &DocumentEditHistory) {
    assert!(history.snapshot().parameters.ids().is_empty());
    let geometry = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert_eq!(geometry.dimensional_constraints().count(), 0);
}

#[test]
fn permanent_identity_exhaustion_retains_distinct_named_causes() {
    assert_exhaustion("parameter", u64::MAX, None);
    let mut names = history(snapshot()).sketch_reservations();
    names.dimension_names.insert(1, u64::MAX);
    assert_exhaustion("name", 0, Some(names));
    let mut constraints = history(snapshot()).sketch_reservations();
    constraints.constraints.insert(1, u64::MAX);
    assert_exhaustion("constraint", 0, Some(constraints));
}
fn assert_exhaustion(kind: &str, parameter_mark: u64, marks: Option<SketchReservationMarks>) {
    let mut history = DocumentEditHistory::new(
        snapshot(),
        RecipeIdAllocator::restore(Some(parameter_mark), &[]).unwrap(),
    )
    .unwrap();
    if let Some(marks) = marks {
        history = history.with_sketch_reservations(marks).unwrap();
    }
    exercise_exhaustion(&mut history, kind);
}
fn exercise_exhaustion(history: &mut DocumentEditHistory, kind: &str) {
    let token = history.begin("Exhausted").unwrap();
    history
        .edit(&token, |snapshot, ids| -> Result<(), String> {
            let failure = snapshot
                .create_sketch_dimension(
                    DocumentSketchId(1),
                    &distance_request("5 cm", false),
                    Default::default(),
                    ids,
                )
                .unwrap_err();
            assert_exhaustion_cause(kind, &failure);
            Ok(())
        })
        .unwrap();
    history.abort(&token).unwrap();
    assert!(history.snapshot().parameters.ids().is_empty());
}
fn assert_exhaustion_cause(kind: &str, failure: &SketchDimensionCreationError) {
    use std::error::Error;
    let correct = match kind {
        "constraint" => matches!(failure, SketchDimensionCreationError::ConstraintIdentity(_)),
        _ => matches!(failure, SketchDimensionCreationError::Identity(_)),
    };
    assert!(correct, "{kind}: {failure:?}");
    assert!(failure.source().is_some());
    assert!(failure.to_string().contains(&u64::MAX.to_string()));
}
#[test]
fn failed_owned_formulas_retain_health_without_zero_targets_for_every_native_kind() {
    for (relation, expected) in base_cases()
        .into_iter()
        .chain(angle_cases())
        .chain(offset_cases())
    {
        let mut history = history(snapshot());
        let mut input = request(relation, expected);
        input.expression = "futureDimension".into();
        let report = create(&mut history, &input).unwrap();
        assert_eq!(report.unavailable_targets.len(), 1);
        assert_eq!(report.degrees_of_freedom, 10);
        assert!((report.measured.value - expected.value).abs() < 1e-12);
        assert_failed_sketch_health(&history);
    }
}
#[test]
fn incompatible_and_undefined_math_formulas_remain_authored_and_sick() {
    for source in ["2 rad", "1 cm / 0", "1e308 cm * 1e308"] {
        let mut history = history(snapshot());
        let report = create(&mut history, &distance_request(source, false)).unwrap();
        assert_eq!(report.unavailable_targets.len(), 1, "{source}: {report:?}");
        assert_eq!(report.measured, length(5.0));
        assert!(matches!(
            history
                .snapshot()
                .parameters
                .get(report.parameter)
                .unwrap()
                .health,
            ParameterHealth::Failed(_)
        ));
        assert_failed_sketch_health(&history);
    }
}
