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
fn pattern_and_spline_handles_share_the_new_recipe_version() {
    use oblikovati_model::sketch::{SketchSpline2D, SketchSplineHandle2D, SketchSplineKind};
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
    for (id, position) in [(10, [0., 0.]), (11, [1., 1.]), (12, [2., 0.])] {
        sketch.add_point(SketchEntityId(id), position).unwrap();
    }
    let spline = SketchSpline2D::new(
        SketchEntityId(13),
        vec![SketchEntityId(10), SketchEntityId(11), SketchEntityId(12)],
        false,
        SketchSplineKind::Fit {
            method: oblikovati_api::types::SPLINE_FIT_SMOOTH,
        },
    )
    .unwrap();
    sketch.add_spline(spline).unwrap();
    sketch
        .add_defining_point(SketchEntityId(14), [3., 1.])
        .unwrap();
    let handle = SketchSplineHandle2D::new(SketchEntityId(13), 1, SketchEntityId(14)).unwrap();
    sketch.add_spline_handle(handle).unwrap();
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 17);
    let restored = Sketch2D::from_recipe(&recipe).unwrap();
    assert!(restored.same_recipe(&sketch));
    assert_eq!(restored.rectangular_patterns(), &[pattern]);
    assert_eq!(restored.spline_handles(), &[handle]);
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

#[test]
fn editing_counts_retains_occurrence_ids_and_reloads_the_new_grid() {
    let mut sketch = circle();
    let mut ids = RecipeIdAllocator::restore(Some(200), &[]).unwrap();
    let old = sketch
        .create_rectangular_pattern(
            SketchEntityId(100),
            definition(),
            &parameters(),
            &mut || ids.allocate().map(SketchEntityId),
        )
        .unwrap();
    let mut changed = definition();
    changed.counts = [4, 2];
    let edit = sketch
        .edit_rectangular_pattern(old.id, changed, &parameters(), &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap();
    assert_eq!(edit.pattern.id, old.id);
    assert_eq!(edit.pattern.members.len(), 7);
    for member in &old.members {
        assert_eq!(
            edit.pattern
                .members
                .iter()
                .find(|item| item.index == member.index)
                .unwrap(),
            member
        );
    }
    assert!(edit.deletion.entities.is_empty());
    let restored = Sketch2D::from_recipe(
        &serde_json::from_slice::<SketchRecipe>(&serde_json::to_vec(&sketch.recipe()).unwrap())
            .unwrap(),
    )
    .unwrap();
    assert!(sketch.same_recipe(&restored));
    assert_eq!(restored.rectangular_patterns()[0].definition.counts, [4, 2]);
}
#[test]
fn fitted_and_suppressed_edits_remove_only_the_affected_occurrences() {
    let mut sketch = circle();
    let mut ids = RecipeIdAllocator::restore(Some(200), &[]).unwrap();
    let old = sketch
        .create_rectangular_pattern(
            SketchEntityId(100),
            definition(),
            &parameters(),
            &mut || ids.allocate().map(SketchEntityId),
        )
        .unwrap();
    let mut changed = definition();
    changed.fitted = true;
    changed.suppressed.insert([1, 1]);
    let edit = sketch
        .edit_rectangular_pattern(old.id, changed, &parameters(), &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap();
    let removed = old
        .members
        .iter()
        .find(|item| item.index == [1, 1])
        .unwrap()
        .entities[0];
    assert!(edit.deletion.entities.contains(&removed));
    assert!(sketch.circle(removed).is_err());
    assert_eq!(edit.pattern.members.len(), 4);
    let solved = sketch
        .solve_with_grounded_points(
            &parameters(),
            &[point(1, [0., 0.])],
            SolveOptions::default(),
            || false,
        )
        .unwrap();
    let end = edit
        .pattern
        .members
        .iter()
        .find(|item| item.index == [2, 0])
        .unwrap()
        .entities[0];
    let center = solved
        .candidate()
        .point(solved.candidate().circle(end).unwrap().center)
        .unwrap()
        .position;
    assert!((center[0] + 3.).abs() < 1e-7 && center[1].abs() < 1e-7);
    assert!(
        Sketch2D::from_recipe(&sketch.recipe())
            .unwrap()
            .same_recipe(&sketch)
    );
}
#[test]
fn failed_pattern_edits_leave_the_original_unchanged() {
    let mut sketch = circle();
    let mut ids = RecipeIdAllocator::restore(Some(200), &[]).unwrap();
    let old = sketch
        .create_rectangular_pattern(
            SketchEntityId(100),
            definition(),
            &parameters(),
            &mut || ids.allocate().map(SketchEntityId),
        )
        .unwrap();
    let before = sketch.clone();
    let mark = ids.high_water();
    let mut invalid = definition();
    invalid.counts[0] = 0;
    assert!(
        sketch
            .edit_rectangular_pattern(old.id, invalid, &parameters(), &mut || ids
                .allocate()
                .map(SketchEntityId))
            .is_err()
    );
    let mut invalid = definition();
    invalid.seed = vec![SketchEntityId(1)];
    assert!(
        sketch
            .edit_rectangular_pattern(old.id, invalid, &parameters(), &mut || ids
                .allocate()
                .map(SketchEntityId))
            .is_err()
    );
    assert!(sketch.same_recipe(&before));
    assert_eq!(ids.high_water(), mark);
}

#[test]
fn reducing_the_grid_reports_constraints_on_deleted_members() {
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
    let removed = pattern
        .members
        .iter()
        .find(|item| item.index == [2, 0])
        .unwrap()
        .entities[0];
    let center = sketch.circle(removed).unwrap().center;
    let position = sketch.point(center).unwrap().position;
    sketch
        .add_constraint(
            SketchConstraintId(7),
            PlanarConstraint::Fix {
                point: center,
                position,
            },
        )
        .unwrap();
    let mut changed = definition();
    changed.counts = [2, 2];
    let edit = sketch
        .edit_rectangular_pattern(pattern.id, changed, &parameters(), &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap();
    assert!(edit.deletion.entities.contains(&removed));
    assert_eq!(edit.deletion.constraints.len(), 1);
    assert_eq!(edit.deletion.constraints[0].id, SketchConstraintId(7));
    assert!(sketch.constraints().is_empty());
}
