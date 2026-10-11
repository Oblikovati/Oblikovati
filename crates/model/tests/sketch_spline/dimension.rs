// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::quantity::Dimension;
fn input() -> SketchDimensionalConstraintCreation {
    SketchDimensionalConstraintCreation::Seeded(SeededDimensionOperands::OffsetSpline {
        offset: id(5),
    })
}
fn dimensioned(seed: f64, expression: &str) -> (Sketch2D, ParameterCollection<u64>) {
    let mut sketch = owned(SketchSplineKind::Control, seed);
    let points = sketch.points().to_vec();
    sketch
        .add_constraint(SketchConstraintId(1), PlanarConstraint::Ground { points })
        .unwrap();
    let mut parameters = ParameterCollection::new(Default::default());
    parameters
        .insert_expression(1, "offset", USER_PARAM, expression)
        .unwrap();
    input()
        .append(&mut sketch, SketchConstraintId(2), 1)
        .unwrap();
    (sketch, parameters)
}
#[test]
fn offset_distance_drives_both_sides_with_one_exact_jacobian_and_zero_freedoms() {
    assert_eq!(input().dimension(), Dimension::Length);
    assert!(input().requires_current_seed());
    for seed in [-2.0, 2.0] {
        assert_signed_solve(seed);
    }
}
fn assert_signed_solve(seed: f64) {
    let (sketch, parameters) = dimensioned(seed, "5 mm");
    let outcome = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 0);
    let row = outcome.numerical().evaluation.jacobian().last().unwrap();
    assert_eq!(&row[..6], &[0.0; 6]);
    assert_eq!(row[6], seed.signum());
    assert_signed_candidate(outcome.candidate(), &sketch, seed.signum() * 0.5);
    assert_eq!(sketch.parameter_dependencies(), [1]);
    assert_eq!(
        sketch.dimension_source(SketchConstraintId(2)).unwrap(),
        (1, Dimension::Length)
    );
}
fn assert_signed_candidate(candidate: &Sketch2D, original: &Sketch2D, expected: f64) {
    assert!((candidate.offset_spline(id(5)).unwrap().distance() - expected).abs() < 1e-8);
    assert_eq!(candidate.points(), original.points());
    assert_eq!(candidate.splines(), original.splines());
    assert_eq!(candidate.offset_spline(id(5)).unwrap().parent(), id(4));
    let measured = candidate.measure_dimension(SketchConstraintId(2)).unwrap();
    assert!((measured.value - expected.abs()).abs() < 1e-8);
}
#[test]
fn source_edits_through_zero_keep_retained_side_and_unit_rank() {
    for seed in [-2.0, 2.0] {
        assert_zero_edit(seed);
    }
}
fn assert_zero_edit(seed: f64) {
    let (sketch, mut parameters) = dimensioned(seed, "0 cm");
    let zero = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert_signed_candidate(zero.candidate(), &sketch, 0.0);
    assert_eq!(zero.numerical().structure.degrees_of_freedom, 0);
    parameters.set_expression(1, "0.25 cm").unwrap();
    let edited = zero
        .candidate()
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert_signed_candidate(edited.candidate(), &sketch, seed.signum() * 0.25);
    assert_eq!(edited.candidate().constraints(), sketch.constraints());
}
#[test]
fn reference_offset_ignores_failed_source_and_measures_current_signed_geometry() {
    let (mut sketch, mut parameters) = dimensioned(-2.0, "1 cm");
    sketch
        .set_dimension_reference(SketchConstraintId(2), true)
        .unwrap();
    parameters.set_expression(1, "missing").unwrap();
    let outcome = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 1);
    assert_signed_candidate(outcome.candidate(), &sketch, -2.0);
    assert!(sketch.parameter_dependencies().is_empty());
}
#[test]
fn invalid_driving_sources_do_not_publish_candidate_offsets() {
    for expression in ["missing", "90 deg", "1 cm / 0", "-1 cm"] {
        let (sketch, parameters) = dimensioned(-2.0, expression);
        let before = sketch.clone();
        assert!(
            sketch
                .solve(&parameters, Default::default(), || false)
                .is_err()
        );
        assert!(sketch.same_recipe(&before));
    }
}
#[test]
fn seed_zero_and_wrong_operand_are_refused_before_appending_relation() {
    let mut sketch = owned(SketchSplineKind::Control, 0.0);
    let before = sketch.clone();
    let error = input()
        .append(&mut sketch, SketchConstraintId(2), 1)
        .unwrap_err()
        .to_string();
    assert!(error.contains("nonzero side"));
    assert!(sketch.same_recipe(&before));
    assert!(
        sketch
            .add_offset_spline_dimension(SketchConstraintId(2), id(4), 1)
            .is_err()
    );
    assert!(sketch.same_recipe(&before));
}
#[test]
fn explicit_zero_seed_side_can_drive_without_selecting_a_new_branch() {
    let mut sketch = owned(SketchSplineKind::Control, 0.0);
    append_explicit_right_side(&mut sketch);
    let mut parameters = ParameterCollection::new(Default::default());
    parameters
        .insert_expression(1, "offset", USER_PARAM, "1 cm")
        .unwrap();
    let outcome = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert_signed_candidate(outcome.candidate(), &sketch, -1.0);
}
#[test]
fn offset_dimension_schema_eleven_retains_branch_and_excludes_old_versions() {
    let (sketch, parameters) = dimensioned(-2.0, "5 mm");
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 11);
    let serialized = serde_json::to_vec(&recipe).unwrap();
    let decoded: SketchRecipe = serde_json::from_slice(&serialized).unwrap();
    let restored = Sketch2D::from_recipe(&decoded).unwrap();
    assert!(restored.same_recipe(&sketch));
    assert_signed_solve_after_restore(&restored, &parameters);
    for version in 1..=10 {
        let mut older = recipe.clone();
        older.version = version;
        assert!(
            Sketch2D::from_recipe(&older)
                .unwrap_err()
                .to_string()
                .contains("expected version 11")
        );
    }
}
fn assert_signed_solve_after_restore(sketch: &Sketch2D, parameters: &ParameterCollection<u64>) {
    let outcome = sketch
        .solve(parameters, Default::default(), || false)
        .unwrap();
    assert_signed_candidate(outcome.candidate(), sketch, -0.5);
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 0);
}
#[test]
fn deletion_restores_one_distance_freedom_and_keeps_the_parent() {
    let (mut sketch, parameters) = dimensioned(-2.0, "5 mm");
    let removed = sketch
        .remove_dimensional_constraint(SketchConstraintId(2))
        .unwrap();
    assert!(matches!(
        removed.relation,
        PlanarConstraint::OffsetSplineDistance { .. }
    ));
    assert_eq!(
        sketch
            .analyze(&parameters, 1e-9)
            .unwrap()
            .degrees_of_freedom,
        1
    );
    assert_eq!(sketch.splines(), &[parent(SketchSplineKind::Control)]);
}

