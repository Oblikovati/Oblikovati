// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::{SketchPlane, SketchPlaneOptions, SketchPlaneRecipe};
fn plane() -> SketchPlane {
    SketchPlane::new(
        [10.0, -20.0, 30.0],
        [0.6, 0.8, 0.0],
        [0.0, 0.0, 1.0],
        Default::default(),
    )
    .unwrap()
}
fn close<const N: usize>(actual: [f64; N], expected: [f64; N]) {
    for (actual, expected) in actual.into_iter().zip(expected) {
        assert!(
            (actual - expected).abs() < 1e-12,
            "actual {actual}, expected {expected}"
        );
    }
}
#[test]
fn standard_and_rotated_translated_planes_map_and_project_with_right_handed_normals() {
    for (plane, model, normal) in [
        (SketchPlane::xy(), [2.0, 3.0, 0.0], [0.0, 0.0, 1.0]),
        (SketchPlane::xz(), [2.0, 0.0, 3.0], [0.0, -1.0, 0.0]),
        (SketchPlane::yz(), [0.0, 2.0, 3.0], [1.0, 0.0, 0.0]),
        (plane(), [11.2, -18.4, 33.0], [0.8, -0.6, 0.0]),
    ] {
        close(plane.to_model([2.0, 3.0]).unwrap(), model);
        close(plane.normal(), normal);
        close(plane.to_sketch(model).unwrap(), [2.0, 3.0]);
        let lifted = std::array::from_fn(|axis| model[axis] + 7.0 * normal[axis]);
        close(plane.to_sketch(lifted).unwrap(), [2.0, 3.0]);
    }
}
#[test]
fn authored_plane_axes_and_origin_survive_recipe_round_trip_without_normal_storage() {
    let plane = plane();
    let encoded = serde_json::to_string(&plane.recipe()).unwrap();
    assert!(!encoded.contains("normal"));
    let recipe: SketchPlaneRecipe = serde_json::from_str(&encoded).unwrap();
    let restored = SketchPlane::from_recipe(recipe, Default::default()).unwrap();
    assert_eq!(restored, plane);
    for (actual, expected) in restored
        .recipe()
        .x_axis
        .into_iter()
        .zip(plane.recipe().x_axis)
    {
        assert_eq!(actual.to_bits(), expected.to_bits());
    }
    assert!(
        serde_json::from_str::<SketchPlaneRecipe>(
            r#"{"origin":[0,0,0],"x_axis":[1,0,0],"y_axis":[0,1,0],"normal":[0,0,1]}"#
        )
        .is_err()
    );
}
#[test]
fn invalid_frames_and_options_name_the_offending_values() {
    let default = SketchPlaneOptions::default();
    for (origin, x, y) in [
        ([f64::INFINITY, 0.0, 0.0], [1.0, 0.0, 0.0], [0.0, 1.0, 0.0]),
        ([0.0; 3], [0.0; 3], [0.0, 1.0, 0.0]),
        ([0.0; 3], [2.0, 0.0, 0.0], [0.0, 1.0, 0.0]),
        ([0.0; 3], [1.0, 0.0, 0.0], [1.0, 0.0, 0.0]),
        ([0.0; 3], [f64::MAX, f64::MAX, 0.0], [0.0, 1.0, 0.0]),
        ([0.0; 3], [1.0, 0.0, 0.0], [f64::NAN, 0.0, 0.0]),
    ] {
        assert!(
            SketchPlane::new(origin, x, y, default)
                .unwrap_err()
                .to_string()
                .contains("expected")
        );
    }
    for options in [
        SketchPlaneOptions {
            version: 2,
            ..default
        },
        SketchPlaneOptions {
            unit_deviation: -1.0,
            ..default
        },
        SketchPlaneOptions {
            unit_deviation: f64::NAN,
            ..default
        },
        SketchPlaneOptions {
            angular_deviation: 1.0,
            ..default
        },
        SketchPlaneOptions {
            angular_deviation: f64::INFINITY,
            ..default
        },
    ] {
        assert!(
            SketchPlane::new([0.0; 3], [1.0, 0.0, 0.0], [0.0, 1.0, 0.0], options)
                .unwrap_err()
                .to_string()
                .contains("options")
        );
    }
}
#[test]
fn metric_axis_checks_do_not_normalize_or_repair_saved_values() {
    let strict = SketchPlaneOptions {
        version: 1,
        unit_deviation: 0.0,
        angular_deviation: 0.0,
    };
    assert!(SketchPlane::new([0.0; 3], [1.0, 0.0, 0.0], [0.0, 1.0, 0.0], strict).is_ok());
    let x = [1.0 + 1e-10, 0.0, 0.0];
    let y = [1e-10, 1.0, 0.0];
    assert!(SketchPlane::new([0.0; 3], x, y, strict).is_err());
    let frame = SketchPlane::new([0.0; 3], x, y, Default::default()).unwrap();
    assert_eq!(frame.recipe().x_axis, x);
    assert_eq!(frame.recipe().y_axis, y);
    close(frame.normal(), [0.0, 0.0, 1.0]);
    assert!(
        SketchPlane::from_recipe(
            SketchPlaneRecipe {
                origin: [0.0; 3],
                x_axis: x,
                y_axis: y
            },
            strict
        )
        .is_err()
    );
}
#[test]
fn nonfinite_queries_and_unrepresentable_results_are_named_errors() {
    let plane = SketchPlane::xy();
    assert!(plane.to_model([f64::NAN, 1.0]).is_err());
    assert!(plane.to_sketch([0.0, 1.0, f64::INFINITY]).is_err());
    let frame = SketchPlane::new(
        [f64::MAX, 0.0, 0.0],
        [1.0, 0.0, 0.0],
        [0.0, 1.0, 0.0],
        Default::default(),
    )
    .unwrap();
    assert!(
        frame
            .to_model([f64::MAX, 0.0])
            .unwrap_err()
            .to_string()
            .contains("result")
    );
    assert!(
        frame
            .to_sketch([-f64::MAX, 0.0, 0.0])
            .unwrap_err()
            .to_string()
            .contains("displacement")
    );
}
#[test]
fn frame_transforms_preserve_working_units_at_extreme_coordinate_scales() {
    for scale in [1e-200, 1e-100, 1.0, 1e100, 1e200] {
        let frame = SketchPlane::new(
            [10.0 * scale, -20.0 * scale, 30.0 * scale],
            [0.6, 0.8, 0.0],
            [0.0, 0.0, 1.0],
            Default::default(),
        )
        .unwrap();
        let model = frame.to_model([2.0 * scale, 3.0 * scale]).unwrap();
        close(model.map(|v| v / scale), [11.2, -18.4, 33.0]);
        close(
            frame.to_sketch(model).unwrap().map(|v| v / scale),
            [2.0, 3.0],
        );
    }
}
fn numbers<const N: usize>(text: &str) -> [f64; N] {
    text.split(',')
        .map(|value| value.parse().unwrap())
        .collect::<Vec<_>>()
        .try_into()
        .unwrap()
}
#[test]
fn placement_transforms_and_normals_match_the_fixed_go_baseline() {
    let planes = [
        SketchPlane::xy(),
        SketchPlane::xz(),
        SketchPlane::yz(),
        plane(),
    ];
    for fixture in include_str!("../fixtures/go-sketch-planes.tsv").lines() {
        let fields: Vec<&str> = fixture.split('\t').collect();
        let plane = planes[fields[0].parse::<usize>().unwrap()];
        close(plane.to_model([2.0, 3.0]).unwrap(), numbers(fields[1]));
        close(
            plane.to_sketch([2.0, 3.0, 7.0]).unwrap(),
            numbers(fields[2]),
        );
        close(plane.normal(), numbers(fields[3]));
    }
}
#[test]
fn loose_metric_options_still_refuse_undefined_normals_and_measure_actual_axis_angle() {
    let loose = SketchPlaneOptions {
        version: 1,
        unit_deviation: 0.9,
        angular_deviation: 0.99,
    };
    assert!(SketchPlane::new([0.0; 3], [0.1, 0.0, 0.0], [0.1, 0.0, 0.0], loose).is_err());
    let strict_angle = SketchPlaneOptions {
        angular_deviation: 0.1,
        ..loose
    };
    assert!(SketchPlane::new([0.0; 3], [0.1, 0.0, 0.0], [0.1, 0.1, 0.0], strict_angle).is_err());
    let frame = SketchPlane::new([0.0; 3], [0.1, 0.0, 0.0], [0.0, 0.1, 0.0], strict_angle).unwrap();
    close(frame.normal(), [0.0, 0.0, 1.0]);
}
