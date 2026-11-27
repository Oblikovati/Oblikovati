// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{SketchPatternPointLink, SketchRecipe};
fn link(source: u64, member: u64, offset: [f64; 2]) -> SketchPatternPointLink {
    SketchPatternPointLink {
        source: SketchEntityId(source),
        member: SketchEntityId(member),
        offset,
    }
}
#[test]
fn copied_points_follow_the_grounded_seed_with_no_extra_freedom_after_reload() {
    let mut sketch = geometry(&[(1, [0., 0.]), (2, [10., 5.]), (3, [20., 5.])]);
    sketch
        .add_pattern_point_link(link(1, 2, [10., 5.]))
        .unwrap();
    sketch
        .add_pattern_point_link(link(1, 3, [20., 5.]))
        .unwrap();
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 14);
    let json = serde_json::to_vec(&recipe).unwrap();
    let recipe: SketchRecipe = serde_json::from_slice(&json).unwrap();
    let restored = Sketch2D::from_recipe(&recipe).unwrap();
    assert!(sketch.same_recipe(&restored));
    let result = solve(&restored, &[point(1, [4., -3.])]);
    assert_solved(&result, 4);
    for (id, expected) in [(2, [14., 2.]), (3, [24., 2.])] {
        let actual = result
            .candidate()
            .point(SketchEntityId(id))
            .unwrap()
            .position;
        assert!(
            (actual[0] - expected[0]).abs() < 1e-7 && (actual[1] - expected[1]).abs() < 1e-7,
            "{actual:?}"
        );
    }
    assert!(result.conflicting_constraints().is_empty());
}
#[test]
fn conflicting_pattern_links_report_the_member_not_an_authored_constraint() {
    let mut sketch = geometry(&[(1, [0., 0.]), (2, [10., 5.])]);
    sketch
        .add_pattern_point_link(link(1, 2, [10., 5.]))
        .unwrap();
    let result = solve(&sketch, &[point(1, [0., 0.]), point(2, [11., 5.])]);
    assert_ne!(result.numerical().termination, SolveTermination::Converged);
    assert_eq!(
        result.conflicting_relations(),
        [SketchRelationId::PatternPointLink(SketchEntityId(2))]
    );
    assert!(result.conflicting_constraints().is_empty());
}
#[test]
fn malformed_links_are_rejected_without_changing_the_sketch() {
    let mut sketch = geometry(&[(1, [0., 0.]), (2, [1., 0.]), (3, [2., 0.])]);
    sketch.add_pattern_point_link(link(1, 2, [1., 0.])).unwrap();
    sketch.add_pattern_point_link(link(2, 3, [1., 0.])).unwrap();
    let before = sketch.clone();
    for candidate in [
        link(3, 1, [0., 0.]),
        link(1, 2, [0., 0.]),
        link(1, 1, [0., 0.]),
        link(1, 99, [0., 0.]),
        link(1, 3, [f64::NAN, 0.]),
    ] {
        assert!(sketch.add_pattern_point_link(candidate).is_err());
        assert!(sketch.same_recipe(&before));
    }
    let mut bad = sketch.recipe();
    bad.version = 13;
    assert!(Sketch2D::from_recipe(&bad).is_err());
    bad.version = 14;
    bad.pattern_point_links = Some(vec![]);
    assert!(Sketch2D::from_recipe(&bad).is_err());
}
#[test]
fn deleting_a_linked_point_removes_only_its_dangling_structural_link() {
    let mut sketch = geometry(&[(1, [0., 0.]), (2, [1., 0.]), (3, [2., 0.])]);
    sketch.add_pattern_point_link(link(1, 2, [1., 0.])).unwrap();
    sketch.add_pattern_point_link(link(1, 3, [2., 0.])).unwrap();
    sketch.delete_entities(&[SketchEntityId(2)]).unwrap();
    assert_eq!(sketch.pattern_point_links(), [link(1, 3, [2., 0.])]);
    sketch.delete_entities(&[SketchEntityId(1)]).unwrap();
    assert!(sketch.pattern_point_links().is_empty());
    assert!(sketch.recipe().version < 14);
}
