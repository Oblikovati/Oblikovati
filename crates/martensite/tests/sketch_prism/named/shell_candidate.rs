// SPDX-License-Identifier: GPL-2.0-only
//! Independent shell result and semantic history checks (#4106).
use super::*;
use oblikovati_martensite::NamedSolid;

#[test]
fn inward_shell_keeps_input_and_names_every_result_face() {
    let history = owner(cube_document());
    let source = certified(&history);
    let opening = key("end", &[1]);
    let candidate = source
        .shell_candidate(&history, document(), std::slice::from_ref(&opening), 0.1)
        .unwrap();
    assert!((candidate.volume().unwrap() - 0.424).abs() < 1e-8);
    let result = candidate
        .name_topology(&history, document(), FeatureId(2))
        .unwrap();
    let body = NamedSolid::Derived(Box::new(result));
    let face_keys = body.face_reference_keys(&history, document()).unwrap();
    let mut areas: Vec<_> = face_keys
        .iter()
        .map(|key| body.face_area(&history, document(), key).unwrap())
        .collect();
    areas.sort_by(f64::total_cmp);
    let expected = [0.36, 0.64, 0.72, 0.72, 0.72, 0.72, 1., 1., 1., 1., 1.];
    assert_eq!(areas.len(), expected.len(), "face areas {areas:?}");
    for (area, expected) in areas.iter().zip(expected) {
        assert!(
            (area - expected).abs() < 1e-8,
            "face area {area}, expected {expected}"
        );
    }
    let mut loops: Vec<_> = face_keys
        .iter()
        .map(|key| body.face_loop_count(&history, document(), key).unwrap())
        .collect();
    loops.sort_unstable();
    assert_eq!(loops, [1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2]);
    let source_body = source.body_reference_key(&history, document()).unwrap();
    assert!(
        (source
            .physical_properties(&history, document(), source_body)
            .unwrap()
            .geometry
            .volume
            - 1.)
            .abs()
            < 1e-8
    );
}

#[test]
fn shell_refuses_result_when_kernel_exceeds_resolution_budget() {
    let history = owner(cube_document());
    let mut strict = certified_resolution();
    strict.maximum_tolerance = 1e-7;
    let source = oblikovati_martensite::retain_certified_named_sketch_prism(
        &history,
        document(),
        FeatureId(1),
        strict,
    )
    .unwrap();
    let opening = key("end", &[1]);
    let error = source
        .shell_candidate(&history, document(), &[opening], 0.1)
        .err()
        .expect("expected shell exceeding maximum_tolerance budget to be refused");
    assert!(error.contains("shell vertex tolerance"), "{error}");
}

#[test]
fn shell_declines_invalid_openings_before_building_geometry() {
    let history = owner(cube_document());
    let source = certified(&history);
    let opening = key("end", &[1]);
    for (faces, thickness) in [
        (vec![], 0.1),
        (vec![opening.clone(), opening.clone()], 0.1),
        (vec![opening.clone()], 0.),
        (vec![opening.clone()], f64::NAN),
        (vec![key("body", &[1])], 0.1),
    ] {
        assert!(
            source
                .shell_candidate(&history, document(), &faces, thickness)
                .is_err()
        );
    }
}
