// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn circles_match_go_geometry_diameter_witness_and_intrinsic_rank() {
    let fixtures: Vec<GoRecipe> = serde_json::from_str(include_str!("go-recipes.json")).unwrap();
    let circles: Vec<_> = fixtures
        .into_iter()
        .filter(|f| f.kind == "circle")
        .collect();
    assert_eq!(circles.len(), 9);
    for fixture in circles {
        let recipe = SketchPlacementRecipe::circle(fixture.input[0], fixture.width).unwrap();
        assert_eq!(recipe.points, fixture.points);
        assert_eq!(fixture.dof, 3);
        assert_eq!(super::rectangles::solved_rank(&recipe), fixture.dof);
        verify_entities(&recipe, &fixture);
        verify_relations(&recipe, &fixture);
        verify_fields(&recipe, &fixture);
    }
}
#[test]
fn circles_refuse_invalid_radius_center_and_overflowing_diameter_or_witness() {
    for radius in [0., -1., f64::NAN, f64::INFINITY, f64::MAX] {
        assert!(SketchPlacementRecipe::circle([0., 0.], radius).is_err());
    }
    for center in [[f64::NAN, 0.], [0., f64::INFINITY], [f64::MAX, 0.]] {
        assert!(SketchPlacementRecipe::circle(center, f64::MAX / 2.).is_err());
    }
}
