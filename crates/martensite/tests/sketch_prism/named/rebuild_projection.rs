// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn certified_cube_projects_exact_hidden_and_visible_curves() {
    use oblikovati_martensite::DrawingProjectionFrame;
    let mut history = owner(cube_document());
    let body = certified(&history);
    let key = body
        .body_reference_key(&history, document())
        .unwrap()
        .to_vec();
    let frame = DrawingProjectionFrame {
        origin: [0., 0., 0.],
        normal: [1., 1., 1.],
        x_direction: [1., -1., 0.],
    };
    let projection = body
        .drawing_projection(&history, document(), &key, frame)
        .unwrap();
    assert_eq!((projection.visible.len(), projection.hidden.len()), (9, 3));
    for curve in projection.visible.iter().chain(&projection.hidden) {
        let [first, last] = curve.parameter_range(&history, document()).unwrap();
        assert!(first < last);
        assert!(
            curve
                .point_at(&history, document(), first)
                .unwrap()
                .iter()
                .all(|value| value.is_finite())
        );
        assert!(
            curve
                .point_at(&history, document(), last)
                .unwrap()
                .iter()
                .all(|value| value.is_finite())
        );
        assert!(curve.point_at(&history, document(), f64::NAN).is_err());
    }
    let top = DrawingProjectionFrame {
        origin: [0., 0., 0.],
        normal: [0., 0., 1.],
        x_direction: [1., 0., 0.],
    };
    let orthographic = body
        .drawing_projection(&history, document(), &key, top)
        .unwrap();
    use oblikovati_martensite::DrawingProjectionCurveKind::{Outline, Sharp};
    assert_eq!(
        orthographic
            .visible
            .iter()
            .filter(|curve| curve.kind == Sharp)
            .count(),
        4
    );
    assert_eq!(
        orthographic
            .visible
            .iter()
            .filter(|curve| curve.kind == Outline)
            .count(),
        4
    );
    let mut bounds = [
        f64::INFINITY,
        f64::INFINITY,
        f64::NEG_INFINITY,
        f64::NEG_INFINITY,
    ];
    for curve in &orthographic.visible {
        for parameter in curve.parameter_range(&history, document()).unwrap() {
            let point = curve.point_at(&history, document(), parameter).unwrap();
            bounds[0] = bounds[0].min(point[0]);
            bounds[1] = bounds[1].min(point[1]);
            bounds[2] = bounds[2].max(point[0]);
            bounds[3] = bounds[3].max(point[1]);
        }
    }
    for (actual, expected) in bounds.into_iter().zip([0., 0., 1., 1.]) {
        assert!(
            (actual - expected).abs() < 1e-8,
            "projected bounds {bounds:?}"
        );
    }
    let rotated = DrawingProjectionFrame {
        origin: [1., 2., 0.],
        normal: [0., 0., 1.],
        x_direction: [0., 1., 0.],
    };
    let rotated_projection = body
        .drawing_projection(&history, document(), &key, rotated)
        .unwrap();
    let mut rotated_bounds = [
        f64::INFINITY,
        f64::INFINITY,
        f64::NEG_INFINITY,
        f64::NEG_INFINITY,
    ];
    for curve in &rotated_projection.visible {
        for parameter in curve.parameter_range(&history, document()).unwrap() {
            let [x, y] = curve.point_at(&history, document(), parameter).unwrap();
            rotated_bounds[0] = rotated_bounds[0].min(x);
            rotated_bounds[1] = rotated_bounds[1].min(y);
            rotated_bounds[2] = rotated_bounds[2].max(x);
            rotated_bounds[3] = rotated_bounds[3].max(y);
        }
    }
    for (actual, expected) in rotated_bounds.into_iter().zip([-2., 0., -1., 1.]) {
        assert!(
            (actual - expected).abs() < 1e-8,
            "rotated projected bounds {rotated_bounds:?}"
        );
    }
    let other = owner(cube_document());
    assert!(
        body.drawing_projection(&other, document(), &key, frame)
            .is_err()
    );
    assert!(
        projection.visible[0]
            .parameter_range(&other, document())
            .is_err()
    );
    let mut invalid = frame;
    invalid.x_direction = invalid.normal;
    assert!(
        body.drawing_projection(&history, document(), &key, invalid)
            .is_err()
    );
    let token = history.begin("Size").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(1, "20 mm")?;
            solve_selected_sketches(snapshot);
            Ok::<(), oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert!(
        projection.visible[0]
            .parameter_range(&history, document())
            .is_err()
    );
    assert!(
        projection.visible[0]
            .point_at(&history, document(), 0.)
            .is_err()
    );
    assert_projected_top_width(&history, 2.);
    history.undo().unwrap();
    assert_projected_top_width(&history, 1.);
    history.redo().unwrap();
    assert_projected_top_width(&history, 2.);
    let reopened = restored_owner(history.snapshot());
    assert_projected_top_width(&reopened, 2.);
}
fn assert_projected_top_width(history: &DocumentEditHistory, expected: f64) {
    use oblikovati_martensite::DrawingProjectionFrame;
    let body = certified(history);
    let key = body.body_reference_key(history, document()).unwrap();
    let frame = DrawingProjectionFrame {
        origin: [0.; 3],
        normal: [0., 0., 1.],
        x_direction: [1., 0., 0.],
    };
    let projection = body
        .drawing_projection(history, document(), key, frame)
        .unwrap();
    let mut max_x = f64::NEG_INFINITY;
    for curve in &projection.visible {
        for parameter in curve.parameter_range(history, document()).unwrap() {
            max_x = max_x.max(curve.point_at(history, document(), parameter).unwrap()[0]);
        }
    }
    assert!(
        (max_x - expected).abs() < 1e-8,
        "projected width {max_x}: expected {expected}"
    );
}

