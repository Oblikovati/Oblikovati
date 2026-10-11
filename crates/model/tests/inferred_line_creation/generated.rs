// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::PRIORITY_PARALLEL_PERPENDICULAR;
#[test]
fn rotated_scaled_reversed_parallel_inputs_use_the_same_owned_relation_path() {
    for index in 0..16 {
        let angle = index as f64 * std::f64::consts::TAU / 16.0;
        for scale in [0.1, 1.0, 100.0] {
            for reverse in [false, true] {
                assert_generated_parallel(angle, scale, reverse);
            }
        }
    }
}
fn assert_generated_parallel(angle: f64, scale: f64, reverse: bool) {
    let delta = [scale * angle.cos(), scale * angle.sin()];
    let mut history = parallel_owner(delta);
    let start = [10.0 * scale, 10.0 * scale];
    let end = [start[0] + delta[0], start[1] + delta[1]];
    let endpoints = if reverse { [end, start] } else { [start, end] };
    let report = create(&mut history, input(endpoints));
    let direction = report.direction.unwrap();
    assert_eq!(direction.inference.kind, INFER_PARALLEL);
    assert_eq!(direction.inference.target, Some(SketchEntityId(3)));
    assert_direction_relation(geometry(&history), report.primitive.entity, INFER_PARALLEL);
}
fn parallel_owner(delta: [f64; 2]) -> DocumentEditHistory {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0.0; 2]).unwrap();
    sketch.add_point(SketchEntityId(2), delta).unwrap();
    sketch
        .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    let mut history = owner(sketch);
    history
        .refresh(|snapshot| -> Result<(), String> {
            let mut settings = default_sketch_settings();
            settings.constraint_priority = PRIORITY_PARALLEL_PERPENDICULAR;
            snapshot.sketch_settings = Some(settings);
            Ok(())
        })
        .unwrap();
    history
}
