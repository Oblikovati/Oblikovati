// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchPolygonSeed;
#[test]
fn polygon_graph_fields_and_rank_match_go() {
    let fixtures: Vec<GoRecipe> = serde_json::from_str(include_str!("go-recipes.json")).unwrap();
    let polygons: Vec<_> = fixtures
        .into_iter()
        .filter(|f| f.kind == "polygon")
        .collect();
    assert_eq!(polygons.len(), 20);
    for fixture in polygons {
        verify_polygon_fixture(fixture);
    }
}
fn verify_polygon_fixture(fixture: GoRecipe) {
    let recipe = SketchPlacementRecipe::polygon(SketchPolygonSeed {
        center: fixture.input[0],
        through: fixture.input[1],
        sides: fixture.sides,
        inscribed: fixture.inscribed,
    })
    .unwrap();
    assert_eq!(recipe.points.len(), fixture.points.len());
    for (actual, expected) in recipe
        .points
        .iter()
        .flatten()
        .zip(fixture.points.iter().flatten())
    {
        close(*actual, *expected);
    }
    verify_entities(&recipe, &fixture);
    verify_relations(&recipe, &fixture);
    verify_fields(&recipe, &fixture);
    assert_eq!(fixture.dof, 4);
    assert_eq!(super::rectangles::solved_rank(&recipe), fixture.dof);
}
#[test]
fn polygon_recipes_refuse_invalid_counts_coordinates_and_radius() {
    for sides in [0, 1, 2, usize::MAX] {
        assert!(SketchPlacementRecipe::polygon(seed(sides, [5., 0.])).is_err());
    }
    for through in [
        [0., 0.],
        [f64::NAN, 0.],
        [f64::INFINITY, 0.],
        [f64::MAX, 0.],
    ] {
        assert!(SketchPlacementRecipe::polygon(seed(6, through)).is_err());
    }
}
fn seed(sides: usize, through: [f64; 2]) -> SketchPolygonSeed {
    SketchPolygonSeed {
        center: [0., 0.],
        through,
        sides,
        inscribed: true,
    }
}
#[test]
fn polygon_diameter_lock_and_frame_constraints_keep_zero_freedoms_during_edits() {
    use oblikovati_model::history::SketchPlacementFieldLock;
    let recipe = SketchPlacementRecipe::polygon(seed(5, [5., 0.])).unwrap();
    let lock = SketchPlacementFieldLock {
        expression: "2 * 5 cm".into(),
        text_point: None,
    };
    let (history, report) = super::rectangles::recipe_history(&recipe, &[Some(lock)]);
    let mut geometry = history
        .snapshot()
        .sketches
        .get(report.sketch)
        .unwrap()
        .geometry()
        .clone();
    constrain_polygon_frame(&mut geometry, &report);
    let mut parameters = history.snapshot().parameters.clone();
    for diameter in [14., 6., 10.] {
        parameters
            .set_expression(report.dimensions[0].1.parameter, &format!("{diameter} cm"))
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
        assert!((geometry.circles()[0].radius - diameter / 2.).abs() < 1e-8);
        assert_regular_geometry(&geometry);
    }
}
fn constrain_polygon_frame(
    geometry: &mut oblikovati_model::sketch::Sketch2D,
    report: &oblikovati_model::history::SketchPlacementCreationReport,
) {
    use oblikovati_model::sketch::{PlanarConstraint as C, SketchConstraintId};
    for (index, relation) in [
        C::Fix {
            point: report.points[5],
            position: [0., 0.],
        },
        C::Horizontal {
            points: [report.points[5], report.points[0]],
        },
    ]
    .into_iter()
    .enumerate()
    {
        geometry
            .add_constraint(SketchConstraintId(1000 + index as u64), relation)
            .unwrap();
    }
}
fn assert_regular_geometry(geometry: &oblikovati_model::sketch::Sketch2D) {
    let circle = geometry.circles()[0];
    let center = geometry.point(circle.center).unwrap().position;
    let vertices = &geometry.points()[..5];
    let edge = super::arcs::center_distance(vertices[0].position, vertices[1].position);
    for (i, vertex) in vertices.iter().enumerate() {
        assert!(
            (super::arcs::center_distance(center, vertex.position) - circle.radius).abs() < 1e-8
        );
        assert!(
            (super::arcs::center_distance(vertex.position, vertices[(i + 1) % 5].position) - edge)
                .abs()
                < 1e-8
        );
    }
}

pub(super) fn verify_circle(actual: SketchPlacementGeometry, expected: &GoEntity) {
    let SketchPlacementGeometry::Circle { center, radius } = actual else {
        panic!("expected circle, got {actual:?}")
    };
    assert_eq!(center, expected.points[0]);
    // Polygon trigonometry already uses the shared Go/Rust arithmetic bound for its vertices.
    close(radius, expected.radius);
}
