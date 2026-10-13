// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn all_rectangle_graphs_fields_and_rank_match_pinned_go_in_every_quadrant() {
    let fixtures: Vec<GoRecipe> = serde_json::from_str(include_str!("go-recipes.json")).unwrap();
    let rectangles: Vec<_> = fixtures
        .into_iter()
        .filter(|f| !f.kind.is_empty())
        .collect();
    assert_eq!(rectangles.len(), 12);
    for fixture in rectangles {
        let [a, b, c] = fixture.input;
        let actual = match fixture.kind.as_str() {
            "rectangle" => SketchPlacementRecipe::rectangle([a, b]),
            "center" => SketchPlacementRecipe::center_rectangle(a, b),
            "threePoint" => SketchPlacementRecipe::three_point_rectangle([a, b], c),
            value => panic!("unknown rectangle kind {value}"),
        }
        .unwrap();
        assert_eq!(actual.points.len(), fixture.points.len());
        for (a, b) in actual
            .points
            .iter()
            .flatten()
            .zip(fixture.points.iter().flatten())
        {
            close(*a, *b);
        }
        assert_eq!(
            fixture.dof,
            if fixture.kind == "threePoint" { 5 } else { 4 }
        );
        assert_eq!(solved_rank(&actual), fixture.dof);
        verify_entities(&actual, &fixture);
        verify_relations(&actual, &fixture);
        verify_fields(&actual, &fixture);
    }
}
#[test]
fn rectangles_refuse_collapsed_edges_and_nonfinite_placements_before_commit() {
    for corners in [
        [[0., 0.], [0., 2.]],
        [[0., 0.], [2., 0.]],
        [[0., 0.], [f64::INFINITY, 2.]],
        [[-f64::MAX, 0.], [f64::MAX, 1.]],
    ] {
        assert!(SketchPlacementRecipe::rectangle(corners).is_err());
        assert!(SketchPlacementRecipe::center_rectangle(corners[0], corners[1]).is_err());
    }
    for (base, height) in [
        ([[0., 0.], [0., 0.]], [1., 2.]),
        ([[0., 0.], [2., 0.]], [1., 0.]),
        ([[0., 0.], [f64::MAX, 0.]], [1., 2.]),
        ([[0., 0.], [2., 0.]], [f64::NAN, 2.]),
    ] {
        assert!(SketchPlacementRecipe::three_point_rectangle(base, height).is_err());
    }
}

#[allow(dead_code)]
#[path = "../common/document_snapshot.rs"]
mod document_snapshot;
fn solved_rank(recipe: &SketchPlacementRecipe) -> usize {
    use oblikovati_model::{
        document_sketch::DocumentSketchId,
        history::DocumentEditHistory,
        sketch::{Sketch2D, SketchPlane},
    };
    let mut snapshot = document_snapshot::empty_snapshot();
    let sketch = DocumentSketchId(1);
    snapshot
        .sketches
        .add(sketch, "Oracle", SketchPlane::xy(), Sketch2D::default())
        .unwrap();
    let mut history = DocumentEditHistory::new(snapshot, Default::default()).unwrap();
    let token = history.begin("Oracle").unwrap();
    history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_placement(sketch, recipe, &[], Default::default(), ids)
        })
        .unwrap();
    let snapshot = history.snapshot();
    let solved = snapshot
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .solve(&snapshot.parameters, Default::default(), || false)
        .unwrap();
    assert_eq!(
        solved.numerical().termination,
        oblikovati_solve::nonlinear::SolveTermination::Converged
    );
    assert_eq!(solved.numerical().structure.redundant_equations, 0);
    solved.numerical().structure.degrees_of_freedom
}
