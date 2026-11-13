// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{identity::RecipeIdAllocator, sketch::SketchRecipe};

#[test]
fn translated_circle_copies_follow_seed_position_and_driven_radius_after_reload() {
    let mut sketch = geometry(&[(1, [0., 0.])]);
    sketch
        .add_circle(SketchEntityId(2), SketchEntityId(1), 1.)
        .unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Radius {
            curve: SketchEntityId(2),
            parameter: 1,
        },
    );
    let mut ids = RecipeIdAllocator::restore(Some(10), &[]).unwrap();
    let reports = sketch
        .copy_associative_translations(&[SketchEntityId(2)], &[[10., 0.], [0., 5.]], &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap();
    assert_eq!(sketch.pattern_point_links().len(), 2);
    assert_eq!(sketch.pattern_radius_links().len(), 2);
    let encoded = serde_json::to_vec(&sketch.recipe()).unwrap();
    let recipe: SketchRecipe = serde_json::from_slice(&encoded).unwrap();
    let restored = Sketch2D::from_recipe(&recipe).unwrap();
    assert!(sketch.same_recipe(&restored));
    let mut params = parameters();
    params.set_expression(1, "5 cm").unwrap();
    let solved = restored
        .solve_with_grounded_points(
            &params,
            &[point(1, [2., -3.])],
            SolveOptions::default(),
            || false,
        )
        .unwrap();
    assert_solved(&solved, 7);
    for (report, center) in reports.iter().zip([[12., -3.], [2., 2.]]) {
        let circle = solved.candidate().circle(report.copies[0]).unwrap();
        let actual = solved.candidate().point(circle.center).unwrap().position;
        assert!((actual[0] - center[0]).abs() < 1e-7 && (actual[1] - center[1]).abs() < 1e-7);
        assert!((circle.radius - 5.).abs() < 1e-7);
    }
    let mut removed = solved.into_candidate();
    removed.delete_entities(&[reports[0].copies[0]]).unwrap();
    assert_eq!(removed.pattern_radius_links().len(), 1);
    assert_eq!(removed.pattern_point_links().len(), 1);
}
#[test]
fn invalid_associative_copy_never_reserves_or_publishes_a_partial_batch() {
    let mut sketch = geometry(&[(1, [0., 0.])]);
    sketch
        .add_circle(SketchEntityId(2), SketchEntityId(1), 1.)
        .unwrap();
    let before = sketch.clone();
    let mut ids = RecipeIdAllocator::restore(Some(10), &[]).unwrap();
    for (selection, offsets) in [
        (vec![SketchEntityId(2)], vec![[1., 0.], [f64::NAN, 0.]]),
        (vec![SketchEntityId(99)], vec![[1., 0.]]),
        (vec![], vec![[1., 0.]]),
    ] {
        assert!(
            sketch
                .copy_associative_translations(&selection, &offsets, &mut || ids
                    .allocate()
                    .map(SketchEntityId))
                .is_err()
        );
        assert!(sketch.same_recipe(&before));
        assert_eq!(ids.high_water(), 10);
    }
}

#[test]
fn radius_links_reject_missing_duplicate_and_cyclic_members() {
    use oblikovati_model::sketch::SketchPatternRadiusLink;
    let mut sketch = geometry(&[(1, [0., 0.])]);
    for id in 2..=4 {
        sketch
            .add_circle(SketchEntityId(id), SketchEntityId(1), 1.)
            .unwrap();
    }
    let link = |source, member| SketchPatternRadiusLink {
        source: SketchEntityId(source),
        member: SketchEntityId(member),
    };
    sketch.add_pattern_radius_link(link(2, 3)).unwrap();
    sketch.add_pattern_radius_link(link(3, 4)).unwrap();
    let before = sketch.clone();
    for invalid in [link(4, 2), link(2, 3), link(2, 2), link(2, 99)] {
        assert!(sketch.add_pattern_radius_link(invalid).is_err());
        assert!(sketch.same_recipe(&before));
    }
    let mut legacy = sketch.recipe();
    legacy.version = 13;
    assert!(Sketch2D::from_recipe(&legacy).is_err());
    let restored = Sketch2D::from_recipe(&sketch.recipe()).unwrap();
    assert!(sketch.same_recipe(&restored));
}

#[test]
fn ellipse_seed_is_declined_before_any_copy_or_identity_reservation() {
    use oblikovati_model::sketch::SketchEllipse2D;
    let mut sketch = geometry(&[(1, [0., 0.])]);
    sketch
        .add_ellipse(
            SketchEllipse2D::new(SketchEntityId(2), SketchEntityId(1), [1., 0.], [3., 1.]).unwrap(),
        )
        .unwrap();
    let before = sketch.clone();
    let mut ids = RecipeIdAllocator::restore(Some(10), &[]).unwrap();
    assert!(
        sketch
            .copy_associative_translations(&[SketchEntityId(2)], &[[5., 0.]], &mut || ids
                .allocate()
                .map(SketchEntityId))
            .unwrap_err()
            .to_string()
            .contains("ellipse")
    );
    assert!(sketch.same_recipe(&before));
    assert_eq!(ids.high_water(), 10);
}