fn append_explicit_right_side(sketch: &mut Sketch2D) {
    sketch
        .add_constraint(
            SketchConstraintId(2),
            PlanarConstraint::OffsetSplineDistance {
                offset: id(5),
                parameter: 1,
                side: LineContactSide::Right,
            },
        )
        .unwrap();
}

#[test]
fn signed_distance_driving_covers_small_and_large_targets_without_changing_parents() {
    for scale in [1e-6, 1.0, 1e6] {
        for sign in [-1.0, 1.0] {
            assert_scaled_distance(scale, sign);
        }
    }
}
fn assert_scaled_distance(scale: f64, sign: f64) {
    let (sketch, parameters) = dimensioned(sign * 2.0 * scale, &format!("{} cm", 0.5 * scale));
    let options = oblikovati_solve::nonlinear::SolveOptions {
        relative_tolerance: 1e-12 * scale.min(1.0),
        ..Default::default()
    };
    let outcome = sketch.solve(&parameters, options, || false).unwrap();
    assert_eq!(
        outcome.numerical().termination,
        oblikovati_solve::nonlinear::SolveTermination::Converged
    );
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 0);
    let actual = outcome.candidate().offset_spline(id(5)).unwrap().distance();
    assert!((actual / (sign * 0.5 * scale) - 1.0).abs() < 1e-8);
    assert_eq!(outcome.candidate().points(), sketch.points());
}
