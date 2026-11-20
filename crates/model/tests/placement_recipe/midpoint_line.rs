// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn midpoint_line_retains_its_center_and_full_length_fields() {
    let recipe = SketchPlacementRecipe::midpoint_line([2., -1.], [5., 3.]).unwrap();
    assert_eq!(recipe.points, vec![[-1., -5.], [5., 3.], [2., -1.]]);
    assert_eq!(recipe.members.len(), 1);
    assert_eq!(
        recipe.members[0].geometry,
        SketchPlacementGeometry::Path(SketchPathMember::Line([0, 1]))
    );
    assert_eq!(
        recipe.relations,
        [SketchPlacementRelation::Midpoint { point: 2, line: 0 }]
    );
    assert_eq!(
        recipe.fields[0].binding,
        SketchPlacementBinding::Distance([0, 1])
    );
    assert_eq!(recipe.fields[0].value, 10.);
    assert_eq!(
        recipe.fields[1].binding,
        SketchPlacementBinding::LineAngle(0)
    );
    close(recipe.fields[1].value, 4_f64.atan2(3.));
    assert_eq!(rectangles::solved_rank(&recipe), 4);
}

#[test]
fn midpoint_line_refuses_zero_nonfinite_and_overflowing_segments() {
    for (center, end) in [
        ([0., 0.], [0., 0.]),
        ([f64::NAN, 0.], [1., 2.]),
        ([0., 0.], [f64::INFINITY, 2.]),
        ([f64::MAX, 0.], [-f64::MAX, 0.]),
    ] {
        assert!(SketchPlacementRecipe::midpoint_line(center, end).is_err());
    }
}

#[test]
fn driving_length_edit_keeps_a_fixed_midpoint_and_angle() {
    use oblikovati_model::{
        history::SketchPlacementFieldLock,
        sketch::{PlanarConstraint, SketchConstraintId},
    };
    let recipe = SketchPlacementRecipe::midpoint_line([0., 0.], [3., 4.]).unwrap();
    let locks = [
        Some(SketchPlacementFieldLock {
            reference: false,
            expression: "10 cm".into(),
            text_point: None,
        }),
        Some(SketchPlacementFieldLock {
            reference: false,
            expression: "53.13010235415598 deg".into(),
            text_point: None,
        }),
    ];
    let (history, report) = rectangles::recipe_history(&recipe, &locks);
    let mut geometry = history
        .snapshot()
        .sketches
        .get(report.sketch)
        .unwrap()
        .geometry()
        .clone();
    geometry
        .add_constraint(
            SketchConstraintId(9000),
            PlanarConstraint::Fix {
                point: report.points[2],
                position: [0., 0.],
            },
        )
        .unwrap();
    let mut parameters = history.snapshot().parameters.clone();
    parameters
        .set_expression(report.dimensions[0].1.parameter, "12 cm")
        .unwrap();
    let options = oblikovati_solve::nonlinear::SolveOptions::default();
    let result = geometry.solve(&parameters, options, || false).unwrap();
    assert_eq!(
        result.numerical().termination,
        oblikovati_solve::nonlinear::SolveTermination::Converged
    );
    assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
    let solved = result.candidate();
    for (index, expected) in [(0, [-3.6, -4.8]), (1, [3.6, 4.8]), (2, [0., 0.])] {
        let point = solved.point(report.points[index]).unwrap().position;
        for axis in 0..2 {
            assert!((point[axis] - expected[axis]).abs() <= 12. * options.relative_tolerance);
        }
    }
}