#[test]
fn exact_projection_retains_a_hole_in_an_oblique_prism() {
    use oblikovati_martensite::DrawingProjectionFrame;
    let history = owner(holed_document(0));
    let body = certified(&history);
    let key = body.body_reference_key(&history, document()).unwrap();
    let frame = DrawingProjectionFrame {
        origin: [0.; 3],
        normal: [0., 0., 1.],
        x_direction: [1., 0., 0.],
    };
    let projection = body
        .drawing_projection(&history, document(), key, frame)
        .unwrap();
    let found = projection.visible.iter().any(|curve| {
        let [first, last] = curve.parameter_range(&history, document()).unwrap();
        let a = curve.point_at(&history, document(), first).unwrap();
        let b = curve.point_at(&history, document(), last).unwrap();
        let near = |point: [f64; 2], expected: [f64; 2]| {
            (point[0] - expected[0]).abs() < 1e-8 && (point[1] - expected[1]).abs() < 1e-8
        };
        (near(a, [1., 1.]) && near(b, [1., 2.])) || (near(b, [1., 1.]) && near(a, [1., 2.]))
    });
    assert!(found, "expected the visible top boundary of the hole");
}

#[test]
fn standard_base_view_centers_and_scales_current_cube_on_sheet() {
    use oblikovati_api::types::{BASE_VIEW_BACK, BASE_VIEW_FRONT, BASE_VIEW_TOP};
    use oblikovati_model::drawing::DrawingBaseViewDefinition;
    let mut history = owner(cube_document());
    let body = certified(&history);
    let key = body.body_reference_key(&history, document()).unwrap();
    for orientation in [BASE_VIEW_FRONT, BASE_VIEW_BACK, BASE_VIEW_TOP] {
        let definition =
            DrawingBaseViewDefinition::new("View", orientation, 0.5, [100., 80.]).unwrap();
        let view = body
            .drawing_base_view(&history, document(), key, &definition)
            .unwrap();
        let mut bounds = [
            f64::INFINITY,
            f64::INFINITY,
            f64::NEG_INFINITY,
            f64::NEG_INFINITY,
        ];
        for (index, curve) in view.curves.visible.iter().enumerate() {
            for parameter in curve.parameter_range(&history, document()).unwrap() {
                let [x, y] = view
                    .sheet_point(true, index, &history, document(), parameter)
                    .unwrap();
                bounds = [
                    bounds[0].min(x),
                    bounds[1].min(y),
                    bounds[2].max(x),
                    bounds[3].max(y),
                ];
            }
        }
        for (actual, expected) in bounds.into_iter().zip([97.5, 77.5, 102.5, 82.5]) {
            assert!((actual - expected).abs() < 1e-8, "sheet bounds {bounds:?}");
        }
        assert!(
            view.sheet_point(true, usize::MAX, &history, document(), 0.)
                .is_err()
        );
    }
    let definition = DrawingBaseViewDefinition::new("View", BASE_VIEW_TOP, 1., [0., 0.]).unwrap();
    let mut invalid = definition.clone();
    invalid.scale = f64::NAN;
    assert!(
        body.drawing_base_view(&history, document(), key, &invalid)
            .is_err()
    );
    invalid = definition.clone();
    invalid.orientation = oblikovati_api::types::BaseViewOrientation(255);
    assert!(
        body.drawing_base_view(&history, document(), key, &invalid)
            .is_err()
    );
    let view = body
        .drawing_base_view(&history, document(), key, &definition)
        .unwrap();
    let parameter = view.curves.visible[0]
        .parameter_range(&history, document())
        .unwrap()[0];
    let token = history.begin("Size").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(1, "20 mm")
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert!(
        view.sheet_point(true, 0, &history, document(), parameter)
            .is_err()
    );
}

