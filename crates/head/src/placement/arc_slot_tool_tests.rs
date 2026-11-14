// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
fn through_slot(width: f64) -> SketchPlacementTool {
    SketchPlacementTool::ThreePointArcSlot {
        width,
        options: CircumcircleOptions {
            maximum_point_residual: 1e-9,
        },
    }
}
#[test]
fn arc_slot_start_end_through_order_keeps_native_center_and_cap_checks() {
    for through in [[0., 5.], [0., -5.], [6., 5.], [6., -5.]] {
        let ends = [[5., 0.], [-5., 0.]];
        let recipe = through_slot(2.)
            .recipe(&ends, through, &MartensiteSketchConstruction)
            .unwrap();
        let expected = MartensiteSketchConstruction
            .circumcircle(
                [ends[0], through, ends[1]],
                CircumcircleOptions {
                    maximum_point_residual: 1e-9,
                },
            )
            .unwrap();
        assert_eq!(recipe.points[0], expected.center);
        assert_eq!([recipe.points[5], recipe.points[6]], ends);
        assert_eq!(recipe.members.len(), 4);
        let check = recipe.circular_checks[0];
        assert_eq!(check.member, 0);
        assert_eq!(check.cap_members, Some([3, 1]));
        assert_eq!(check.points, [ends[0], through, ends[1]]);
    }
}
#[test]
fn centered_arc_slot_matches_the_shared_go_recipe_in_both_directions() {
    for (end, ccw) in [([0., 10.], true), ([0., -10.], false), ([-10., 0.], false)] {
        let tool = SketchPlacementTool::CenterPointArcSlot { width: 2. };
        let actual = tool
            .recipe(&[[0., 0.], [10., 0.]], end, &MartensiteSketchConstruction)
            .unwrap();
        let expected =
            SketchPlacementRecipe::arc_slot([0., 0.], [[10., 0.], end], 2., ccw).unwrap();
        assert_eq!(actual, expected);
    }
}
#[test]
fn arc_slot_tools_reject_bad_width_bounds_and_collapsed_inner_radius() {
    for width in [0., -1., f64::NAN, f64::INFINITY] {
        assert!(through_slot(width).validate().is_err());
        assert!(
            SketchPlacementTool::CenterPointArcSlot { width }
                .validate()
                .is_err()
        );
    }
    for bound in [-1., f64::NAN, f64::INFINITY] {
        assert!(
            SketchPlacementTool::ThreePointArcSlot {
                width: 2.,
                options: CircumcircleOptions {
                    maximum_point_residual: bound
                }
            }
            .validate()
            .is_err()
        );
    }
    assert!(
        through_slot(10.)
            .recipe(
                &[[5., 0.], [-5., 0.]],
                [0., 5.],
                &MartensiteSketchConstruction
            )
            .unwrap_err()
            .contains("inner radius")
    );
}
