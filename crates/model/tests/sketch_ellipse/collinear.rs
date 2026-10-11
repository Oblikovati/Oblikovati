// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_solve::nonlinear::SolveTermination;
fn collinear_sketch(major: bool, reverse: bool, scale: f64) -> Sketch2D {
    let mut recipe = line_pair(major, false, reverse, scale).recipe();
    recipe.constraints.clear();
    let mut sketch = Sketch2D::from_recipe(&recipe).unwrap();
    install_collinear(&mut sketch, major, reverse);

    sketch
}
#[test]
fn collinear_line_passes_through_grounded_ellipse_center_for_both_axes_and_orders() {
    for (major, _, reverse) in boolean_cases()
        .into_iter()
        .filter(|(_, perpendicular, _)| !perpendicular)
    {
        for scale in [1e-6, 1.0, 1e6] {
            check_collinear_solution(&collinear_sketch(major, reverse, scale), major, scale);
        }
    }
}
fn check_collinear_solution(sketch: &Sketch2D, major: bool, scale: f64) {
    let center = *sketch.point(SketchEntityId(1)).unwrap();
    let parameters = ParameterCollection::new(Default::default());
    let outcome = sketch
        .solve_with_grounded_points(&parameters, &[center], Default::default(), || false)
        .unwrap();
    assert_eq!(
        outcome.numerical().termination,
        SolveTermination::Converged,
        "major={major} scale={scale} sketch={sketch:?} outcome={:?}",
        outcome.numerical()
    );
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 4);
    assert_eq!(outcome.candidate().point(center.id).unwrap(), &center);
    check_collinear_geometry(outcome.candidate(), major, scale);
    assert!(sketch.same_recipe(&Sketch2D::from_recipe(&sketch.recipe()).unwrap()));
}
fn check_collinear_geometry(candidate: &Sketch2D, major: bool, scale: f64) {
    let ellipse = candidate.ellipse(SketchEntityId(2)).unwrap();
    assert_eq!(ellipse.radii(), [3.0, 1.0]);
    let center = candidate.point(ellipse.center()).unwrap().position;
    for id in candidate.line(SketchEntityId(5)).unwrap().endpoints {
        let position = candidate.point(id).unwrap().position;
        assert!(
            (position[usize::from(major)] - center[usize::from(major)]).abs() / scale.max(1.0)
                < 1e-8
        );
    }
}
#[test]
fn collinear_ellipse_pairs_have_two_independent_equations_and_round_trip_anchors() {
    let (sketch, axes) = ellipse_pair(true);
    let mut recipe = sketch.recipe();
    recipe.constraints.clear();
    let mut sketch = Sketch2D::from_recipe(&recipe).unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::AxisCollinear { axes },
        )
        .unwrap();
    let parameters = ParameterCollection::new(Default::default());
    let outcome = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 8);
    assert_eq!(outcome.numerical().termination, SolveTermination::Converged);
    assert_eq!(outcome.numerical().evaluation.values().len(), 2);
    assert!(sketch.same_recipe(&Sketch2D::from_recipe(&sketch.recipe()).unwrap()));
}

fn install_collinear(sketch: &mut Sketch2D, major: bool, reverse: bool) {
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::AxisCollinear {
                axes: line_axes(major, reverse, 5),
            },
        )
        .unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(2),
            PlanarConstraint::EllipseAxisAlignment {
                ellipse: SketchEntityId(2),
                major_axis: true,
                vertical: false,
            },
        )
        .unwrap();
}
