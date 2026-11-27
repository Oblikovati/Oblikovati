// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn fixed_coordinates_keep_bits_full_jacobians_and_other_variables_solve_normally() {
    use oblikovati_solve::nonlinear::solve_constraints_with_constants;
    for coordinate in [-0.0_f64, 2.0, 1e-6] {
        let target = Target {
            target: coordinate,
            coefficient: 1.0,
        };
        let blocks = [block(&target, &[0]), block(&Gap, &[0, 1])];
        let input = [coordinate + 0.5, 9.0];
        let result = solve_constraints_with_constants(
            &input,
            &blocks,
            &[(0, coordinate)],
            Default::default(),
            || false,
        )
        .unwrap();
        assert_eq!(result.termination, SolveTermination::Converged);
        assert_eq!(result.values[0].to_bits(), coordinate.to_bits());
        assert_close(result.values[1], coordinate + 2.0, 1e-8);
        assert_eq!(result.evaluation.jacobian(), &[vec![1., 0.], vec![-1., 1.]]);
        assert_eq!(result.structure.degrees_of_freedom, 0);
        assert_eq!(input, [coordinate + 0.5, 9.0]);
    }
}
#[test]
fn incompatible_constant_equations_report_conflicts_without_moving_the_constant() {
    use oblikovati_solve::nonlinear::solve_constraints_with_constants;
    let target = Target {
        target: 2.,
        coefficient: 1.,
    };
    let result = solve_constraints_with_constants(
        &[2.],
        &[block(&target, &[0])],
        &[(0, 3.)],
        Default::default(),
        || false,
    )
    .unwrap();
    assert_eq!(result.values, [3.]);
    assert_eq!(result.termination, SolveTermination::Stalled);
    assert_eq!(result.conflicts, [0]);
    assert_eq!(result.evaluation.jacobian(), &[vec![1.]]);
}

#[test]
fn cancelled_constant_solves_return_the_unaccepted_warm_start_unchanged() {
    use oblikovati_solve::nonlinear::solve_constraints_with_constants;
    let target = Target {
        target: 0.,
        coefficient: 1.,
    };
    let values = [2.];
    let blocks = [block(&target, &[0])];
    let result =
        solve_constraints_with_constants(&values, &blocks, &[(0, -0.)], Default::default(), || {
            true
        })
        .unwrap();
    assert_eq!(result.termination, SolveTermination::Cancelled);
    assert_eq!(result.iterations, 0);
    assert_eq!(result.values, values);
    assert_eq!(
        result.evaluation,
        evaluate_residuals(&values, &blocks).unwrap()
    );
}
