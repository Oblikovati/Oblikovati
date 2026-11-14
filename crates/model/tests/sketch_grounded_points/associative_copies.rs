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

#[test]
fn parameter_driven_copies_move_on_spacing_and_radius_edits_after_reload() {
    use oblikovati_api::types::USER_PARAM;
    use oblikovati_model::sketch::SketchPatternOffsetTerm;
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
    let mut params = parameters();
    params
        .insert_expression(2, "dx", USER_PARAM, "10 cm")
        .unwrap();
    params
        .insert_expression(3, "dy", USER_PARAM, "5 cm")
        .unwrap();
    let terms = |i: f64, j: f64| {
        vec![
            SketchPatternOffsetTerm {
                parameter: 2,
                coefficient: [-i, 0.],
            },
            SketchPatternOffsetTerm {
                parameter: 3,
                coefficient: [0., j],
            },
        ]
    };
    let occurrences = vec![
        terms(0., 1.),
        terms(1., 0.),
        terms(1., 1.),
        terms(2., 0.),
        terms(2., 1.),
    ];
    let mut ids = RecipeIdAllocator::restore(Some(10), &[]).unwrap();
    let reports = sketch
        .copy_parametric_translations(&[SketchEntityId(2)], &occurrences, &params, &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap();
    assert_eq!(sketch.parameter_dependencies(), [1, 2, 3]);
    let bytes = serde_json::to_vec(&sketch.recipe()).unwrap();
    let recipe: SketchRecipe = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(recipe.version, 15);
    let restored = Sketch2D::from_recipe(&recipe).unwrap();
    assert!(sketch.same_recipe(&restored));
    params.set_expression(1, "4 cm").unwrap();
    params.set_expression(2, "6 cm").unwrap();
    params.set_expression(3, "2 cm").unwrap();
    let solved = restored
        .solve_with_grounded_points(
            &params,
            &[point(1, [1., 3.])],
            SolveOptions::default(),
            || false,
        )
        .unwrap();
    assert_solved(&solved, 16);
    for (report, expected) in
        reports
            .iter()
            .zip([[1., 5.], [-5., 3.], [-5., 5.], [-11., 3.], [-11., 5.]])
    {
        let circle = solved.candidate().circle(report.copies[0]).unwrap();
        let center = solved.candidate().point(circle.center).unwrap().position;
        assert!(
            (center[0] - expected[0]).abs() < 1e-7 && (center[1] - expected[1]).abs() < 1e-7,
            "{center:?}"
        );
        assert!((circle.radius - 4.).abs() < 1e-7);
    }
    let mut deleted = solved.into_candidate();
    deleted.delete_entities(&[SketchEntityId(1)]).unwrap();
    assert!(deleted.pattern_offsets().is_empty());
}

#[test]
fn bad_pattern_parameters_are_declined_before_copy_and_report_dependencies() {
    use oblikovati_api::types::USER_PARAM;
    use oblikovati_model::sketch::SketchPatternOffsetTerm;
    let mut sketch = geometry(&[(1, [0., 0.])]);
    let before = sketch.clone();
    let mut params = parameters();
    params
        .insert_expression(2, "negative", USER_PARAM, "-1 cm")
        .unwrap();
    params
        .insert_expression(3, "angle", USER_PARAM, "2 rad")
        .unwrap();
    let mut ids = RecipeIdAllocator::restore(Some(10), &[]).unwrap();
    for parameter in [0, 2, 3, 99] {
        let terms = vec![vec![SketchPatternOffsetTerm {
            parameter,
            coefficient: [1., 0.],
        }]];
        assert!(
            sketch
                .copy_parametric_translations(&[SketchEntityId(1)], &terms, &params, &mut || ids
                    .allocate()
                    .map(SketchEntityId))
                .is_err()
        );
        assert!(sketch.same_recipe(&before));
        assert_eq!(ids.high_water(), 10);
    }
}

#[test]
fn offset_formula_schema_rejects_legacy_missing_duplicate_and_dangling_records() {
    use oblikovati_model::sketch::SketchPatternOffsetTerm;
    let mut sketch = geometry(&[(1, [0., 0.])]);
    let mut ids = RecipeIdAllocator::restore(Some(10), &[]).unwrap();
    let terms = vec![vec![SketchPatternOffsetTerm {
        parameter: 1,
        coefficient: [1., 0.],
    }]];
    sketch
        .copy_parametric_translations(&[SketchEntityId(1)], &terms, &parameters(), &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap();
    let recipe = sketch.recipe();
    let mut invalid = recipe.clone();
    invalid.version = 14;
    assert!(Sketch2D::from_recipe(&invalid).is_err());
    invalid = recipe.clone();
    invalid.pattern_offsets = None;
    assert!(Sketch2D::from_recipe(&invalid).is_err());
    invalid = recipe.clone();
    let duplicate = invalid.pattern_offsets.as_ref().unwrap()[0].clone();
    invalid.pattern_offsets.as_mut().unwrap().push(duplicate);
    assert!(Sketch2D::from_recipe(&invalid).is_err());
    invalid = recipe;
    invalid.pattern_offsets.as_mut().unwrap()[0].member = SketchEntityId(999);
    assert!(Sketch2D::from_recipe(&invalid).is_err());
}

#[test]
fn unhealthy_spacing_keeps_its_dependency_and_declines_solving() {
    use oblikovati_model::sketch::SketchPatternOffsetTerm;
    let mut sketch = geometry(&[(1, [0., 0.])]);
    let mut ids = RecipeIdAllocator::restore(Some(10), &[]).unwrap();
    let mut params = parameters();
    let terms = vec![vec![SketchPatternOffsetTerm {
        parameter: 1,
        coefficient: [1., 0.],
    }]];
    sketch
        .copy_parametric_translations(&[SketchEntityId(1)], &terms, &params, &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap();
    let before = sketch.clone();
    params.set_expression(1, "missing + 1 cm").unwrap();
    let error = sketch
        .solve(&params, SolveOptions::default(), || false)
        .unwrap_err();
    assert!(matches!(
        error,
        SketchError::PatternParameter { parameter: 1, .. }
    ));
    assert_eq!(sketch.parameter_dependencies(), [1]);
    assert!(before.same_recipe(&sketch));
}
