// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn ellipse_fields_frame_and_rank_match_go() {
    let fixtures: Vec<GoRecipe> = serde_json::from_str(include_str!("go-recipes.json")).unwrap();
    let ellipses: Vec<_> = fixtures
        .into_iter()
        .filter(|f| f.kind == "ellipse")
        .collect();
    assert_eq!(ellipses.len(), 12);
    for fixture in ellipses {
        let [center, axis, radii] = fixture.input;
        let recipe = SketchPlacementRecipe::ellipse(center, axis, radii).unwrap();
        assert_eq!(recipe.points, fixture.points);
        verify_entities(&recipe, &fixture);
        verify_relations(&recipe, &fixture);
        verify_fields(&recipe, &fixture);
        assert_eq!(fixture.dof, 5);
        assert_eq!(super::rectangles::solved_rank(&recipe), fixture.dof);
    }
}
#[test]
fn ellipse_recipe_refuses_invalid_frames_radii_and_witnesses() {
    for (center, axis, radii) in [
        ([0., 0.], [0., 0.], [5., 2.]),
        ([0., 0.], [f64::NAN, 0.], [5., 2.]),
        ([0., 0.], [1., 0.], [0., 0.]),
        ([0., 0.], [1., 0.], [2., 5.]),
        ([0., 0.], [1., 0.], [5., -1.]),
        ([0., 0.], [1., 0.], [f64::INFINITY, 2.]),
        ([f64::NAN, 0.], [1., 0.], [5., 2.]),
        ([f64::MAX, 0.], [1., 0.], [f64::MAX, 2.]),
    ] {
        assert!(SketchPlacementRecipe::ellipse(center, axis, radii).is_err());
    }
    for axis in [[f64::MAX, 0.], [f64::MIN_POSITIVE, 0.]] {
        let recipe = SketchPlacementRecipe::ellipse([0., 0.], axis, [5., 2.]).unwrap();
        assert_eq!(recipe.fields[0].witness, [[0., 0.], [5., 0.]]);
    }
}
#[test]
fn locked_major_radius_creates_one_parameter_and_minor_field_only_steers() {
    use oblikovati_model::history::SketchPlacementFieldLock;
    let recipe = SketchPlacementRecipe::ellipse([3., 4.], [0.6, 0.8], [5., 2.]).unwrap();
    let locks = [
        Some(SketchPlacementFieldLock {
            expression: "2.5 * 2 cm".into(),
            text_point: None,
        }),
        Some(SketchPlacementFieldLock {
            expression: "2 cm".into(),
            text_point: None,
        }),
    ];
    let (history, report) = super::rectangles::recipe_history(&recipe, &locks);
    assert_eq!(report.dimensions.len(), 1);
    let mut parameters = history.snapshot().parameters.clone();
    let geometry = history
        .snapshot()
        .sketches
        .get(report.sketch)
        .unwrap()
        .geometry();
    for radius in [7., 4., 10.] {
        parameters
            .set_expression(report.dimensions[0].1.parameter, &format!("{radius} cm"))
            .unwrap();
        let solved = geometry
            .solve(&parameters, Default::default(), || false)
            .unwrap();
        assert_eq!(
            solved.numerical().termination,
            oblikovati_solve::nonlinear::SolveTermination::Converged
        );
        assert_eq!(solved.numerical().structure.degrees_of_freedom, 4);
        let ellipse = solved.candidate().ellipses()[0];
        assert!((ellipse.radii()[0] - radius).abs() < 1e-8);
        assert_eq!(ellipse.radii()[1], 2.);
        assert_eq!(ellipse.major_axis(), geometry.ellipses()[0].major_axis());
    }
}

#[test]
fn placed_ellipse_can_be_fully_constrained_and_edit_its_radius_formula() {
    use oblikovati_model::history::SketchPlacementFieldLock;
    let recipe = SketchPlacementRecipe::ellipse([3., 4.], [1., 0.], [5., 2.]).unwrap();
    let lock = SketchPlacementFieldLock {
        expression: "2.5 * 2 cm".into(),
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
    let mut parameters = history.snapshot().parameters.clone();
    parameters
        .insert_expression(
            1000,
            "minor_input",
            oblikovati_api::types::USER_PARAM,
            "2 cm",
        )
        .unwrap();
    constrain_placed_ellipse(&mut geometry, &report);
    for expression in ["3 * 2 cm", "2 * 2 cm"] {
        parameters
            .set_expression(report.dimensions[0].1.parameter, expression)
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
        assert_eq!(geometry.points()[0].position, [3., 4.]);
        assert!((geometry.ellipses()[0].radii()[1] - 2.).abs() < 1e-8);
    }
    assert!((geometry.ellipses()[0].radii()[0] - 4.).abs() < 1e-8);
}
fn constrain_placed_ellipse(
    geometry: &mut oblikovati_model::sketch::Sketch2D,
    report: &oblikovati_model::history::SketchPlacementCreationReport,
) {
    use oblikovati_model::sketch::{PlanarConstraint as C, SketchConstraintId};
    for (index, constraint) in [
        C::Fix {
            point: report.points[0],
            position: [3., 4.],
        },
        C::EllipseAxisAlignment {
            ellipse: report.members[0],
            major_axis: true,
            vertical: false,
        },
        C::EllipseMinorRadius {
            ellipse: report.members[0],
            parameter: 1000,
        },
    ]
    .into_iter()
    .enumerate()
    {
        geometry
            .add_constraint(SketchConstraintId(1000 + index as u64), constraint)
            .unwrap();
    }
}

pub(super) fn verify_frame(actual: SketchPlacementGeometry, expected: &GoEntity) {
    let SketchPlacementGeometry::Ellipse {
        center,
        major_axis,
        radii,
    } = actual
    else {
        panic!("expected an ellipse, got {actual:?}")
    };
    assert_eq!(center, expected.points[0]);
    assert_eq!(radii, [expected.radius, expected.minor_radius]);
    // The existing scaled normalization can differ from Go's unit seed by one rounding step.
    for (actual, expected) in major_axis.into_iter().zip(expected.major_axis) {
        close(actual, expected);
    }
}
