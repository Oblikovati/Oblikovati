// SPDX-License-Identifier: GPL-2.0-only
//! Resolve ribbon pick ownership even when viewport selection fails (#4099).
use super::{HeadWindow, HeadWork};
use oblikovati_gui::{GuiCommandState, GuiModelPickCue};

pub(super) fn project_cues(window: &dyn HeadWindow, commands: &mut GuiCommandState) {
    let shared = commands
        .face_toolbar_anchor
        .as_ref()
        .zip(commands.model_pick_cue.as_ref())
        .is_some_and(|(toolbar, cue)| toolbar.position == cue.position);
    project_cue(window, &mut commands.face_toolbar_anchor);
    project_extrude_handle(window, commands);
    if shared {
        commands
            .model_pick_cue
            .clone_from(&commands.face_toolbar_anchor);
    } else {
        project_cue(window, &mut commands.model_pick_cue);
    }
}

fn project_extrude_handle(window: &dyn HeadWindow, commands: &mut GuiCommandState) {
    let Some(handle) = commands.extrude_handle.as_mut() else {
        return;
    };
    let [start, end] = handle.axis;
    let projected = window
        .model_pixel(start)
        .ok()
        .zip(window.model_pixel(end).ok());
    handle.pixels = projected.map(|(start, end)| [start, end]);
    if let Ok(camera) = window.view_camera()
        && camera.perspective
    {
        let forward: [f64; 3] = std::array::from_fn(|i| camera.target[i] - camera.eye[i]);
        let depth = |point: [f64; 3]| {
            forward
                .into_iter()
                .enumerate()
                .map(|(i, along)| along * (point[i] - camera.eye[i]))
                .sum::<f64>()
        };
        let start_depth = depth(start);
        let end_depth = depth(end);
        if start_depth > 0. && end_depth > 0. {
            handle.depth_ratio = end_depth / start_depth;
        } else {
            handle.pixels = None;
        }
    }
}

fn project_cue(window: &dyn HeadWindow, cue: &mut Option<GuiModelPickCue>) {
    if let Some(anchor) = cue {
        match window.model_pixel(anchor.position) {
            Ok(pixel) => anchor.pixel = pixel,
            Err(_) => *cue = None,
        }
    }
}

pub(super) fn dispatch(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    document: u64,
    pixel: [i32; 2],
    toggle: bool,
) -> Result<bool, String> {
    let result = select(window, owner, document, pixel, toggle);
    if !toggle && matches!(result, Ok(true)) {
        window.resolve_extrusion_model_pick(&mut || {
            let state = owner.gui_command_state().ok()?;
            let [(key, _)] = state.selected_face_appearances.as_slice() else {
                return None;
            };
            Some((document, key.clone()))
        });
    }
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
    let depth = depth_window(window)?;
    owner.click_model_with_modifiers(document, line, depth, toggle)
}
pub(super) fn depth_window(window: &mut dyn HeadWindow) -> Result<Option<f64>, String> {
    let Some([_, height]) = window.window_capture_extent()? else {
        return Ok(None);
    };
    crate::part_scene::pick_candidates::depth_window(window.view_camera()?.scale, height).map(Some)
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
        projections: std::cell::Cell<u8>,
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
        fn model_pixel(&self, point: [f64; 3]) -> Result<[i32; 2], String> {
            self.projections.set(self.projections.get() + 1);
            if point[0] < 0. {
                return Err("test projection refused".into());
            }
            Ok([point[0] as i32, point[1] as i32])
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

    #[test]
    fn face_label_and_toolbar_share_projection_and_decline_together() {
        let window = RefusingProjectionWindow::default();
        let mut commands = GuiCommandState::default();
        for (index, position) in [[3., 4., 5.], [-1., 4., 5.]].into_iter().enumerate() {
            let cue = GuiModelPickCue {
                pixel: [0; 2],
                position,
            };
            commands.face_toolbar_anchor = Some(cue.clone());
            commands.model_pick_cue = Some(cue);
            project_cues(&window, &mut commands);
            assert_eq!(usize::from(window.projections.get()), index + 1);
            assert_eq!(commands.face_toolbar_anchor, commands.model_pick_cue);
            assert_eq!(
                commands.model_pick_cue.as_ref().map(|cue| cue.pixel),
                (index == 0).then_some([3, 4])
            );
        }
    }
}
