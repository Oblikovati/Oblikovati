// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::collections::VecDeque;
#[derive(Default)]
struct FakeHeadWindow {
    inputs: VecDeque<Vec<ViewportEvent>>,
    outcomes: VecDeque<Result<HeadPresentation, String>>,
    wakes: VecDeque<Result<bool, String>>,
    calls: Vec<&'static str>,
    event_failure: bool,
    cleanup_failure: bool,
    timeouts: Vec<Option<std::time::Duration>>,
}
impl HeadWindow for FakeHeadWindow {
    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        assert!(limit > 0 && limit <= 64);
        self.calls.push("events");
        if self.event_failure {
            return Err("fake event source disconnected".into());
        }
        Ok(self.inputs.pop_front().unwrap_or_default())
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        self.calls.push("present");
        self.outcomes
            .pop_front()
            .unwrap_or(Ok(HeadPresentation::Completed))
    }
    fn wait(&mut self, timeout: Option<std::time::Duration>) -> Result<bool, String> {
        self.timeouts.push(timeout);
        self.calls.push("wait");
        self.wakes
            .pop_front()
            .expect("expected a planned fake native wake")
    }
    fn finish(&mut self) -> Result<(), String> {
        self.calls.push("finish");
        if self.cleanup_failure {
            return Err("fake validation fault".into());
        }
        Ok(())
    }
}
#[test]
fn bounded_smoke_presents_the_requested_attempts_without_idle_waits() {
    let mut window = FakeHeadWindow::default();
    let summary = run_window(&mut window, HeadOptions { frames: Some(3) }).unwrap();
    assert_eq!(
        summary,
        HeadSummary {
            attempts: 3,
            completed: 3
        }
    );
    let draws = window
        .calls
        .iter()
        .filter(|call| **call == "present")
        .count();
    assert_eq!(draws, 3);
    assert!(!window.calls.contains(&"wait"));
    assert_eq!(window.calls.last(), Some(&"finish"));
}
#[test]
fn idle_window_waits_and_worker_wake_requests_one_new_frame() {
    let mut window = FakeHeadWindow {
        inputs: [vec![], vec![], vec![ViewportEvent::CloseRequested]].into(),
        wakes: [Ok(true), Ok(false)].into(),
        ..Default::default()
    };
    let summary = run_window(&mut window, HeadOptions::default()).unwrap();
    assert_eq!(summary.completed, 2);
    assert_eq!(
        window.calls,
        [
            "events", "present", "wait", "events", "present", "wait", "events", "finish"
        ]
    );
}
#[test]
fn surface_change_schedules_new_input_and_is_not_counted_as_completed() {
    let mut window = FakeHeadWindow {
        outcomes: [
            Ok(HeadPresentation::SurfaceChanged),
            Ok(HeadPresentation::Completed),
        ]
        .into(),
        ..Default::default()
    };
    assert_eq!(
        run_window(&mut window, HeadOptions { frames: Some(2) }).unwrap(),
        HeadSummary {
            attempts: 2,
            completed: 1
        }
    );
    assert!(!window.calls.contains(&"wait"));
}
#[test]
fn close_before_initial_presentation_retires_without_a_draw_or_wait() {
    let mut window = FakeHeadWindow {
        inputs: [vec![ViewportEvent::CloseRequested]].into(),
        ..Default::default()
    };
    assert_eq!(
        run_window(&mut window, HeadOptions::default()).unwrap(),
        HeadSummary::default()
    );
    assert_eq!(window.calls, ["events", "finish"]);
}
#[test]
fn hidden_window_waits_for_remap_and_does_not_present_offscreen() {
    let mut window = FakeHeadWindow {
        inputs: [
            vec![ViewportEvent::Mapped(false)],
            vec![ViewportEvent::Mapped(true)],
            vec![ViewportEvent::CloseRequested],
        ]
        .into(),
        wakes: [Ok(false), Ok(false)].into(),
        ..Default::default()
    };
    let summary = run_window(&mut window, HeadOptions::default()).unwrap();
    assert_eq!(summary.completed, 1);
    assert_eq!(
        window.calls,
        [
            "events", "wait", "events", "present", "wait", "events", "finish"
        ]
    );
}
#[test]
fn hidden_bounded_smoke_declines_instead_of_spinning_or_waiting_forever() {
    let mut window = FakeHeadWindow {
        inputs: [vec![ViewportEvent::Mapped(false)]].into(),
        ..Default::default()
    };
    assert!(
        run_window(&mut window, HeadOptions { frames: Some(3) })
            .unwrap_err()
            .contains("mapped")
    );
    assert_eq!(window.calls, ["events", "finish"]);
}
#[test]
fn renderer_failure_and_cleanup_failure_both_reach_the_caller() {
    let mut window = FakeHeadWindow {
        outcomes: [Err("fake renderer fault".into())].into(),
        cleanup_failure: true,
        ..Default::default()
    };
    let error = run_window(&mut window, HeadOptions { frames: Some(1) }).unwrap_err();
    assert!(error.contains("fake renderer fault") && error.contains("fake validation fault"));
    assert_eq!(window.calls, ["events", "present", "finish"]);
}
#[test]
fn event_failure_and_idle_wait_failure_always_retire_the_window() {
    let mut failed_events = FakeHeadWindow {
        event_failure: true,
        ..Default::default()
    };
    assert!(
        run_window(&mut failed_events, HeadOptions::default())
            .unwrap_err()
            .contains("disconnected")
    );
    assert_eq!(failed_events.calls, ["events", "finish"]);
}
#[test]
fn idle_wait_failure_retires_the_window() {
    let mut failed_wait = FakeHeadWindow {
        wakes: [Err("fake wait fault".into())].into(),
        ..Default::default()
    };
    assert!(
        run_window(&mut failed_wait, HeadOptions::default())
            .unwrap_err()
            .contains("wait fault")
    );
    assert_eq!(failed_wait.calls.last(), Some(&"finish"));
}
#[test]
fn cleanup_failure_is_not_hidden_by_a_successful_smoke_run() {
    let mut window = FakeHeadWindow {
        cleanup_failure: true,
        ..Default::default()
    };
    assert!(
        run_window(&mut window, HeadOptions { frames: Some(1) })
            .unwrap_err()
            .contains("validation fault")
    );
}
#[test]
fn smoke_with_only_changed_surfaces_cannot_report_success() {
    let mut window = FakeHeadWindow {
        outcomes: [Ok(HeadPresentation::SurfaceChanged)].into(),
        ..Default::default()
    };
    assert!(
        run_window(&mut window, HeadOptions { frames: Some(1) })
            .unwrap_err()
            .contains("completed native frame")
    );
}
#[test]
fn focus_exposure_and_resize_each_request_presentation() {
    let mut control = FrameControl::new(HeadOptions::default());
    for event in [
        ViewportEvent::Expose,
        ViewportEvent::Focus(false),
        ViewportEvent::Resized([800, 600]),
    ] {
        control.pending = false;
        control.events(&[event], true);
        assert!(control.pending);
    }
    control.summary.attempts = u64::MAX;
    assert!(
        control
            .presented(HeadPresentation::Completed)
            .unwrap_err()
            .contains("unused u64")
    );
}
#[derive(Default)]
struct FakeHeadWork {
    statuses: VecDeque<Result<crate::HeadWorkStatus, String>>,
    close_answers: VecDeque<bool>,
    calls: Vec<&'static str>,
    fail_finish: bool,
    preview: Option<oblikovati_viewport::ViewportSketchPreview>,
}
impl crate::HeadWork for FakeHeadWork {
    fn sketch_preview(&self) -> Option<oblikovati_viewport::ViewportSketchPreview> {
        self.preview.clone()
    }
    fn poll(&mut self, budget: usize) -> Result<crate::HeadWorkStatus, String> {
        assert_eq!(budget, 32);
        self.calls.push("poll");
        self.statuses
            .pop_front()
            .unwrap_or(Ok(crate::HeadWorkStatus::default()))
    }
    fn close_allowed(&mut self) -> Result<bool, String> {
        self.calls.push("close");
        Ok(self.close_answers.pop_front().unwrap_or(true))
    }
    fn finish(&mut self) -> Result<(), String> {
        self.calls.push("finish");
        if self.fail_finish {
            return Err("fake owner retirement fault".into());
        }
        Ok(())
    }
}
#[test]
fn refused_close_keeps_the_window_alive_until_the_owner_approves_another_request() {
    let mut window = FakeHeadWindow {
        inputs: [
            vec![ViewportEvent::CloseRequested],
            vec![ViewportEvent::CloseRequested],
        ]
        .into(),
        wakes: [Ok(false)].into(),
        ..Default::default()
    };
    let mut owner = FakeHeadWork {
        close_answers: [false, true].into(),
        ..Default::default()
    };
    let summary = run_window_with_owner(&mut window, HeadOptions::default(), &mut owner).unwrap();
    assert_eq!(summary.completed, 1);
    assert_eq!(owner.calls, ["poll", "close", "poll", "close", "finish"]);
    let expected = ["events", "present", "wait", "events", "finish"];
    assert_eq!(window.calls, expected);
}
#[test]
fn queued_work_prevents_idle_wait_between_bounded_owner_batches() {
    let mut window = FakeHeadWindow {
        inputs: [vec![], vec![ViewportEvent::CloseRequested]].into(),
        ..Default::default()
    };
    let mut owner = FakeHeadWork {
        statuses: [Ok(crate::HeadWorkStatus {
            changed: true,
            pending: true,
            wake_after: None,
        })]
        .into(),
        ..Default::default()
    };
    let summary = run_window_with_owner(&mut window, HeadOptions::default(), &mut owner).unwrap();
    assert_eq!(summary.completed, 1);
    assert!(!window.calls.contains(&"wait"));
}
#[test]
fn owner_poll_and_retirement_errors_preserve_window_retirement() {
    let mut window = FakeHeadWindow::default();
    let mut owner = FakeHeadWork {
        statuses: [Err("fake owner drain fault".into())].into(),
        fail_finish: true,
        ..Default::default()
    };
    let error = run_window_with_owner(&mut window, HeadOptions::default(), &mut owner).unwrap_err();
    assert!(error.contains("drain fault") && error.contains("retirement fault"));
    assert_eq!(window.calls, ["events", "finish"]);
    assert_eq!(owner.calls, ["poll", "finish"]);
}
#[test]
fn timed_owner_service_wait_does_not_force_an_unchanged_second_frame() {
    let timeout = std::time::Duration::from_secs(1) / 60;
    let mut owner = FakeHeadWork {
        statuses: [Ok(crate::HeadWorkStatus {
            wake_after: Some(timeout),
            ..Default::default()
        })]
        .into(),
        ..Default::default()
    };
    let mut window = FakeHeadWindow {
        inputs: [vec![], vec![], vec![ViewportEvent::CloseRequested]].into(),
        wakes: [Ok(false), Ok(false)].into(),
        ..Default::default()
    };
    let summary = run_window_with_owner(&mut window, HeadOptions::default(), &mut owner).unwrap();
    assert_eq!(summary.completed, 1);
    assert_eq!(window.timeouts, [Some(timeout), None]);
}

