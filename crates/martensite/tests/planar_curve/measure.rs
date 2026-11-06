// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn line(sketch: &mut Sketch2D, id: u64, ends: [[f64; 2]; 2]) -> SketchEntityId {
    let points = [id * 10, id * 10 + 1].map(SketchEntityId);
    sketch.add_point(points[0], ends[0]).unwrap();
    sketch.add_point(points[1], ends[1]).unwrap();
    let entity = SketchEntityId(id);
    sketch.add_line(entity, points).unwrap();
    entity
}
fn segments() -> (Sketch2D, [SketchEntityId; 2]) {
    let mut sketch = Sketch2D::default();
    let first = line(&mut sketch, 1, [[0., 0.], [4., 0.]]);
    let second = line(&mut sketch, 2, [[6., 3.], [6., 7.]]);
    (sketch, [first, second])
}
#[test]
fn a_segment_measures_its_length_and_its_ends() {
    let (sketch, [first, _]) = segments();
    let measure = measure_sketch_curve(&sketch, first).unwrap();
    assert!((measure.length - 4.).abs() < 1e-9, "{measure:?}");
    assert_eq!(measure.endpoints, Some([[0., 0.], [4., 0.]]));
    assert!(measure.circular.is_none());
}
#[test]
fn two_segments_report_their_nearest_and_farthest_ends() {
    let (sketch, entities) = segments();
    let range = sketch_curve_distance_range(&sketch, entities).unwrap();
    assert!(
        (range.minimum.distance - 13f64.sqrt()).abs() < 1e-9,
        "{range:?}"
    );
    assert_eq!(range.minimum.points, [[4., 0.], [6., 3.]]);
    assert!(
        (range.maximum.distance - 85f64.sqrt()).abs() < 1e-9,
        "{range:?}"
    );
    assert_eq!(range.maximum.points, [[0., 0.], [6., 7.]]);
}
#[test]
fn parallel_segments_fall_back_to_their_ends() {
    let mut sketch = Sketch2D::default();
    let first = line(&mut sketch, 1, [[0., 0.], [4., 0.]]);
    let second = line(&mut sketch, 2, [[1., 2.], [3., 2.]]);
    let range = sketch_curve_distance_range(&sketch, [first, second]).unwrap();
    assert!((range.minimum.distance - 2.).abs() < 1e-9, "{range:?}");
    assert!(
        (range.maximum.distance - 13f64.sqrt()).abs() < 1e-9,
        "{range:?}"
    );
}
#[test]
fn a_point_inside_a_segments_span_is_nearest_at_its_foot() {
    let (sketch, [first, _]) = segments();
    let range = sketch_point_curve_distance_range(&sketch, [1., 3.], first).unwrap();
    assert!((range.minimum.distance - 3.).abs() < 1e-9, "{range:?}");
    assert!(
        (range.maximum.distance - 18f64.sqrt()).abs() < 1e-9,
        "{range:?}"
    );
}
