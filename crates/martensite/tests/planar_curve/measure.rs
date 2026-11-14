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
    assert!(measure.conic.is_none());
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

#[test]
fn rotated_ellipses_return_converged_length_and_canonical_axis_properties() {
    use oblikovati_model::sketch::SketchEllipse2D;
    for axis in [[1., 0.], [0.6, 0.8]] {
        let mut sketch = Sketch2D::default();
        let center = SketchEntityId(10);
        let entity = SketchEntityId(1);
        sketch.add_point(center, [0.2, -0.3]).unwrap();
        sketch
            .add_ellipse(SketchEllipse2D::new(entity, center, axis, [0.8, 0.4]).unwrap())
            .unwrap();
        let measure = measure_sketch_curve(&sketch, entity).unwrap();
        assert!(
            (measure.length - 3.87537928821907).abs() < 1e-12,
            "{measure:?}"
        );
        let conic = measure.conic.unwrap();
        assert_eq!(
            conic.curve_type,
            oblikovati_api::types::ELLIPSE_FULL_CURVE2D
        );
        assert_eq!(conic.center, [0.2, -0.3]);
        assert_eq!(conic.radii, [0.8, 0.4]);
        assert_eq!(conic.sweep, std::f64::consts::TAU);
    }
}
