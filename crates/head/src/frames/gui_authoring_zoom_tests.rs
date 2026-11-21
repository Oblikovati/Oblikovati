// SPDX-License-Identifier: GPL-2.0-only
//! Verify selected zoom does not leave a partial camera on failure.
use super::*;
use crate::HeadPresentation;
use oblikovati_viewport::ViewportEvent;
use std::time::Duration;

#[derive(Default)]
struct RefusingZoomWindow {
    restored: bool,
}
impl HeadWindow for RefusingZoomWindow {
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
    fn navigate_view(
        &mut self,
        _: oblikovati_viewport::ViewportNavigation,
    ) -> Result<bool, String> {
        Ok(true)
    }
    fn set_view_camera(&mut self, _: oblikovati_viewport::ViewportCamera) -> Result<bool, String> {
        Err("camera refused".into())
    }
    fn previous_view(&mut self) -> Result<bool, String> {
        self.restored = true;
        Ok(true)
    }
}
#[test]
fn failed_selected_zoom_restores_the_previous_view() {
    let mut window = RefusingZoomWindow::default();
    let camera = oblikovati_viewport::ViewportCamera::default();
    let error = apply_selected_fit(&mut window, camera, camera).unwrap_err();
    assert_eq!(error, "camera refused");
    assert!(window.restored);
}
#[test]
fn empty_selected_face_does_not_touch_the_window() {
    let mut window = RefusingZoomWindow::default();
    assert!(!frame_selected_edges(&mut window, &[], [640, 480]).unwrap());
    assert!(!window.restored);
}
