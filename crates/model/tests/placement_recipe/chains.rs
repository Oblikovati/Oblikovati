// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::{SketchPlacementCreationReport, SketchPolylineCreation},
    sketch::{PlanarConstraint, Sketch2D, SketchConstraintId, SketchRecipe},
};
#[test]
fn chains_match_go_members_final_segment_fields_and_rank_in_each_quadrant() {
    let fixtures: Vec<GoRecipe> = serde_json::from_str(include_str!("go-recipes.json")).unwrap();
    let chains: Vec<_> = fixtures.into_iter().filter(|f| f.kind == "chain").collect();
    assert_eq!(chains.len(), 20);
    for fixture in chains {
        let recipe = SketchPlacementRecipe::line_chain(&fixture.chain, fixture.closed).unwrap();
        assert_eq!(recipe.points, fixture.points);
        verify_entities(&recipe, &fixture);
        verify_relations(&recipe, &fixture);
        verify_fields(&recipe, &fixture);
        assert_eq!(fixture.dof, 2 * fixture.chain.len());
        assert_eq!(super::rectangles::solved_rank(&recipe), fixture.dof);
        verify_chain_adoption(&recipe, fixture.closed);
    }
}
fn verify_chain_adoption(recipe: &SketchPlacementRecipe, closed: bool) {
    let (mut history, report) = super::rectangles::recipe_history(recipe, &[]);
    let geometry = history
        .snapshot()
        .sketches
        .get(report.sketch)
        .unwrap()
        .geometry()
        .clone();
    assert_eq!(geometry.points().len(), recipe.points.len());
    assert_eq!(geometry.entity_ids(), report.members);
    for (line, member) in geometry.lines().iter().zip(&recipe.members) {
        let SketchPlacementGeometry::Path(SketchPathMember::Line(indices)) = member.geometry else {
            panic!("expected line")
        };
        assert_eq!(line.endpoints, indices.map(|index| report.points[index]));
    }
    verify_chain_reload(&geometry);
    verify_typed_chain_matches(recipe, closed, &geometry);
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
        geometry.same_recipe(
            history
                .snapshot()
                .sketches
                .get(report.sketch)
                .unwrap()
                .geometry()
        )
    );
}
fn verify_chain_reload(geometry: &Sketch2D) {
    let encoded = serde_json::to_vec(&geometry.recipe()).unwrap();
    let decoded: SketchRecipe = serde_json::from_slice(&encoded).unwrap();
    let restored = Sketch2D::from_recipe(&decoded).unwrap();
    assert!(geometry.same_recipe(&restored));
}
fn verify_typed_chain_matches(recipe: &SketchPlacementRecipe, closed: bool, expected: &Sketch2D) {
    let mut empty = recipe.clone();
    empty.points.clear();
    empty.members.clear();
    empty.fields.clear();
    let (mut history, report) = super::rectangles::recipe_history(&empty, &[]);
    let token = history.begin("Typed chain").unwrap();
    let input = SketchPolylineCreation {
        points: recipe.points.clone(),
        closed,
        construction: false,
    };
    history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_polyline(report.sketch, &input, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert!(
        expected.same_recipe(
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
fn chain_fields_refuse_incomplete_collapsed_nonfinite_and_unrepresentable_segments() {
    for points in [vec![], vec![[0., 0.]]] {
        assert!(SketchPlacementRecipe::line_chain(&points, false).is_err());
    }
    assert!(SketchPlacementRecipe::line_chain(&[[0., 0.], [1., 1.]], true).is_err());
    for points in [
        [[0., 0.], [0., 0.], [2., 1.]],
        [[0., 0.], [1., f64::NAN], [2., 1.]],
        [[0., 0.], [f64::INFINITY, 1.], [2., 1.]],
        [[0., 0.], [f64::MAX, 1.], [2., 1.]],
        [[0., 0.], [f64::MIN_POSITIVE, 0.], [2., 1.]],
    ] {
        assert!(SketchPlacementRecipe::line_chain(&points, false).is_err());
        assert!(SketchPlacementRecipe::line_chain(&points, true).is_err());
    }
    assert!(SketchPlacementRecipe::line_chain(&[[0., 0.], [1., 1.], [0., 0.]], true).is_err());
}
#[test]
fn final_length_dimension_and_shared_vertices_follow_formula_edits_at_zero_dof() {
    for closed in [false, true] {
        let recipe =
            SketchPlacementRecipe::line_chain(&[[0., 0.], [4., 0.], [4., 3.]], closed).unwrap();
        let locks = [
            Some(oblikovati_model::history::SketchPlacementFieldLock {
                expression: if closed { "5 cm" } else { "3 cm" }.into(),
                text_point: None,
            }),
            None,
        ];
        let (history, report) = super::rectangles::recipe_history(&recipe, &locks);
        assert_eq!(report.dimensions.len(), 1);
        let mut geometry = history
            .snapshot()
            .sketches
            .get(report.sketch)
            .unwrap()
            .geometry()
            .clone();
        constrain_chain_frame(&mut geometry, &report);
        verify_chain_dimensions(
            geometry,
            &report,
            history.snapshot().parameters.clone(),
            closed,
        );
    }
}
fn constrain_chain_frame(geometry: &mut Sketch2D, report: &SketchPlacementCreationReport) {
    for (index, relation) in [
        PlanarConstraint::Fix {
            point: report.points[0],
            position: [0., 0.],
        },
        PlanarConstraint::Fix {
            point: report.points[1],
            position: [4., 0.],
        },
        PlanarConstraint::Vertical {
            points: [report.points[1], report.points[2]],
        },
    ]
    .into_iter()
    .enumerate()
    {
        geometry
            .add_constraint(SketchConstraintId(100 + index as u64), relation)
            .unwrap();
    }
}
fn verify_chain_dimensions(
    mut geometry: Sketch2D,
    report: &SketchPlacementCreationReport,
    mut parameters: oblikovati_model::parameter::ParameterCollection<u64>,
    closed: bool,
) {
    let endpoints: Vec<_> = geometry
        .lines()
        .iter()
        .map(|line| (line.id, line.endpoints))
        .collect();
    for length in if closed {
        [5_f64, 10., 6.]
    } else {
        [3_f64, 6., 2.]
    } {
        parameters
            .set_expression(
                report.dimensions[0].1.parameter,
                &format!("2 * {} cm", length / 2.),
            )
            .unwrap();
        let solved = geometry
            .solve(&parameters, Default::default(), || false)
            .unwrap();
        assert_eq!(
            solved.numerical().termination,
            oblikovati_solve::nonlinear::SolveTermination::Converged
        );
        assert_eq!(solved.numerical().structure.degrees_of_freedom, 0);
        assert_eq!(solved.numerical().structure.redundant_equations, 0);
        geometry = solved.into_candidate();
        let expected_height = if closed {
            (length * length - 16.).sqrt()
        } else {
            length
        };
        let actual = geometry.point(report.points[2]).unwrap().position;
        assert!((actual[0] - 4.).abs() < 1e-8 && (actual[1] - expected_height).abs() < 1e-8);
        assert_eq!(
            geometry
                .lines()
                .iter()
                .map(|line| (line.id, line.endpoints))
                .collect::<Vec<_>>(),
            endpoints
        );
    }
}

#[test]
fn coincident_nonadjacent_vertices_keep_distinct_ids_until_explicit_closure() {
    let recipe = SketchPlacementRecipe::line_chain(&[[0., 0.], [4., 0.], [0., 0.]], false).unwrap();
    let (history, report) = super::rectangles::recipe_history(&recipe, &[]);
    let geometry = history
        .snapshot()
        .sketches
        .get(report.sketch)
        .unwrap()
        .geometry();
    assert_ne!(report.points[0], report.points[2]);
    assert_eq!(
        geometry.point(report.points[0]).unwrap().position,
        geometry.point(report.points[2]).unwrap().position
    );
    assert_eq!(
        geometry.lines()[1].endpoints,
        [report.points[1], report.points[2]]
    );
}
