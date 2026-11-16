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
