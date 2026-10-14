// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
use oblikovati_model::sketch::SketchPlacementGeometry;
fn circumcircle_tool(bound: f64) -> SketchPlacementTool {
    SketchPlacementTool::ThreePointCircle {
        options: CircumcircleOptions {
            maximum_point_residual: bound,
        },
    }
}
#[test]
fn incomplete_point_sequences_are_refused_before_indexing() {
    for tool in [
        SketchPlacementTool::StraightSlot { width: 2. },
        SketchPlacementTool::Circle,
        circumcircle_tool(0.),
        SketchPlacementTool::Rectangle,
        SketchPlacementTool::CenterRectangle,
        SketchPlacementTool::ThreePointRectangle,
    ] {
        for accepted in [vec![], vec![[0., 0.]; 3]] {
            assert!(
                tool.recipe(&accepted, [1., 2.], &MartensiteSketchConstruction)
                    .unwrap_err()
                    .contains("expected")
            );
        }
    }
}
#[test]
fn three_point_circle_preserves_native_seeds_for_each_point_order() {
    let first = [[0., 0.], [10., 0.], [5., 5.]];
    for points in [
        first,
        [first[2], first[1], first[0]],
        [first[1], first[2], first[0]],
    ] {
        verify_ordered_seed(points);
    }
}
fn verify_ordered_seed(points: [[f64; 2]; 3]) {
    let options = CircumcircleOptions {
        maximum_point_residual: 1e-12,
    };
    let native = MartensiteSketchConstruction
        .circumcircle(points, options)
        .unwrap();
    let recipe = circumcircle_tool(options.maximum_point_residual)
        .recipe(&points[..2], points[2], &MartensiteSketchConstruction)
        .unwrap();
    verify_native_circle_recipe(&recipe, native);
    assert!((native.center[0] - 5.).abs() < 1e-12);
    assert!(native.center[1].abs() < 1e-12);
    assert!((native.radius - 5.).abs() < 1e-12);
}
#[test]
fn caller_accuracy_controls_native_acceptance_without_a_retry() {
    let points = [[0.1, 0.2], [2.3, -1.7], [4.4, 5.6]];
    let construct = |bound| {
        circumcircle_tool(bound).recipe(&points[..2], points[2], &MartensiteSketchConstruction)
    };
    let native = MartensiteSketchConstruction
        .circumcircle(
            points,
            CircumcircleOptions {
                maximum_point_residual: 1e-12,
            },
        )
        .unwrap();
    assert!(native.point_residuals.iter().any(|r| *r > 0.));
    assert!(construct(0.).is_err());
    let recipe = construct(1e-12).unwrap();
    verify_native_circle_recipe(&recipe, native);
}
#[test]
fn degenerate_circle_inputs_do_not_produce_a_recipe() {
    for points in [
        [[0., 0.], [0., 0.], [1., 1.]],
        [[0., 0.], [1., 1.], [2., 2.]],
        [[0., 0.], [1., 0.], [f64::NAN, 2.]],
        [[0., 0.], [1e200, 0.], [0., 1e200]],
        [[0., 0.], [1e-200, 0.], [0., 1e-200]],
    ] {
        assert!(
            circumcircle_tool(1e-9)
                .recipe(&points[..2], points[2], &MartensiteSketchConstruction)
                .is_err()
        );
    }
    assert!(
        SketchPlacementTool::Circle
            .recipe(&[[0., 0.]], [0., 0.], &MartensiteSketchConstruction)
            .is_err()
    );
}
fn verify_native_circle_recipe(
    recipe: &SketchPlacementRecipe,
    native: oblikovati_model::sketch::PlanarCircumcircle,
) {
    assert_eq!(recipe.points, vec![native.center]);
    assert_eq!(
        recipe.members[0].geometry,
        SketchPlacementGeometry::Circle {
            center: 0,
            radius: native.radius
        }
    );
}