#[test]
fn standard_view_projects_all_current_part_bodies_together() {
    use oblikovati_api::types::BASE_VIEW_FRONT;
    use oblikovati_martensite::{CertifiedNamedPrismEvaluator, drawing_base_view_bodies};
    use oblikovati_model::{drawing::DrawingBaseViewDefinition, feature::AuthoredFeature};
    let mut snapshot = cube_document();
    let definition = SketchPrismDefinition::new(
        SKETCH,
        selected_sketch_boundary(),
        [0., 0., -1.].map(DatumLength::Constant),
    )
    .unwrap();
    let second =
        AuthoredFeature::sketch_prism(FeatureId(2), "Second body", definition, &snapshot).unwrap();
    snapshot.work_geometry.append(second).unwrap();
    let mut history = owner(snapshot);
    let mut evaluator =
        CertifiedNamedPrismEvaluator::new(&history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let bodies: Vec<_> = result.ordered_bodies().collect();
    assert_eq!(bodies.len(), 2);
    let definition =
        DrawingBaseViewDefinition::new("Front", BASE_VIEW_FRONT, 1., [100., 80.]).unwrap();
    let view = drawing_base_view_bodies(&history, document(), &bodies, &definition).unwrap();
    let mut bounds = [
        f64::INFINITY,
        f64::INFINITY,
        f64::NEG_INFINITY,
        f64::NEG_INFINITY,
    ];
    for (index, curve) in view.curves.visible.iter().enumerate() {
        for parameter in curve.parameter_range(&history, document()).unwrap() {
            let [x, y] = view
                .sheet_point(true, index, &history, document(), parameter)
                .unwrap();
            bounds = [
                bounds[0].min(x),
                bounds[1].min(y),
                bounds[2].max(x),
                bounds[3].max(y),
            ];
        }
    }
    for (actual, expected) in bounds.into_iter().zip([95., 70., 105., 90.]) {
        assert!((actual - expected).abs() < 1e-8, "sheet bounds {bounds:?}");
    }
    let top =
        DrawingBaseViewDefinition::new("Top", oblikovati_api::types::BASE_VIEW_TOP, 1., [0., 0.])
            .unwrap();
    let top_view = drawing_base_view_bodies(&history, document(), &bodies, &top).unwrap();
    let near = drawing_base_view_bodies(&history, document(), &[bodies[0]], &top).unwrap();
    let far = drawing_base_view_bodies(&history, document(), &[bodies[1]], &top).unwrap();
    assert!(
        top_view.curves.hidden.len() > near.curves.hidden.len() + far.curves.hidden.len(),
        "the far body must gain hidden edges when the near body is present"
    );
    assert!(drawing_base_view_bodies(&history, document(), &[], &definition).is_err());
    assert!(
        drawing_base_view_bodies(&history, document(), &[bodies[0], bodies[0]], &definition)
            .is_err()
    );
    assert!(
        drawing_base_view_bodies(&owner(cube_document()), document(), &bodies, &definition)
            .is_err()
    );
    let parameter = view.curves.visible[0]
        .parameter_range(&history, document())
        .unwrap()[0];
    drop(result);
    drop(evaluator);
    let token = history.begin("Size").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(1, "20 mm")
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert!(
        view.sheet_point(true, 0, &history, document(), parameter)
            .is_err()
    );
}
