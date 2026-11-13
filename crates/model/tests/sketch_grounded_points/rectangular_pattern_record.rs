// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    identity::RecipeIdAllocator,
    sketch::{SketchRecipe, SketchRectangularPatternDefinition},
};
use std::collections::BTreeSet;
fn definition() -> SketchRectangularPatternDefinition {
    SketchRectangularPatternDefinition {
        seed: vec![SketchEntityId(2)],
        counts: [3, 2],
        directions: [[-1., 0.], [0., 1.]],
        spacing: [1, 1],
        fitted: false,
        suppressed: BTreeSet::new(),
    }
}
fn circle() -> Sketch2D {
    let mut sketch = geometry(&[(1, [0., 0.])]);
    sketch
        .add_circle(SketchEntityId(2), SketchEntityId(1), 1.)
        .unwrap();
    sketch
}
#[test]
fn rectangular_pattern_retains_grid_and_associations_through_reload() {
    let mut sketch = circle();
    let mut ids = RecipeIdAllocator::restore(Some(200), &[]).unwrap();
    let pattern = sketch
        .create_rectangular_pattern(
            SketchEntityId(100),
            definition(),
            &parameters(),
            &mut || ids.allocate().map(SketchEntityId),
        )
        .unwrap();
    assert_eq!(pattern.members.len(), 5);
    assert_eq!(sketch.circles().len(), 6);
    assert_eq!(sketch.pattern_point_links().len(), 5);
    assert_eq!(sketch.pattern_radius_links().len(), 5);
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 16);
    let json = serde_json::to_vec(&recipe).unwrap();
    let recipe: SketchRecipe = serde_json::from_slice(&json).unwrap();
    let restored = Sketch2D::from_recipe(&recipe).unwrap();
    assert!(restored.same_recipe(&sketch));
    assert_eq!(restored.rectangular_patterns(), &[pattern]);
    assert_eq!(restored.parameter_dependencies(), [1]);
}
#[test]
fn pattern_ownership_prevents_dangling_members_and_explicit_removal_preserves_seed() {
    let mut sketch = circle();
    let mut ids = RecipeIdAllocator::restore(Some(200), &[]).unwrap();
    let pattern = sketch
        .create_rectangular_pattern(
            SketchEntityId(100),
            definition(),
            &parameters(),
            &mut || ids.allocate().map(SketchEntityId),
        )
        .unwrap();
    let before = sketch.clone();
    for id in [
        SketchEntityId(1),
        SketchEntityId(2),
        pattern.members[0].entities[0],
    ] {
        assert!(sketch.delete_entities(&[id]).is_err());
        assert!(sketch.same_recipe(&before));
    }
    let removed = sketch.remove_rectangular_pattern(pattern.id).unwrap();
    assert_eq!(removed.entities.len(), 10);
    assert!(sketch.rectangular_patterns().is_empty());
    assert_eq!(sketch.circles().len(), 1);
    assert_eq!(sketch.circle(SketchEntityId(2)).unwrap().radius, 1.);
    assert!(sketch.pattern_point_links().is_empty());
    assert!(sketch.pattern_radius_links().is_empty());
}
#[test]
fn malformed_records_and_unreserved_id_collision_are_rejected() {
    let mut sketch = circle();
    let mut ids = RecipeIdAllocator::restore(Some(200), &[]).unwrap();
    let pattern = sketch
        .create_rectangular_pattern(
            SketchEntityId(100),
            definition(),
            &parameters(),
            &mut || ids.allocate().map(SketchEntityId),
        )
        .unwrap();
    assert!(sketch.add_point(pattern.id, [0., 0.]).is_err());
    let recipe = sketch.recipe();
    let mut changed_offset = sketch.pattern_offsets()[0].clone();
    changed_offset.terms[0].coefficient[0] += 1.0;
    assert!(sketch.set_pattern_offset(changed_offset).is_err());
    assert!(sketch.same_recipe(&Sketch2D::from_recipe(&recipe).unwrap()));
    let mut invalid = recipe.clone();
    invalid.version = 15;
    assert!(Sketch2D::from_recipe(&invalid).is_err());
    invalid = recipe.clone();
    invalid.rectangular_patterns.as_mut().unwrap()[0].members[0].index = [2, 1];
    assert!(Sketch2D::from_recipe(&invalid).is_err());
    invalid = recipe.clone();
    invalid.pattern_radius_links.as_mut().unwrap().clear();
    assert!(Sketch2D::from_recipe(&invalid).is_err());
    invalid = recipe.clone();
    invalid.pattern_offsets.as_mut().unwrap()[0].terms[0].coefficient[0] += 1.0;
    assert!(Sketch2D::from_recipe(&invalid).is_err());
    invalid = recipe;
    invalid.rectangular_patterns = None;
    assert!(Sketch2D::from_recipe(&invalid).is_err());
}
#[test]
fn singleton_pattern_has_a_persistent_identity_and_parameter_dependencies() {
    let mut sketch = circle();
    let mut ids = RecipeIdAllocator::restore(Some(200), &[]).unwrap();
    let mut spec = definition();
    spec.counts = [1, 1];
    let pattern = sketch
        .create_rectangular_pattern(SketchEntityId(100), spec, &parameters(), &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap();
    assert!(pattern.members.is_empty());
    assert_eq!(ids.high_water(), 200);
    assert_eq!(sketch.parameter_dependencies(), [1]);
    let restored = Sketch2D::from_recipe(&sketch.recipe()).unwrap();
    assert!(restored.same_recipe(&sketch));
    let mut invalid = parameters();
    invalid.set_expression(1, "-1 cm").unwrap();
    assert!(
        restored
            .solve(&invalid, SolveOptions::default(), || false)
            .is_err()
    );
    sketch.remove_rectangular_pattern(pattern.id).unwrap();
    assert_eq!(sketch.circles().len(), 1);
}
