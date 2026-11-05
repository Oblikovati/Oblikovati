// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn arc_geometry_fields_rank_and_free_circularity_match_go() {
    let fixtures: Vec<GoRecipe> = serde_json::from_str(include_str!("go-recipes.json")).unwrap();
    let arcs: Vec<_> = fixtures.into_iter().filter(|f| f.kind == "arc").collect();
    assert_eq!(arcs.len(), 10);
    for fixture in arcs {
        verify_go_arc(fixture);
    }
}
fn verify_go_arc(fixture: GoRecipe) {
    let [center, start, end] = fixture.input;
    let recipe = SketchPlacementRecipe::arc(center, [start, end], fixture.ccw).unwrap();
    assert_eq!(recipe.points, fixture.points);
    assert!(recipe.relations.is_empty());
    verify_entities(&recipe, &fixture);
    verify_fields(&recipe, &fixture);
    let solved = super::rectangles::solved_recipe(&recipe);
    assert_eq!(fixture.dof, 5);
    assert_eq!(solved.numerical().structure.degrees_of_freedom, fixture.dof);
    let actual: Vec<_> = solved
        .candidate()
        .points()
        .iter()
        .map(|point| point.position)
        .collect();
    let expected = fixture.solved_points.unwrap();
    assert_circular_positions(&actual);
    assert_circular_positions(&expected);
    if center_distance(center, start) == center_distance(center, end) {
        assert_positions(&actual, &expected);
    }
}
#[test]
fn arc_recipes_refuse_nonfinite_or_zero_radial_input() {
    for points in [
        [[0., 0.], [0., 0.], [1., 0.]],
        [[0., 0.], [1., 0.], [0., 0.]],
        [[0., 0.], [1., 0.], [1., 0.]],
        [[f64::NAN, 0.], [1., 0.], [0., 1.]],
        [[0., 0.], [f64::MAX, 0.], [0., 1.]],
        [[0., 0.], [1., 0.], [0., f64::INFINITY]],
    ] {
        assert!(SketchPlacementRecipe::arc(points[0], [points[1], points[2]], true).is_err());
    }
}
pub(super) fn center_distance(center: [f64; 2], point: [f64; 2]) -> f64 {
    (point[0] - center[0]).hypot(point[1] - center[1])
}
fn assert_circular_positions(points: &[[f64; 2]]) {
    let start = center_distance(points[0], points[1]);
    let end = center_distance(points[0], points[2]);
    assert!(
        start > 0. && (start - end).abs() < 1e-8,
        "noncircular points {points:?}"
    );
}
fn assert_positions(actual: &[[f64; 2]], expected: &[[f64; 2]]) {
    assert_eq!(actual.len(), expected.len());
    for (a, b) in actual.iter().flatten().zip(expected.iter().flatten()) {
        assert!((a - b).abs() < 1e-7, "{actual:?} != {expected:?}");
    }
}
#[test]
fn fully_constrained_arc_radius_edits_match_unique_go_solutions() {
    let fixtures: Vec<GoRecipe> = serde_json::from_str(include_str!("go-recipes.json")).unwrap();
    let constrained: Vec<_> = fixtures
        .into_iter()
        .filter(|f| f.locked.is_some())
        .collect();
    assert_eq!(constrained.len(), 8);
    for fixture in constrained {
        verify_locked_arc(fixture);
    }
}
fn verify_locked_arc(fixture: GoRecipe) {
    use oblikovati_model::history::SketchPlacementFieldLock;
    let [center, start, end] = fixture.input;
    let recipe = SketchPlacementRecipe::arc(center, [start, end], fixture.ccw).unwrap();
    let locks = [
        Some(SketchPlacementFieldLock {
            expression: "2.5 * 2 cm".into(),
            text_point: None,
        }),
        None,
    ];
    let (history, report) = super::rectangles::recipe_history(&recipe, &locks);
    let mut geometry = history
        .snapshot()
        .sketches
        .get(report.sketch)
        .unwrap()
        .geometry()
        .clone();
    lock_arc_directions(&mut geometry, &report, center);
    let mut parameters = history.snapshot().parameters.clone();
    for sample in fixture.locked.unwrap() {
        parameters
            .set_expression(
                report.dimensions[0].1.parameter,
                &format!("{} cm", sample.radius),
            )
            .unwrap();
        let solved = geometry
            .solve(&parameters, Default::default(), || false)
            .unwrap();
        assert_eq!(
            solved.numerical().termination,
            oblikovati_solve::nonlinear::SolveTermination::Converged
        );
        assert_eq!(sample.dof, 0);
        assert_eq!(solved.numerical().structure.degrees_of_freedom, 0);
        assert_eq!(solved.numerical().structure.redundant_equations, 0);
        geometry = solved.into_candidate();
        assert_positions(
            &geometry
                .points()
                .iter()
                .map(|p| p.position)
                .collect::<Vec<_>>(),
            &sample.points,
        );
    }
}
fn lock_arc_directions(
    geometry: &mut oblikovati_model::sketch::Sketch2D,
    report: &oblikovati_model::history::SketchPlacementCreationReport,
    center: [f64; 2],
) {
    use oblikovati_model::sketch::{PlanarConstraint as C, SketchConstraintId};
    for (i, constraint) in [
        C::Fix {
            point: report.points[0],
            position: center,
        },
        C::Horizontal {
            points: [report.points[0], report.points[1]],
        },
        C::Vertical {
            points: [report.points[0], report.points[2]],
        },
    ]
    .into_iter()
    .enumerate()
    {
        geometry
            .add_constraint(SketchConstraintId(1000 + i as u64), constraint)
            .unwrap();
    }
}
