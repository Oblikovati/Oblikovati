// SPDX-License-Identifier: GPL-2.0-only
//! Ordered Measure picks retain the first relevant projection error.
use super::*;
use crate::{HeadPresentation, HeadWindow};
use std::time::Duration;

struct ProjectedMeasureWindow {
    projections: Vec<Result<[i32; 2], String>>,
}
impl HeadWindow for ProjectedMeasureWindow {
    fn sketch_pixel_results(
        &self,
        _: SketchPlane,
        _: &[[f64; 2]],
    ) -> Result<Vec<Result<[i32; 2], String>>, String> {
        Ok(self.projections.clone())
    }
    fn events(&mut self, _: usize) -> Result<Vec<ViewportEvent>, String> {
        Ok(Vec::new())
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        Ok(HeadPresentation::Completed)
    }
    fn wait(&mut self, _: Option<Duration>) -> Result<bool, String> {
        Ok(false)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}
#[test]
fn earlier_hit_wins_over_a_later_projection_failure() {
    let window = ProjectedMeasureWindow {
        projections: vec![Ok([0, 0]), Err("later projection".into())],
    };
    let positions = vec![Ok([1., 2.]), Ok([3., 4.])];
    assert_eq!(
        first_near(SketchPlane::xy(), [0, 0], 4., positions, &window).unwrap(),
        Some([1., 2.])
    );
}
#[test]
fn relevant_projection_failure_is_reported_before_a_later_hit() {
    let window = ProjectedMeasureWindow {
        projections: vec![Err("first projection".into()), Ok([0, 0])],
    };
    let positions = vec![Ok([1., 2.]), Ok([3., 4.])];
    assert_eq!(
        first_near(SketchPlane::xy(), [0, 0], 4., positions, &window).unwrap_err(),
        "first projection"
    );
}
#[test]
fn malformed_result_count_is_refused() {
    let window = ProjectedMeasureWindow {
        projections: vec![],
    };
    let error = first_near(SketchPlane::xy(), [0, 0], 4., vec![Ok([1., 2.])], &window).unwrap_err();
    assert!(error.contains("expected 1"), "{error}");
}

#[test]
fn earlier_hit_wins_over_a_later_geometry_failure() {
    let window = ProjectedMeasureWindow {
        projections: vec![Ok([0, 0])],
    };
    let positions = vec![Ok([1., 2.]), Err("later geometry".into())];
    assert_eq!(
        first_near(SketchPlane::xy(), [0, 0], 4., positions, &window).unwrap(),
        Some([1., 2.])
    );
}

#[test]
fn indexed_midpoints_use_hidden_defining_points() {
    use oblikovati_model::sketch::{Sketch2D, SketchEntityId as Id};
    let mut sketch = Sketch2D::default();
    for (id, position) in [(1, [0., 0.]), (2, [3., 4.]), (3, [-4., 3.])] {
        sketch.add_defining_point(Id(id), position).unwrap();
    }
    sketch.add_line(Id(4), [Id(1), Id(2)]).unwrap();
    sketch.add_arc(Id(5), Id(1), [Id(2), Id(3)], true).unwrap();
    let index = sketch.read_index();
    assert_eq!(line_middle(&index, &sketch.lines()[0]).unwrap(), [1.5, 2.]);
    let middle = arc_middle(&index, &sketch.arcs()[0]).unwrap();
    assert!((middle[0] + 2.0_f64.sqrt() / 2.).abs() < 1e-12);
    assert!((middle[1] - 7.0 / 2.0_f64.sqrt()).abs() < 1e-12);
}

#[test]
fn indexed_line_midpoints_match_linear_lookup() {
    let sketch = midpoint_benchmark_sketch(200);
    let index = sketch.read_index();
    for line in sketch.lines() {
        let [a, b] = line.endpoints.map(|id| sketch.point(id).unwrap().position);
        let expected = [(a[0] + b[0]) * 0.5, (a[1] + b[1]) * 0.5];
        assert_eq!(line_middle(&index, line).unwrap(), expected);
    }
}

#[test]
#[ignore = "compare midpoint lookup in release mode"]
fn measure_midpoint_lookup_scaling() {
    for count in [200_u64, 1000] {
        let sketch = midpoint_benchmark_sketch(count);
        let mut linear = Vec::new();
        let mut indexed = Vec::new();
        for sample in 0..140 {
            let (first, second) = midpoint_lookup_pair(&sketch);
            if sample >= 20 {
                linear.push(first);
                indexed.push(second);
            }
        }
        linear.sort();
        indexed.sort();
        eprintln!(
            "midpoint lookup {count} lines: linear p50={:?} p95={:?}; indexed p50={:?} p95={:?}",
            linear[60], linear[113], indexed[60], indexed[113]
        );
    }
}

fn midpoint_benchmark_sketch(count: u64) -> oblikovati_model::sketch::Sketch2D {
    use oblikovati_model::sketch::{Sketch2D, SketchEntityId as Id};
    let mut sketch = Sketch2D::default();
    for i in 0..=count {
        sketch
            .add_defining_point(Id(i + 1), [i as f64, (i % 2) as f64])
            .unwrap();
    }
    for i in 0..count {
        sketch
            .add_line(Id(count + 2 + i), [Id(i + 1), Id(i + 2)])
            .unwrap();
    }
    sketch
}

fn midpoint_lookup_pair(sketch: &oblikovati_model::sketch::Sketch2D) -> (Duration, Duration) {
    let start = std::time::Instant::now();
    for line in sketch.lines() {
        let [a, b] = line.endpoints.map(|id| sketch.point(id).unwrap().position);
        std::hint::black_box([(a[0] + b[0]) * 0.5, (a[1] + b[1]) * 0.5]);
    }
    let linear = start.elapsed();
    let start = std::time::Instant::now();
    let index = sketch.read_index();
    for line in sketch.lines() {
        std::hint::black_box(line_middle(&index, line).unwrap());
    }
    (linear, start.elapsed())
}
