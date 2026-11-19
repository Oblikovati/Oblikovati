// SPDX-License-Identifier: GPL-2.0-only
//! Origin datum display sizing uses scene bounds without changing model geometry (#4139).
use super::*;
use oblikovati_model::sketch::SketchPlane;
use std::time::Duration;

struct FakeOriginWindow {
    bounds: Option<[[f64; 3]; 2]>,
}
impl HeadWindow for FakeOriginWindow {
    fn presentation_bounds(&self) -> Option<[[f64; 3]; 2]> {
        self.bounds
    }
    fn model_pixel(&self, point: [f64; 3]) -> Result<[i32; 2], String> {
        Ok([point[0] as i32, point[1] as i32])
    }
    fn sketch_pixel(&self, _: SketchPlane, point: [f64; 2]) -> Result<[i32; 2], String> {
        Ok(point.map(|value| value as i32))
    }
    fn events(&mut self, _: usize) -> Result<Vec<oblikovati_viewport::ViewportEvent>, String> {
        Ok(vec![])
    }
    fn present(&mut self) -> Result<crate::HeadPresentation, String> {
        Ok(crate::HeadPresentation::Completed)
    }
    fn wait(&mut self, _: Option<Duration>) -> Result<bool, String> {
        Ok(false)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}

#[test]
fn automatic_extent_tracks_scene_and_keeps_fixed_projection_separate() {
    let window = FakeOriginWindow {
        bounds: Some([[-20., -5., 0.], [30., 8., 2.]]),
    };
    let mut state = oblikovati_gui::GuiCommandState::default();
    state.document = Some(1);
    state.authoring = oblikovati_gui::GuiAuthoringContext::Part;
    let planes = origin_plane_outlines(&window, &state);
    assert_eq!(planes[0].corners[0], [-6, -6]);
    assert_eq!(planes[0].automatic_corners[0], [-33, -33]);
    let axes = origin_axis_lines(&window, &state);
    assert_eq!(axes[0].ends, [[-6, 0], [6, 0]]);
    assert_eq!(axes[0].automatic_ends, [[-33, 0], [33, 0]]);
}

#[test]
fn empty_small_or_nonfinite_bounds_keep_a_readable_default() {
    assert_eq!(automatic_origin_reach(None), ORIGIN_PLANE_REACH);
    assert_eq!(
        automatic_origin_reach(Some([[0.; 3], [1.; 3]])),
        ORIGIN_PLANE_REACH
    );
    assert_eq!(
        automatic_origin_reach(Some([[0.; 3], [f64::INFINITY; 3]])),
        ORIGIN_PLANE_REACH
    );
    assert_eq!(
        automatic_origin_reach(Some([[0.; 3], [f64::NAN, 20., 0.]])),
        ORIGIN_PLANE_REACH
    );
}
