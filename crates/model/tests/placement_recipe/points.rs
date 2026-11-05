// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{Sketch2D, SketchPlacementMember, SketchRecipe};
#[test]
fn points_match_go_geometry_and_rank_without_duplicate_identity() {
    let fixtures: Vec<GoRecipe> = serde_json::from_str(include_str!("go-recipes.json")).unwrap();
    let points: Vec<_> = fixtures.into_iter().filter(|f| f.kind == "point").collect();
    assert_eq!(points.len(), 4);
    for fixture in points {
        let recipe = SketchPlacementRecipe::point(fixture.input[0]).unwrap();
        assert_eq!(recipe.points, fixture.points);
        assert_eq!(fixture.dof, 2);
        assert_eq!(super::rectangles::solved_rank(&recipe), fixture.dof);
        verify_entities(&recipe, &fixture);
        verify_relations(&recipe, &fixture);
        verify_fields(&recipe, &fixture);
        verify_point_history(&recipe);
    }
}
fn verify_point_history(recipe: &SketchPlacementRecipe) {
    let (mut history, report) = super::rectangles::recipe_history(recipe, &[]);
    assert_eq!(report.points, report.members);
    assert_eq!(report.points.len(), 1);
    let geometry = history
        .snapshot()
        .sketches
        .get(report.sketch)
        .unwrap()
        .geometry();
    assert_eq!(geometry.entity_ids(), report.members);
    assert_eq!(geometry.owned_entity_ids(), report.members);
    assert_eq!(
        geometry.point(report.members[0]).unwrap().position,
        recipe.points[0]
    );
    let before = geometry.clone();
    let encoded = serde_json::to_vec(&geometry.recipe()).unwrap();
    let restored: SketchRecipe = serde_json::from_slice(&encoded).unwrap();
    assert!(before.same_recipe(&Sketch2D::from_recipe(&restored).unwrap()));
    history.undo().unwrap();
    assert!(
        history
            .snapshot()
            .sketches
            .get(report.sketch)
            .unwrap()
            .geometry()
            .points()
            .is_empty()
    );
    history.redo().unwrap();
    assert!(
        before.same_recipe(
            history
                .snapshot()
                .sketches
                .get(report.sketch)
                .unwrap()
                .geometry()
        )
    );
}
#[test]
fn point_positions_preserve_finite_bits_and_refuse_nonfinite_inputs() {
    for point in [
        [-0., 0.],
        [f64::MIN_POSITIVE, -f64::MIN_POSITIVE],
        [f64::MAX, -f64::MAX],
    ] {
        let recipe = SketchPlacementRecipe::point(point).unwrap();
        let (history, report) = super::rectangles::recipe_history(&recipe, &[]);
        let actual = history
            .snapshot()
            .sketches
            .get(report.sketch)
            .unwrap()
            .geometry()
            .point(report.members[0])
            .unwrap()
            .position;
        assert_eq!(actual.map(f64::to_bits), point.map(f64::to_bits));
    }
    for point in [[f64::NAN, 0.], [0., f64::INFINITY], [f64::NEG_INFINITY, 0.]] {
        assert!(SketchPlacementRecipe::point(point).is_err());
    }
}
#[test]
fn a_visible_point_can_share_a_curve_endpoint_in_authored_member_order() {
    for point_first in [false, true] {
        let mut recipe = SketchPlacementRecipe::point([3., 4.]).unwrap();
        recipe.points.push([8., 4.]);
        recipe.members.push(SketchPlacementMember {
            geometry: SketchPlacementGeometry::Path(SketchPathMember::Line([0, 1])),
            construction: false,
        });
        if !point_first {
            recipe.members.reverse();
        }
        let (history, report) = super::rectangles::recipe_history(&recipe, &[]);
        let geometry = history
            .snapshot()
            .sketches
            .get(report.sketch)
            .unwrap()
            .geometry();
        assert_eq!(geometry.entity_ids(), report.members);
        assert_eq!(geometry.owned_entity_ids().len(), 3);
        assert_eq!(geometry.points().len(), 2);
        let line = report.members[usize::from(point_first)];
        assert_eq!(
            geometry.line(line).unwrap().endpoints,
            [report.points[0], report.points[1]]
        );
        let restored = Sketch2D::from_recipe(&geometry.recipe()).unwrap();
        assert!(geometry.same_recipe(&restored));
    }
}
#[test]
fn malformed_point_members_are_refused_without_publishing_partial_geometry() {
    let seed = SketchPlacementRecipe::point([3., 4.]).unwrap();
    let (mut history, report) = super::rectangles::recipe_history(&seed, &[]);
    let before = history
        .snapshot()
        .sketches
        .get(report.sketch)
        .unwrap()
        .geometry()
        .clone();
    let mut duplicate = seed.clone();
    duplicate.members.push(duplicate.members[0]);
    let mut bad_index = seed.clone();
    bad_index.members[0].geometry = SketchPlacementGeometry::Point(9);
    let mut construction = seed.clone();
    construction.members[0].construction = true;
    let mut nonfinite = seed.clone();
    nonfinite.points[0][1] = f64::NAN;
    for recipe in [duplicate, bad_index, construction, nonfinite] {
        let token = history.begin("Invalid point").unwrap();
        assert!(
            history
                .edit(&token, |snapshot, ids| snapshot.create_sketch_placement(
                    report.sketch,
                    &recipe,
                    &[],
                    Default::default(),
                    ids
                ))
                .is_err()
        );
        assert!(
            before.same_recipe(
                history
                    .snapshot()
                    .sketches
                    .get(report.sketch)
                    .unwrap()
                    .geometry()
            )
        );
        history.commit(&token, false).unwrap();
    }
}
