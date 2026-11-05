// SPDX-License-Identifier: GPL-2.0-only
use super::recompute::parameters;
use super::*;
use oblikovati_api::types::MODEL_VALUE_UPPER;
use oblikovati_model::document_sketch::{
    PlacedSketch, PlacedSketchHealth, SketchEvaluation, SketchReadinessCause,
};
use std::error::Error;
fn solved(parameters: &ParameterCollection<u64>) -> DocumentSketches {
    document()
        .solve(parameters, SolveOptions::default(), || false)
        .candidate
}
fn first(sketches: &DocumentSketches) -> &PlacedSketch {
    sketches.get(DocumentSketchId(1)).unwrap()
}
fn assert_decline(
    placed: &PlacedSketch,
    parameters: &ParameterCollection<u64>,
    cause: SketchReadinessCause,
) {
    let error = placed.require_fully_constrained(parameters).unwrap_err();
    assert_eq!(error.sketch, placed.id());
    assert_eq!(error.cause, cause);
    let text = error.to_string();
    assert!(text.contains("document sketch") && text.contains("expected"));
    assert_eq!(
        error.source().is_some(),
        matches!(error.cause, SketchReadinessCause::Failed(_))
    );
}
fn assert_verified(placed: &PlacedSketch, parameters: &ParameterCollection<u64>) {
    let diagnostics = placed.require_fully_constrained(parameters).unwrap();
    assert_eq!(diagnostics.termination, SolveTermination::Converged);
    assert_eq!(diagnostics.structure.degrees_of_freedom, 0);
}
fn assert_stale(placed: &PlacedSketch, parameters: &ParameterCollection<u64>) {
    assert_eq!(
        placed.health(parameters),
        PlacedSketchHealth::Stale(vec![1])
    );
    assert_decline(placed, parameters, SketchReadinessCause::Stale(vec![1]));
}
fn assert_current_failure(placed: &PlacedSketch, parameters: &ParameterCollection<u64>) {
    let PlacedSketchHealth::Current(SketchEvaluation::Failed(cause)) = placed.health(parameters)
    else {
        panic!("expected local parameter failure");
    };
    assert_decline(
        placed,
        parameters,
        SketchReadinessCause::Failed(cause.clone()),
    );
}
fn assert_recipe_restores_without_health(
    solved: &DocumentSketches,
    parameters: &ParameterCollection<u64>,
) {
    let restored = DocumentSketches::from_recipe(&solved.recipe()).unwrap();
    assert_eq!(solved, &restored);
    assert!(solved.same_recipe(&restored));
    assert_eq!(
        first(&restored).health(parameters),
        PlacedSketchHealth::Unevaluated
    );
    let encoded = serde_json::to_string(&solved.recipe()).unwrap();
    assert!(!encoded.contains("evaluation") && !encoded.contains("health"));
}
#[test]
fn new_and_restored_recipes_require_fresh_verification_without_persisting_derived_health() {
    let parameters = parameters();
    assert_eq!(
        first(&document()).health(&parameters),
        PlacedSketchHealth::Unevaluated
    );
    assert_decline(
        first(&document()),
        &parameters,
        SketchReadinessCause::Unevaluated,
    );
    let solved = solved(&parameters);
    assert_verified(first(&solved), &parameters);
    assert_recipe_restores_without_health(&solved, &parameters);
}
fn edit_without_changing_target(parameters: &mut ParameterCollection<u64>) {
    parameters.set_expression(1, "50 mm").unwrap();
    parameters
        .insert_expression(2, "other", USER_PARAM, "10 cm")
        .unwrap();
    parameters.rename(1, "span").unwrap();
}
#[test]
fn changed_targets_are_stale_while_equal_values_unrelated_edits_and_rename_stay_current() {
    let mut parameters = parameters();
    let solved = solved(&parameters);
    edit_without_changing_target(&mut parameters);
    assert_verified(first(&solved), &parameters);
    parameters.set_expression(1, "6 cm").unwrap();
    assert_stale(first(&solved), &parameters);
    parameters.set_expression(1, "5 cm").unwrap();
    assert_verified(first(&solved), &parameters);
}
#[test]
fn selected_tolerance_values_invalidate_actual_model_reads() {
    let mut parameters = parameters();
    let original = solved(&parameters);
    parameters.set_symmetric(1, 0.1).unwrap();
    assert_verified(first(&original), &parameters);
    parameters
        .set_model_value_type(1, MODEL_VALUE_UPPER)
        .unwrap();
    assert_stale(first(&original), &parameters);
    assert_verified(first(&solved(&parameters)), &parameters);
}
#[test]
fn upstream_formula_changes_mark_the_dimension_read_stale() {
    let mut parameters = parameters();
    parameters
        .insert_expression(2, "base", USER_PARAM, "3 cm")
        .unwrap();
    parameters.set_expression(1, "base * 2").unwrap();
    let recomputed = solved(&parameters);
    parameters.set_expression(2, "4 cm").unwrap();
    assert_stale(first(&recomputed), &parameters);
}
#[test]
fn failed_parameter_last_good_values_are_not_usable_and_recovery_requires_verification() {
    let mut parameters = parameters();
    let original = solved(&parameters);
    parameters.set_expression(1, "unknown + 1 cm").unwrap();
    assert_stale(first(&original), &parameters);
    let failed = solved(&parameters);
    assert_current_failure(first(&failed), &parameters);
    parameters.set_expression(1, "6 cm").unwrap();
    assert_stale(first(&failed), &parameters);
    assert_verified(first(&solved(&parameters)), &parameters);
}
#[test]
fn missing_parameter_dependencies_remain_tracked_through_creation_and_deletion() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    let failed = solved(&parameters);
    assert_eq!(first(&failed).geometry().parameter_dependencies(), [1]);
    assert_current_failure(first(&failed), &parameters);
    parameters
        .insert_expression(1, "width", USER_PARAM, "5 cm")
        .unwrap();
    assert_stale(first(&failed), &parameters);
    let valid = solved(&parameters);
    parameters.remove(1).unwrap();
    assert_stale(first(&valid), &parameters);
}
#[test]
fn exact_signed_zero_targets_invalidate_even_when_numeric_equality_holds() {
    let mut parameters = parameters();
    parameters.set_expression(1, "0 cm").unwrap();
    let original = solved(&parameters);
    parameters.set_expression(1, "-0 cm").unwrap();
    assert_eq!(
        parameters.model_value(1).unwrap().value.to_bits(),
        (-0.0_f64).to_bits()
    );
    assert_stale(first(&original), &parameters);
}
#[test]
fn geometry_replacement_clears_verification_but_name_and_plane_edits_keep_planar_results() {
    let parameters = parameters();
    let mut solved = solved(&parameters);
    solved.rename(DocumentSketchId(1), "Renamed").unwrap();
    solved
        .replace_plane(DocumentSketchId(1), SketchPlane::xz())
        .unwrap();
    assert_verified(first(&solved), &parameters);
    let geometry = first(&solved).geometry().clone();
    solved
        .replace_geometry(DocumentSketchId(1), geometry)
        .unwrap();
    assert_decline(
        first(&solved),
        &parameters,
        SketchReadinessCause::Unevaluated,
    );
}
#[test]
fn iteration_limit_is_a_named_decline_with_diagnostics() {
    let parameters = parameters();
    let options = SolveOptions {
        max_iterations: 0,
        ..Default::default()
    };
    let limited = document().solve(&parameters, options, || false).candidate;
    assert_decline(
        first(&limited),
        &parameters,
        SketchReadinessCause::Unconverged(SolveTermination::IterationLimit),
    );
}
#[test]
fn unconstrained_geometry_is_refused_with_its_free_variable_count() {
    let parameters = parameters();
    let mut free = Sketch2D::default();
    free.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    let mut sketches = DocumentSketches::default();
    sketches
        .add(DocumentSketchId(1), "Free", SketchPlane::xy(), free)
        .unwrap();
    let free = sketches
        .solve(&parameters, SolveOptions::default(), || false)
        .candidate;
    assert_decline(
        first(&free),
        &parameters,
        SketchReadinessCause::FreeVariables(2),
    );
}
#[test]
fn cancelled_batches_keep_prior_diagnostics_and_stale_inputs() {
    let mut parameters = parameters();
    let original = solved(&parameters);
    parameters.set_expression(1, "8 cm").unwrap();
    let cancelled = original.solve(&parameters, SolveOptions::default(), || true);
    assert!(cancelled.report.cancelled);
    assert!(cancelled.candidate.same_recipe(&original));
    assert_eq!(
        first(&cancelled.candidate).health(&parameters),
        first(&original).health(&parameters)
    );
    parameters.set_expression(1, "5 cm").unwrap();
    assert_verified(first(&cancelled.candidate), &parameters);
}

#[test]
fn redundant_converged_geometry_keeps_its_warning_in_checked_diagnostics() {
    let mut sketches = document();
    let geometry = super::recompute::duplicate_fix([0.0, 0.0]);
    sketches
        .replace_geometry(DocumentSketchId(1), geometry)
        .unwrap();
    let parameters = parameters();
    let solved = sketches
        .solve(&parameters, SolveOptions::default(), || false)
        .candidate;
    let diag = first(&solved)
        .require_fully_constrained(&parameters)
        .unwrap();
    assert!(diag.structure.redundant_equations > 0);
    assert_eq!(
        diag.structure.status,
        oblikovati_solve::diagnostics::ConstraintStatus::OverConstrained
    );
}
