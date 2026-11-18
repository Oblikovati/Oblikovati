// SPDX-License-Identifier: GPL-2.0-only
//! Exact named rigid motion over detached prism and derived bodies (#4097).
use super::*;
use oblikovati_martensite::{NamedSolid, RigidMotion};

fn moved(
    body: &NamedSolid,
    history: &DocumentEditHistory,
    feature: u64,
    motion: &[RigidMotion],
) -> NamedSolid {
    let candidate = body.rigid_candidate(history, document(), motion).unwrap();
    NamedSolid::Derived(Box::new(
        candidate
            .name_topology(history, document(), FeatureId(feature))
            .unwrap(),
    ))
}
fn near(actual: [f64; 3], expected: [f64; 3]) {
    for (actual, expected) in actual.into_iter().zip(expected) {
        assert!(
            (actual - expected).abs() < 1e-9,
            "coordinate {actual}: expected {expected}"
        );
    }
}
#[test]
fn rigid_motion_retains_every_named_entity_and_analytic_face() {
    let history = owner(cube_document());
    let body = NamedSolid::Prism(Box::new(certified(&history)));
    let faces = body.face_reference_keys(&history, document()).unwrap();
    let lower = body.subentity_reference_keys(&history, document()).unwrap();
    let end = key("end", &[1]);
    let before = body.face_bounds_center(&history, document(), &end).unwrap();
    let result = moved(
        &body,
        &history,
        2,
        &[RigidMotion::Translation([3., -2., 5.])],
    );
    near(
        result
            .face_bounds_center(&history, document(), &end)
            .unwrap(),
        [before[0] + 3., before[1] - 2., before[2] + 5.],
    );
    near(
        body.face_bounds_center(&history, document(), &end).unwrap(),
        before,
    );
    assert_eq!(
        result.face_reference_keys(&history, document()).unwrap(),
        faces
    );
    assert_eq!(
        result
            .subentity_reference_keys(&history, document())
            .unwrap(),
        lower
    );
    for face in &faces {
        assert!(
            (result.face_area(&history, document(), face).unwrap()
                - body.face_area(&history, document(), face).unwrap())
            .abs()
                < 1e-9
        );
        assert_eq!(
            result.face_loop_count(&history, document(), face).unwrap(),
            body.face_loop_count(&history, document(), face).unwrap()
        );
        assert_eq!(
            result
                .face_topology(&history, document(), face)
                .unwrap()
                .surface,
            body.face_topology(&history, document(), face)
                .unwrap()
                .surface
        );
    }
    assert!(
        (result.volume(&history, document()).unwrap() - body.volume(&history, document()).unwrap())
            .abs()
            < 1e-9
    );
    assert_eq!(result.achieved_tolerance(), body.achieved_tolerance());
}
#[test]
fn rotations_and_translations_apply_in_order_and_remain_reusable() {
    let history = owner(cube_document());
    let body = NamedSolid::Prism(Box::new(certified(&history)));
    let face = key("end", &[1]);
    let first = moved(
        &body,
        &history,
        2,
        &[
            RigidMotion::Rotation {
                origin: [0.; 3],
                direction: [0., 0., 1.],
                radians: std::f64::consts::FRAC_PI_2,
            },
            RigidMotion::Translation([3., 0., 0.]),
        ],
    );
    near(
        first
            .face_bounds_center(&history, document(), &face)
            .unwrap(),
        [2.5, 0.5, 1.],
    );
    let second = moved(
        &first,
        &history,
        3,
        &[RigidMotion::Translation([0., 1., 0.])],
    );
    near(
        second
            .face_bounds_center(&history, document(), &face)
            .unwrap(),
        [2.5, 1.5, 1.],
    );
    near(
        body.face_bounds_center(&history, document(), &face)
            .unwrap(),
        [0.5, 0.5, 1.],
    );
}
#[test]
fn rigid_motion_rejects_invalid_inputs_without_changing_the_body() {
    let history = owner(cube_document());
    let body = NamedSolid::Prism(Box::new(certified(&history)));
    for moves in [
        vec![],
        vec![RigidMotion::Translation([f64::NAN, 0., 0.])],
        vec![
            RigidMotion::Translation([f64::MAX, 0., 0.]),
            RigidMotion::Translation([f64::MAX, 0., 0.]),
        ],
        vec![RigidMotion::Rotation {
            origin: [0.; 3],
            direction: [0.; 3],
            radians: 1.,
        }],
        vec![RigidMotion::Rotation {
            origin: [0.; 3],
            direction: [0., 0., 1.],
            radians: f64::INFINITY,
        }],
    ] {
        assert!(
            body.rigid_candidate(&history, document(), &moves).is_err(),
            "unexpectedly accepted {moves:?}"
        );
    }
    assert!((body.volume(&history, document()).unwrap() - 1.).abs() < 1e-9);
}

#[test]
fn rotation_axis_normalization_handles_extreme_finite_scales() {
    let history = owner(cube_document());
    let body = NamedSolid::Prism(Box::new(certified(&history)));
    let face = key("end", &[1]);
    for scale in [f64::MAX, f64::MIN_POSITIVE] {
        let rotated = moved(
            &body,
            &history,
            2,
            &[RigidMotion::Rotation {
                origin: [0.; 3],
                direction: [0., 0., scale],
                radians: std::f64::consts::FRAC_PI_2,
            }],
        );
        near(
            rotated
                .face_bounds_center(&history, document(), &face)
                .unwrap(),
            [-0.5, 0.5, 1.],
        );
    }
}
