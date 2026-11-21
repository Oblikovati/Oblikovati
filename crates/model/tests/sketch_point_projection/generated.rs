// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn generated_off_plane_sources_retain_local_coordinates_in_rotated_translated_frames() {
    for index in 0..64 {
        let (snapshot, expected) = generated_snapshot(index);
        let before = snapshot.sketches.recipe();
        let result = project(&snapshot, SOURCE);
        for (actual, expected) in result.local_position.into_iter().zip(expected) {
            let bound = 64.0 * f64::EPSILON * (1.0 + f64::from(index));
            assert!(
                (actual - expected).abs() <= bound,
                "index {index}: {actual} != {expected}"
            );
        }
        assert_eq!(before, snapshot.sketches.recipe());
        assert_eq!(result, project(&snapshot, SOURCE));
    }
}
fn generated_snapshot(index: u32) -> (DocumentSnapshot, [f64; 2]) {
    let angle = f64::from(index) * 0.137;
    let (sine, cosine) = angle.sin_cos();
    let origin = [f64::from(index), -3.0, 11.0];
    let axes = [[cosine, sine, 0.0], [-sine, cosine, 0.0]];
    let expected = [-2.25, 7.5];
    let source = model_source(origin, axes, expected, f64::from(index));
    let plane = SketchPlane::new(origin, axes[0], axes[1], Default::default()).unwrap();
    let mut snapshot = snapshot();
    replace(&mut snapshot, 2, WorkGeometryDefinition::Fixed(plane));
    replace(
        &mut snapshot,
        1,
        WorkGeometryDefinition::Point(source.map(DatumLength::Constant)),
    );
    (snapshot, expected)
}
fn model_source(origin: [f64; 3], axes: [[f64; 3]; 2], local: [f64; 2], height: f64) -> [f64; 3] {
    [
        origin[0] + local[0] * axes[0][0] + local[1] * axes[1][0],
        origin[1] + local[0] * axes[0][1] + local[1] * axes[1][1],
        origin[2] + height,
    ]
}
