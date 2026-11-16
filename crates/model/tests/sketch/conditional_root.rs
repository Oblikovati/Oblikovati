// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{SketchIntervalTarget, SketchRelationId};
fn inverse() -> Vec<Vec<f64>> {
    vec![
        vec![1., 0., 0., 0., 0., 0., 0., 0.],
        vec![0., 1., 0., 0., 0., 0., 0., 0.],
        vec![1., 0., 0., 0., 0., 0., 1., 0.],
        vec![0., 1., -1., 0., 0., 0., 0., 0.],
        vec![1., 0., 0., 0., -1., 0., 1., 0.],
        vec![0., 1., 0., 1., 0., 0., 0., 1.],
        vec![1., 0., 0., 0., 0., 1., 0., 0.],
        vec![0., 1., 0., 0., 0., 0., 0., 1.],
    ]
}
fn fixed() -> Vec<oblikovati_model::sketch::FixedCoordinateAssumption> {
    (0..2)
        .map(
            |equation| oblikovati_model::sketch::FixedCoordinateAssumption {
                relation: SketchRelationId::Authored(constraint(1)),
                equation,
                value: 0.,
            },
        )
        .collect()
}
fn targets() -> Vec<SketchIntervalTarget> {
    vec![
        SketchIntervalTarget {
            relation: SketchRelationId::Authored(constraint(6)),
            parameter: 1,
            bounds: [5., 5.],
        },
        SketchIntervalTarget {
            relation: SketchRelationId::Authored(constraint(7)),
            parameter: 2,
            bounds: [2.5, 2.5],
        },
    ]
}
fn box_for(outcome: &oblikovati_model::sketch::SketchSolveOutcome) -> Vec<[f64; 2]> {
    outcome
        .numerical()
        .values
        .iter()
        .map(|value| [value - 0.001, value + 0.001])
        .collect()
}
#[test]
fn rectangle_conditionally_encloses_exact_analytic_root() {
    let outcome = rectangle()
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    let bounds = box_for(&outcome);
    let result = outcome
        .conditional_root(&parameters(), &bounds, &targets(), &fixed(), &inverse())
        .unwrap();
    let exact = [0., 0., 5., 0., 5., 2.5, 0., 2.5];
    for (value, interval) in exact.into_iter().zip(result.enclosure()) {
        assert!(interval[0] <= value && value <= interval[1]);
    }
    assert!(result.contraction_upper() < 1.0);
    assert_eq!(result.center(), outcome.numerical().values);
    assert_eq!(result.sketch(), outcome.candidate());
    assert_eq!(result.targets(), targets());
    assert_eq!(result.relations().len(), 8);
    assert_eq!(result.source_rows(), outcome.residual_rows());
    assert_eq!(result.preconditioner(), inverse());
}
#[test]
fn stale_parameters_bad_boxes_and_forged_preconditioners_are_refused() {
    let outcome = rectangle()
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    let bounds = box_for(&outcome);
    let mut stale = parameters();
    stale.set_expression(1, "60 mm").unwrap();
    assert!(
        outcome
            .conditional_root(&stale, &bounds, &targets(), &fixed(), &inverse())
            .unwrap_err()
            .contains("unchanged")
    );
    let mut bad = bounds.clone();
    bad[0] = [
        outcome.numerical().values[0] + 1.,
        outcome.numerical().values[0] + 2.,
    ];
    assert!(
        outcome
            .conditional_root(&parameters(), &bad, &targets(), &fixed(), &inverse())
            .unwrap_err()
            .contains("inside")
    );
    assert!(
        outcome
            .conditional_root(
                &parameters(),
                &bounds,
                &targets(),
                &fixed(),
                &vec![vec![0.; 8]; 8]
            )
            .unwrap_err()
            .contains("contraction")
    );
    let mut wrong = targets();
    wrong[0].parameter = 2;
    assert!(
        outcome
            .conditional_root(&parameters(), &bounds, &wrong, &fixed(), &inverse())
            .is_err()
    );
}
#[test]
fn underdetermined_and_singular_systems_are_refused() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [0., 0.]).unwrap();
    sketch.add_point(entity(2), [1., 0.]).unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Fix {
            point: entity(1),
            position: [0., 0.],
        },
    );
    let outcome = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert!(
        outcome
            .conditional_root(&parameters(), &box_for(&outcome), &[], &fixed(), &[])
            .unwrap_err()
            .contains("square")
    );
    relation(
        &mut sketch,
        2,
        PlanarConstraint::Horizontal {
            points: [entity(1), entity(2)],
        },
    );
    relation(
        &mut sketch,
        3,
        PlanarConstraint::Horizontal {
            points: [entity(1), entity(2)],
        },
    );
    let outcome = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert!(
        outcome
            .conditional_root(
                &parameters(),
                &box_for(&outcome),
                &[],
                &fixed(),
                &vec![vec![0.; 4]; 4]
            )
            .unwrap_err()
            .contains("contraction")
    );
}

