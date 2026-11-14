// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn polygon(points: [[f64; 2]; 4], edges: [[u64; 2]; 4]) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (index, point) in points.into_iter().enumerate() {
        sketch
            .add_point(SketchEntityId(index as u64 + 1), point)
            .unwrap();
    }
    for (index, endpoints) in edges.into_iter().enumerate() {
        sketch
            .add_line(
                SketchEntityId(index as u64 + 10),
                endpoints.map(SketchEntityId),
            )
            .unwrap();
    }
    sketch
}
#[test]
fn accepts_only_a_closed_axis_aligned_loop() {
    let points = [[0., 0.], [4., 0.], [4., 3.], [0., 3.]];
    assert!(is_axis_aligned_closed(&polygon(
        points,
        [[1, 2], [2, 3], [3, 4], [4, 1]]
    )));
    assert!(!is_axis_aligned_closed(&polygon(
        points,
        [[1, 2], [2, 1], [3, 4], [4, 3]]
    )));
    let flat = [[0., 0.], [1., 0.], [2., 0.], [3., 0.]];
    assert!(!is_axis_aligned_closed(&polygon(
        flat,
        [[1, 2], [2, 3], [3, 4], [4, 1]]
    )));
    let tilted = [[0., 0.], [4., 1.], [4., 3.], [0., 3.]];
    assert!(!is_axis_aligned_closed(&polygon(
        tilted,
        [[1, 2], [2, 3], [3, 4], [4, 1]]
    )));
}
