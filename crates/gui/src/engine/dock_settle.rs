// SPDX-License-Identifier: GPL-2.0-only
//! Settle dock closures before presenting a frame without replaying input.
use super::{GuiFrameOutput, GuiFrameRequest};

pub(super) fn settle(
    request: &GuiFrameRequest,
    mut draw: impl FnMut(&GuiFrameRequest) -> Result<(GuiFrameOutput, bool), String>,
) -> Result<GuiFrameOutput, String> {
    let (mut output, mut missing) = draw(request)?;
    if !missing {
        return Ok(output);
    }
    let mut redraw = request.clone();
    redraw.events.clear();
    for _ in 0..8 {
        let (next, more) = draw(&redraw)?;
        output = merge(output, next);
        missing = more;
        if !missing {
            return Ok(output);
        }
    }
    Err("dock layout did not settle after 8 redraws; expected stable tab closure".into())
}

fn merge(mut previous: GuiFrameOutput, mut next: GuiFrameOutput) -> GuiFrameOutput {
    previous.actions.append(&mut next.actions);
    next.actions = previous.actions;
    previous
        .parameter_actions
        .append(&mut next.parameter_actions);
    next.parameter_actions = previous.parameter_actions;
    next.window_action = previous.window_action.or(next.window_action);
    next.capture.pointer |= previous.capture.pointer;
    next.capture.keyboard |= previous.capture.keyboard;
    next.capture.text |= previous.capture.text;
    next.capture.popup |= previous.capture.popup;
    next.activity.queued_input |= previous.activity.queued_input;
    next.activity.pending_layout |= previous.activity.pending_layout;
    next.activity.pending_navigation |= previous.activity.pending_navigation;
    next.activity.pending_animation |= previous.activity.pending_animation;
    next.activity.interaction |= previous.activity.interaction;
    next
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{GuiCommandState, GuiFrameEngine, GuiInputEvent, GuiShellAction};

    struct FrameProbe {
        template: GuiFrameOutput,
        late_closes: usize,
        event_counts: Vec<usize>,
    }
    impl FrameProbe {
        fn draw(&mut self, request: &GuiFrameRequest) -> Result<(GuiFrameOutput, bool), String> {
            self.event_counts.push(request.events.len());
            let mut output = self.template.clone();
            output.actions.clear();
            if self.event_counts.len() == 1 {
                output.actions.push(GuiShellAction::ToggleCenterPointFormat);
            }
            Ok((output, self.event_counts.len() <= self.late_closes))
        }
    }
    fn request() -> GuiFrameRequest {
        GuiFrameRequest {
            extent: [640, 360],
            events: vec![GuiInputEvent::PointerMoved([10., 10.])],
            commands: GuiCommandState::default(),
            status: None,
            delta_seconds: 1. / 60.,
            framebuffer_scale: [1., 1.],
        }
    }
    fn probe(request: &GuiFrameRequest, late_closes: usize) -> FrameProbe {
        let mut engine = GuiFrameEngine::new().unwrap();
        FrameProbe {
            template: engine.frame(request).unwrap(),
            late_closes,
            event_counts: Vec::new(),
        }
    }
    #[test]
    fn normal_input_requires_one_draw() {
        let request = request();
        let mut probe = probe(&request, 0);
        let output = settle(&request, |frame| probe.draw(frame)).unwrap();
        assert_eq!(probe.event_counts, [1]);
        assert_eq!(output.actions, [GuiShellAction::ToggleCenterPointFormat]);
    }
    #[test]
    fn late_close_redraw_keeps_actions_without_replaying_input() {
        let request = request();
        let mut probe = probe(&request, 1);
        let output = settle(&request, |frame| probe.draw(frame)).unwrap();
        assert_eq!(probe.event_counts, [1, 0]);
        assert_eq!(output.actions, [GuiShellAction::ToggleCenterPointFormat]);
    }
    #[test]
    fn idle_frames_do_not_require_internal_redraws() {
        let mut request = request();
        request.events.clear();
        let mut engine = GuiFrameEngine::new().unwrap();
        for _ in 0..8 {
            engine.frame(&request).unwrap();
        }
        let start = std::time::Instant::now();
        let mut draws = 0;
        for _ in 0..20 {
            settle(&request, |frame| {
                draws += 1;
                engine.draw_controls(frame)
            })
            .unwrap();
        }
        eprintln!("20 idle frames: {:?}, {draws} draws", start.elapsed());
        assert_eq!(draws, 20);
    }
    #[test]
    fn unstable_docking_has_a_redraw_bound() {
        let request = request();
        let mut probe = probe(&request, usize::MAX);
        let error = settle(&request, |frame| probe.draw(frame)).unwrap_err();
        assert!(error.contains("8 redraws"));
        assert_eq!(probe.event_counts, [1, 0, 0, 0, 0, 0, 0, 0, 0]);
    }
}
