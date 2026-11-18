// SPDX-License-Identifier: GPL-2.0-only
//! Resolve ribbon pick ownership even when viewport selection fails (#4099).
use super::{HeadWindow, HeadWork};

pub(super) fn dispatch(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    document: u64,
    pixel: [i32; 2],
    toggle: bool,
) -> Result<bool, String> {
    let result = select(window, owner, document, pixel, toggle);
    if !window.has_pending_ribbon_model_pick() {
        return result;
    }
    let resolved = result.and_then(|changed| {
        owner
            .gui_command_state()
            .map(|state| (changed, state.model_pick_cue.is_some()))
    });
    window.resolve_ribbon_model_pick(resolved.as_ref().is_ok_and(|(_, face)| *face));
    resolved.map(|(changed, _)| changed)
}
fn select(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    document: u64,
    pixel: [i32; 2],
    toggle: bool,
) -> Result<bool, String> {
    let line = window.view_line(pixel)?;
    let depth = match window.window_capture_extent()? {
        Some([_, height]) => Some(crate::part_scene::pick_candidates::depth_window(
            window.view_camera()?.scale,
            height,
        )?),
        None => None,
    };
    owner.click_model_with_modifiers(document, line, depth, toggle)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{HeadPresentation, HeadWorkStatus};
    use oblikovati_viewport::ViewportEvent;
    use std::time::Duration;

    #[derive(Default)]
    struct RefusingProjectionWindow {
        pending: bool,
        resolutions: Vec<bool>,
    }
    impl HeadWindow for RefusingProjectionWindow {
        fn has_pending_ribbon_model_pick(&self) -> bool {
            self.pending
        }
        fn resolve_ribbon_model_pick(&mut self, face: bool) {
            self.pending = false;
            self.resolutions.push(face);
        }
        fn view_line(&self, _: [i32; 2]) -> Result<[[f64; 3]; 2], String> {
            Err("test projection refused".into())
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
    struct IdleWork;
    impl crate::HeadSketchWork for IdleWork {}
    impl HeadWork for IdleWork {
        fn poll(&mut self, _: usize) -> Result<HeadWorkStatus, String> {
            Ok(Default::default())
        }
        fn close_allowed(&mut self) -> Result<bool, String> {
            Ok(true)
        }
        fn finish(&mut self) -> Result<(), String> {
            Ok(())
        }
    }
    #[test]
    fn projection_error_cancels_only_the_pending_pick() {
        let mut window = RefusingProjectionWindow {
            pending: true,
            ..Default::default()
        };
        assert_eq!(
            dispatch(&mut window, &mut IdleWork, 1, [3, 4], false).unwrap_err(),
            "test projection refused"
        );
        assert_eq!(window.resolutions, [false]);
        assert_eq!(
            dispatch(&mut window, &mut IdleWork, 1, [3, 4], false).unwrap_err(),
            "test projection refused"
        );
        assert_eq!(window.resolutions, [false]);
    }
}
