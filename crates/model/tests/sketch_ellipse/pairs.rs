// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{
    PlanarConstraint, SketchAxisOperand, SketchConstraintId, SketchPoint2D,
};
fn line_axes(major_axis: bool, reverse: bool, line: u64) -> [SketchAxisOperand; 2] {
    let mut axes = [
        SketchAxisOperand::Line {
            line: SketchEntityId(line),
        },
        SketchAxisOperand::Ellipse {
            ellipse: SketchEntityId(2),
            major_axis,
        },
    ];
    if reverse {
        axes.reverse();
    }
    axes
}
fn with_relation(
    mut sketch: Sketch2D,
    axes: [SketchAxisOperand; 2],
    perpendicular: bool,
) -> Sketch2D {
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::AxisRelation {
                axes,
                perpendicular,
            },
        )
        .unwrap();
    sketch
}
fn line_pair(major_axis: bool, perpendicular: bool, reverse: bool, scale: f64) -> Sketch2D {
    let mut sketch = owned_ellipse();
    sketch.add_point(SketchEntityId(3), [0.0; 2]).unwrap();
    sketch
        .add_point(SketchEntityId(4), [4.0 * scale, 0.0])
        .unwrap();
    sketch
        .add_line(SketchEntityId(5), [SketchEntityId(3), SketchEntityId(4)])
        .unwrap();
    with_relation(sketch, line_axes(major_axis, reverse, 5), perpendicular)
}
fn boolean_cases() -> Vec<(bool, bool, bool)> {
    [false, true]
        .into_iter()
        .flat_map(|major| {
            [false, true].into_iter().flat_map(move |perpendicular| {
                [false, true].map(|reverse| (major, perpendicular, reverse))
            })
        })
        .collect()
}
#[test]
fn grounded_line_rotates_either_ellipse_axis_in_both_operand_orders_and_scales() {
    for (major, perpendicular, reverse) in boolean_cases() {
        for scale in [1e-6, 1.0, 1e6] {
            check_grounded_pair(
                &line_pair(major, perpendicular, reverse, scale),
                major == perpendicular,
            );
        }
    }
}
fn check_grounded_pair(sketch: &Sketch2D, major_vertical: bool) {
    let anchors = [3, 4].map(|id| *sketch.point(SketchEntityId(id)).unwrap());
    let parameters = ParameterCollection::new(Default::default());
    let outcome = sketch
        .solve_with_grounded_points(&parameters, &anchors, Default::default(), || false)
        .unwrap();
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 4);
    check_fixed_pair_geometry(outcome.candidate(), &anchors, major_vertical);
    assert!(sketch.same_recipe(&Sketch2D::from_recipe(&sketch.recipe()).unwrap()));
}
fn check_fixed_pair_geometry(
    candidate: &Sketch2D,
    anchors: &[SketchPoint2D],
    major_vertical: bool,
) {
    let value = candidate.ellipse(SketchEntityId(2)).unwrap();
    assert!(value.major_axis()[usize::from(!major_vertical)].abs() < 1e-8);
    assert_eq!(value.radii(), [3.0, 1.0]);
    for anchor in anchors {
        assert_eq!(candidate.point(anchor.id).unwrap(), anchor);
    }
}
fn ellipse_pair(major: bool) -> (Sketch2D, [SketchAxisOperand; 2]) {
    let mut sketch = owned_ellipse();
    append_second_ellipse(&mut sketch);
    let axes = [
        SketchAxisOperand::Ellipse {
            ellipse: SketchEntityId(2),
            major_axis: major,
        },
        SketchAxisOperand::Ellipse {
            ellipse: SketchEntityId(4),
            major_axis: !major,
        },
    ];
    (with_relation(sketch, axes, true), axes)
}
#[test]
fn ellipse_pairs_remove_one_freedom_and_preserve_both_axis_selectors() {
    for major in [false, true] {
        let (sketch, axes) = ellipse_pair(major);
        let parameters = ParameterCollection::new(Default::default());
        let outcome = sketch
            .solve(&parameters, Default::default(), || false)
            .unwrap();
        assert_eq!(outcome.numerical().structure.degrees_of_freedom, 9);
        assert!(outcome.numerical().evaluation.values()[0].abs() < 1e-10);
        assert_eq!(
            axes.map(SketchAxisOperand::entity),
            [SketchEntityId(2), SketchEntityId(4)]
        );
        assert!(sketch.same_recipe(&Sketch2D::from_recipe(&sketch.recipe()).unwrap()));
    }
}
#[test]
fn axis_pair_zero_line_direction_is_a_named_error() {
    let sketch = line_pair(true, false, false, 0.0);
    let cause = sketch
        .solve(
            &ParameterCollection::new(Default::default()),
            Default::default(),
            || false,
        )
        .unwrap_err()
        .to_string();
    assert!(cause.contains("nonzero direction"));
}
#[test]
fn axis_pair_refuses_missing_owners_without_adopting_a_constraint() {
    let mut sketch = line_pair(true, false, false, 1.0);
    let cause = sketch
        .add_constraint(
            SketchConstraintId(2),
            PlanarConstraint::AxisRelation {
                axes: line_axes(true, false, 99),
                perpendicular: false,
            },
        )
        .unwrap_err()
        .to_string();
    assert!(cause.contains("99"));
    assert_eq!(sketch.constraints().len(), 1);
}

fn append_second_ellipse(sketch: &mut Sketch2D) {
    sketch.add_point(SketchEntityId(3), [7.0, 8.0]).unwrap();
    sketch
        .add_ellipse(
            SketchEllipse2D::new(SketchEntityId(4), SketchEntityId(3), [1.0, 0.0], [1.0; 2])
                .unwrap(),
        )
        .unwrap();
}

#[test]
fn perpendicular_start_has_a_live_gradient_and_rotates_to_parallel() {
    let mut recipe = line_pair(true, false, false, 1.0).recipe();
    recipe.ellipses.as_mut().unwrap()[0] = ellipse([0.0, 1.0], [3.0, 1.0]);
    let sketch = Sketch2D::from_recipe(&recipe).unwrap();
    check_grounded_pair(&sketch, false);
}

#[path = "collinear.rs"]
mod collinear;
