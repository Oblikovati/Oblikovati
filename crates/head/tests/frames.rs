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
    fn wait(&mut self) -> Result<bool, String> {
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
        control.events(&[event]);
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
