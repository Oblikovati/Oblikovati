// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
#[test]
fn ellipse_input_uses_perpendicular_distance_and_ignores_side_and_axial_motion() {
    let tool = SketchPlacementTool::Ellipse;
    let accepted = [[3., 4.], [6., 8.]];
    let reference = tool
        .recipe(&accepted, [1.4, 5.2], &MartensiteSketchConstruction)
        .unwrap();
    for cursor in [[1.4, 5.2], [4.6, 2.8], [7.4, 13.2]] {
        let recipe = tool
            .recipe(&accepted, cursor, &MartensiteSketchConstruction)
            .unwrap();
        assert_eq!(recipe.points, reference.points);
        assert!((recipe.fields[0].value - 5.).abs() < 1e-12);
        assert!((recipe.fields[1].value - 2.).abs() < 1e-12);
    }
}
#[test]
fn ellipse_input_refuses_zero_nonfinite_overflow_and_reversed_radii() {
    for points in [
        [[0., 0.], [0., 0.], [0., 1.]],
        [[0., 0.], [5., 0.], [2., 0.]],
        [[0., 0.], [5., 0.], [0., 6.]],
        [[0., 0.], [f64::NAN, 0.], [0., 1.]],
        [[-f64::MAX, 0.], [f64::MAX, 0.], [0., 1.]],
        [[0., 0.], [5., 0.], [0., f64::INFINITY]],
    ] {
        assert!(
            SketchPlacementTool::Ellipse
                .recipe(&points[..2], points[2], &MartensiteSketchConstruction)
                .is_err()
        );
    }
    assert!(
        SketchPlacementTool::Ellipse
            .recipe(
                &[[0., 0.], [5., 0.]],
                [0., 5.],
                &MartensiteSketchConstruction
            )
            .is_ok()
    );
}
