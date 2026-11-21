// SPDX-License-Identifier: GPL-2.0-only
//! Checked point batches for context-menu selection (#4035).
use super::*;
use crate::{HeadPresentation, HeadWindow};
use std::cell::Cell;
use std::time::Duration;

struct ContextBatchWindow {
    calls: Cell<usize>,
    short_batch: bool,
    fail_batch: bool,
}
impl HeadWindow for ContextBatchWindow {
    fn sketch_pixel(&self, _: SketchPlane, _: [f64; 2]) -> Result<[i32; 2], String> {
        Err("unexpected scalar projection".into())
    }
    fn sketch_pixels(&self, _: SketchPlane, points: &[[f64; 2]]) -> Result<Vec<[i32; 2]>, String> {
        self.calls.set(self.calls.get() + 1);
        if self.fail_batch {
            return Err("projection failed".into());
        }
        let count = points.len().saturating_sub(usize::from(self.short_batch));
        Ok(points[..count]
            .iter()
            .map(|p| p.map(|v| v as i32))
            .collect())
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
fn fixture() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, position) in [(3, [5., 0.]), (1, [0., 0.]), (2, [-5., 0.]), (4, [7., 0.])] {
        sketch.add_point(SketchEntityId(id), position).unwrap();
    }
    sketch
}
fn window(short_batch: bool, fail_batch: bool) -> ContextBatchWindow {
    ContextBatchWindow {
        calls: Cell::new(0),
        short_batch,
        fail_batch,
    }
}
#[test]
fn context_points_use_one_batch_and_keep_distance_and_id_order() {
    let window = window(false, false);
    let points = context_points(&fixture(), SketchPlane::xy(), [0, 0], &window).unwrap();
    assert_eq!(
        points,
        [
            (0., SketchEntityId(1)),
            (5., SketchEntityId(2)),
            (5., SketchEntityId(3))
        ]
    );
    assert_eq!(window.calls.get(), 1);
}
#[test]
fn context_points_reject_a_short_batch_and_propagate_projection_errors() {
    let short = context_points(&fixture(), SketchPlane::xy(), [0, 0], &window(true, false));
    assert!(short.unwrap_err().contains("expected 4 pixels"));
    let failure = context_points(&fixture(), SketchPlane::xy(), [0, 0], &window(false, true));
    assert_eq!(failure.unwrap_err(), "projection failed");
}

#[test]
fn empty_context_does_not_request_projection() {
    let window = window(false, true);
    let points = context_points(&Sketch2D::default(), SketchPlane::xy(), [0, 0], &window).unwrap();
    assert!(points.is_empty());
    assert_eq!(window.calls.get(), 0);
}
