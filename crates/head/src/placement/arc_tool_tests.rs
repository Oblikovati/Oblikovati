// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
use oblikovati_model::sketch::{SketchPathMember, SketchPlacementGeometry};
fn three_point_arc() -> SketchPlacementTool {
    SketchPlacementTool::ThreePointArc {
        options: CircumcircleOptions {
            maximum_point_residual: 1e-9,
        },
    }
}
#[test]
fn arc_click_order_is_start_end_then_through_with_native_winding() {
    for through in [[0., 5.], [0., -5.], [6., 5.], [6., -5.]] {
        let endpoints = [[5., 0.], [-5., 0.]];
        let recipe = three_point_arc()
            .recipe(&endpoints, through, &MartensiteSketchConstruction)
            .unwrap();
        let inputs = [endpoints[0], through, endpoints[1]];
        let seed = MartensiteSketchConstruction
            .circumcircle(
                inputs,
                CircumcircleOptions {
                    maximum_point_residual: 1e-9,
                },
            )
            .unwrap();
        assert_eq!(recipe.points, vec![seed.center, endpoints[0], endpoints[1]]);
        assert_arc_winding(&recipe, seed.counter_clockwise);
        assert_eq!(recipe.circular_checks.len(), 1);
        assert_eq!(recipe.circular_checks[0].points, inputs);
    }
}
fn assert_arc_winding(recipe: &SketchPlacementRecipe, ccw: bool) {
    assert_eq!(recipe.members.len(), 1);
    assert_eq!(
        recipe.members[0].geometry,
        SketchPlacementGeometry::Path(SketchPathMember::Arc {
            center: 0,
            endpoints: [1, 2],
            counter_clockwise: ccw,
        })
    );
}
#[test]
fn centered_arcs_keep_seed_points_and_choose_clockwise_for_a_half_turn() {
    for (end, ccw) in [([0., 2.], true), ([0., -2.], false), ([-5., 0.], false)] {
        let recipe = SketchPlacementTool::CenterPointArc
            .recipe(&[[0., 0.], [5., 0.]], end, &MartensiteSketchConstruction)
            .unwrap();
        assert_eq!(recipe.points, vec![[0., 0.], [5., 0.], end]);
        assert_arc_winding(&recipe, ccw);
        assert!(recipe.circular_checks.is_empty());
    }
}
#[test]
fn arc_tools_refuse_invalid_accuracy_collinearity_and_zero_sweep() {
    for bound in [-1., f64::NAN, f64::INFINITY] {
        assert!(
            SketchPlacementTool::ThreePointArc {
                options: CircumcircleOptions {
                    maximum_point_residual: bound
                }
            }
            .validate()
            .is_err()
        );
    }
    assert!(
        three_point_arc()
            .recipe(
                &[[0., 0.], [10., 0.]],
                [5., 0.],
                &MartensiteSketchConstruction
            )
            .is_err()
    );
    for end in [[0., 0.], [5., 0.], [10., 0.], [f64::NAN, 1.]] {
        assert!(
            SketchPlacementTool::CenterPointArc
                .recipe(&[[0., 0.], [5., 0.]], end, &MartensiteSketchConstruction)
                .is_err()
        );
    }
}
#[test]
fn a_three_point_arc_asks_for_its_ends_then_a_point_on_it() {
    let prompts = [0, 1, 2].map(|placed| three_point_arc().prompt(placed));
    assert_eq!(
        prompts,
        [
            "Select start of arc",
            "Select end of arc",
            "Select point on arc"
        ]
    );
    assert_eq!(three_point_arc().stage_fields(0), 0..0);
    assert_eq!(three_point_arc().stage_fields(1), 0..2);
    assert_eq!(three_point_arc().stage_fields(2), 0..1);
}
#[test]
fn a_third_point_on_the_chord_gives_no_curve_and_one_off_it_does() {
    let ends = [[0., 0.], [10., 0.]];
    let service = &MartensiteSketchConstruction;
    assert!(three_point_arc().has_no_curve_through(&ends, [5., 0.], service));
    assert!(!three_point_arc().has_no_curve_through(&ends, [5., 2.], service));
    assert!(!SketchPlacementTool::Rectangle.has_no_curve_through(&ends, [5., 0.], service));
}
#[test]
fn a_direction_may_be_zero_where_a_size_may_not() {
    let tool = three_point_arc();
    assert!(tool.typed_value_is_usable(1, 0.));
    assert!(!tool.typed_value_is_usable(0, 0.));
    assert!(tool.typed_value_is_usable(0, 4.));
    assert!(SketchPlacementTool::Rectangle.typed_value_is_usable(0, 0.));
}
