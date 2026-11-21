// SPDX-License-Identifier: GPL-2.0-only
use super::*;
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
    let expected = ["events", "confirm_close", "present", "events", "finish"];
    assert_eq!(window.calls, expected);
    assert!(window.confirmation_drawn);
}
#[test]
fn refused_close_draws_confirmation_without_another_input() {
    let mut window = FakeHeadWindow {
        inputs: [
            vec![ViewportEvent::CloseRequested],
            vec![],
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
    let result = run_window_with_owner(&mut window, HeadOptions::default(), &mut owner).unwrap();
    assert!(window.confirmation_drawn);
    assert_eq!(result.completed, 2);
    assert_eq!(
        window.calls,
        [
            "events",
            "confirm_close",
            "present",
            "events",
            "present",
            "wait",
            "events",
            "finish"
        ]
    );
}
#[test]
fn refused_close_without_gui_reports_an_error() {
    let mut window = FakeHeadWindow {
        inputs: [vec![ViewportEvent::CloseRequested]].into(),
        no_gui: true,
        ..Default::default()
    };
    let mut owner = FakeHeadWork {
        close_answers: [false].into(),
        ..Default::default()
    };
    let error = run_window_with_owner(&mut window, HeadOptions::default(), &mut owner).unwrap_err();
    assert!(error.contains("expected an initialized GUI"));
    assert_eq!(window.calls.last(), Some(&"finish"));
}
#[test]
fn refused_close_preserves_keyboard_input_routing() {
    let input = ViewportEvent::Input(oblikovati_viewport::ViewportInput::Text('a'));
    let mut window = FakeHeadWindow {
        inputs: [
            vec![ViewportEvent::CloseRequested],
            vec![input],
            vec![ViewportEvent::CloseRequested],
        ]
        .into(),
        wakes: [Ok(false), Ok(false)].into(),
        ..Default::default()
    };
    let mut owner = FakeHeadWork {
        close_answers: [false, true].into(),
        ..Default::default()
    };
    run_window_with_owner(&mut window, HeadOptions::default(), &mut owner).unwrap();
    assert!(window.gui_events.contains(&input));
    assert!(!window.gui_events.contains(&ViewportEvent::CloseRequested));
    assert!(owner.resolved_events.contains(&input));
    assert!(
        !owner
            .resolved_events
            .contains(&ViewportEvent::CloseRequested)
    );
}