#[test]
fn uncertain_targets_are_enclosed_and_overlarge_assumptions_decline() {
    let outcome = rectangle()
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    let bounds = box_for(&outcome);
    let mut input = targets();
    input[0].bounds = [4.9999, 5.0001];
    input[1].bounds = [2.4999, 2.5001];
    let proof = outcome
        .conditional_root(&parameters(), &bounds, &input, &fixed(), &inverse())
        .unwrap();
    for width in input[0].bounds {
        for height in input[1].bounds {
            let root = [0., 0., width, 0., width, height, 0., height];
            for (value, interval) in root.into_iter().zip(proof.enclosure()) {
                assert!(interval[0] <= value && value <= interval[1]);
            }
        }
    }
    input[0].bounds = [4., 6.];
    assert!(
        outcome
            .conditional_root(&parameters(), &bounds, &input, &fixed(), &inverse())
            .unwrap_err()
            .contains("strict interior")
    );
}

#[test]
fn projected_anchor_solve_is_not_misidentified_as_an_unprojected_solve() {
    let sketch = rectangle();
    let anchor = oblikovati_model::sketch::SketchPoint2D {
        id: entity(1),
        position: [0., 0.],
    };
    let outcome = sketch
        .solve_with_grounded_points(&parameters(), &[anchor], SolveOptions::default(), || false)
        .unwrap();
    assert!(
        outcome
            .conditional_root(
                &parameters(),
                &box_for(&outcome),
                &targets(),
                &fixed(),
                &inverse()
            )
            .unwrap_err()
            .contains("unprojected")
    );
    assert!(
        outcome
            .conditional_root_from_sources(&parameters(), &box_for(&outcome), &fixed(), &inverse())
            .is_err()
    );
}
#[test]
fn redundant_equation_system_is_refused_even_if_a_root_exists() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [0., 0.]).unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Fix {
            point: entity(1),
            position: [0., 0.],
        },
    );
    relation(
        &mut sketch,
        2,
        PlanarConstraint::Fix {
            point: entity(1),
            position: [0., 0.],
        },
    );
    let outcome = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert!(
        outcome
            .conditional_root(&parameters(), &box_for(&outcome), &[], &fixed(), &[])
            .unwrap_err()
            .contains("square")
    );
}

#[test]
fn source_bound_root_rejects_a_same_value_but_changed_expression() {
    let mut parameters = parameters();
    parameters.set_expression(2, "width").unwrap();
    let outcome = rectangle()
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    let bounds = box_for(&outcome);
    let root = outcome
        .conditional_root_from_sources(&parameters, &bounds, &fixed(), &inverse())
        .unwrap();
    assert_eq!(root.parameter_bounds().unwrap().len(), 2);
    assert_eq!(root.fixed_assumptions(), fixed());
    assert_eq!(
        root.parameter_bounds().unwrap()[1].sources(),
        &[(2, "width".into()), (1, "50 mm".into())]
    );
    for (value, interval) in [0., 0., 5., 0., 5., 5., 0., 5.]
        .into_iter()
        .zip(root.enclosure())
    {
        assert!(interval[0] <= value && value <= interval[1]);
    }
    parameters.set_expression(1, "5 cm").unwrap();
    assert_eq!(parameters.model_value(1).unwrap().value, 5.0);
    assert!(
        outcome
            .conditional_root_from_sources(&parameters, &bounds, &fixed(), &inverse())
            .unwrap_err()
            .contains("unchanged accepted solve inputs")
    );
}

#[test]
fn source_bound_root_encloses_formula_driven_rectangle() {
    let mut parameters = parameters();
    parameters.set_expression(2, "width / 2").unwrap();
    let outcome = rectangle()
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    let bounds = box_for(&outcome);
    let root = outcome
        .conditional_root_from_sources(&parameters, &bounds, &fixed(), &inverse())
        .unwrap();
    for (exact, interval) in [0., 0., 5., 0., 5., 2.5, 0., 2.5]
        .into_iter()
        .zip(root.enclosure())
    {
        assert!(interval[0] <= exact && exact <= interval[1]);
    }
    assert_eq!(
        root.parameter_bounds().unwrap()[1].sources(),
        &[(2, "width / 2".into()), (1, "50 mm".into())]
    );
    parameters.set_expression(2, "25 mm").unwrap();
    assert!(
        outcome
            .conditional_root_from_sources(&parameters, &bounds, &fixed(), &inverse())
            .is_err()
    );
}

#[test]
fn authored_fixed_coordinates_need_explicit_exact_value_assumptions() {
    let mut source_parameters = parameters();
    source_parameters.set_expression(2, "width").unwrap();
    let outcome = rectangle()
        .solve(&source_parameters, SolveOptions::default(), || false)
        .unwrap();
    let bounds = box_for(&outcome);
    assert!(
        outcome
            .conditional_root_from_sources(&source_parameters, &bounds, &[], &inverse())
            .unwrap_err()
            .contains("exact-value assumption")
    );
    let mut wrong = fixed();
    wrong[0].value = 1.;
    assert!(
        outcome
            .conditional_root_from_sources(&source_parameters, &bounds, &wrong, &inverse())
            .is_err()
    );
    let mut duplicate = fixed();
    duplicate.push(duplicate[0].clone());
    assert!(
        outcome
            .conditional_root_from_sources(&source_parameters, &bounds, &duplicate, &inverse())
            .is_err()
    );
    let root =
        outcome.conditional_root_from_sources(&source_parameters, &bounds, &fixed(), &inverse());
    assert!(root.is_ok());
}