struct FakeSceneWork;
impl HeadWork for FakeSceneWork {
    fn poll(&mut self, _: usize) -> Result<crate::HeadWorkStatus, String> {
        Ok(Default::default())
    }
    fn close_allowed(&mut self) -> Result<bool, String> {
        Ok(true)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
    fn take_scene(&mut self) -> Result<Option<crate::HeadSceneUpdate>, String> {
        Ok(Some(crate::HeadSceneUpdate {
            scene: oblikovati_viewport::ViewportScene {
                background: [0.; 3],
                triangles: Vec::new(),
            },
            status: crate::HeadSceneStatus {
                source: Some((1, 0)),
                visible_bodies: 0,
                health: Vec::new(),
                presentation_declines: Vec::new(),
            },
        }))
    }
}
#[test]
fn an_unsupported_scene_window_refuses_publication_and_retires() {
    let mut window = FakeHeadWindow::default();
    let error = run_window_with_owner(
        &mut window,
        HeadOptions { frames: Some(1) },
        &mut FakeSceneWork,
    )
    .unwrap_err();
    assert!(error.contains("expected a window with model presentation"));
    assert!(!window.calls.contains(&"present"));
    assert_eq!(window.calls.last(), Some(&"finish"));
}

#[test]
fn a_window_without_command_feedback_refuses_to_hide_a_notice() {
    let mut window = FakeHeadWindow::default();
    let cause = window
        .show_command_notice(crate::DesktopCommandNotice::NoActiveDocument)
        .unwrap_err();
    assert!(cause.contains("NoActiveDocument"));
    assert!(cause.contains("expected a window with command feedback"));
}

struct StartupHeadWork {
    polls: usize,
}
impl HeadWork for StartupHeadWork {
    fn startup_ready(&self) -> bool {
        self.polls >= 3
    }
    fn poll(&mut self, _: usize) -> Result<crate::HeadWorkStatus, String> {
        self.polls += 1;
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
fn bounded_frames_start_counting_after_owner_startup() {
    let mut window = FakeHeadWindow::default();
    let mut owner = StartupHeadWork { polls: 0 };
    let summary =
        run_window_with_owner(&mut window, HeadOptions { frames: Some(2) }, &mut owner).unwrap();
    assert_eq!(summary.attempts, 4);
    assert_eq!(owner.polls, 4);
}

#[test]
fn preview_refusal_clears_stale_geometry_and_reports_without_stopping() {
    let packet = oblikovati_viewport::ViewportSketchPreview {
        solve_options: Default::default(),
        plane: oblikovati_model::sketch::SketchPlane::xy(),
        recipe: oblikovati_model::sketch::SketchPlacementRecipe::straight_slot(
            [[0., 0.], [10., 0.]],
            2.,
        )
        .unwrap(),
    };
    assert!(
        FakeHeadWindow::default()
            .set_sketch_preview(Some(&packet))
            .is_err()
    );
    let owner = FakeHeadWork {
        preview: Some(packet),
        ..Default::default()
    };
    let mut window = RefusingPreviewWindow::default();
    assert!(publish_sketch_preview(&mut window, &owner).unwrap());
    assert_eq!(window.cleared, 1);
    let message = window.notice.unwrap().message();
    assert!(message.contains("Preview could not be drawn"));
    assert!(message.contains("sample budget"));
}
#[derive(Default)]
struct RefusingPreviewWindow {
    cleared: usize,
    notice: Option<crate::DesktopCommandNotice>,
}
impl HeadWindow for RefusingPreviewWindow {
    fn set_sketch_preview(
        &mut self,
        preview: Option<&oblikovati_viewport::ViewportSketchPreview>,
    ) -> Result<bool, String> {
        if preview.is_some() {
            return Err("sample budget exceeded".into());
        }
        self.cleared += 1;
        Ok(true)
    }
    fn show_command_notice(&mut self, notice: crate::DesktopCommandNotice) -> Result<(), String> {
        self.notice = Some(notice);
        Ok(())
    }
    fn events(&mut self, _: usize) -> Result<Vec<ViewportEvent>, String> {
        Ok(vec![])
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        Ok(HeadPresentation::Completed)
    }
    fn wait(&mut self, _: Option<std::time::Duration>) -> Result<bool, String> {
        Ok(false)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}

#[test]
fn unsupported_window_declines_edit_framing_without_mutation() {
    assert!(
        crate::work::IdleHeadWork
            .decline_sketch_edit("test".into())
            .is_err()
    );
    let mut window = FakeHeadWindow::default();
    assert!(!window.set_sketch_edit(None).unwrap());
    let edit = crate::HeadSketchEdit {
        document: 1,
        sketch: oblikovati_model::document_sketch::DocumentSketchId(2),
        plane: oblikovati_model::sketch::SketchPlane::xy(),
    };
    assert!(
        window
            .set_sketch_edit(Some(edit))
            .unwrap_err()
            .contains("framing")
    );
}

#[path = "frames/sketch_display.rs"]
mod sketch_display;
