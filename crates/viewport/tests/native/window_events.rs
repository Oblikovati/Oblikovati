// SPDX-License-Identifier: GPL-2.0-only
//! The desktop owner can wait for a secondary target without losing its input.
use oblikovati_viewport::{
    Viewport, ViewportEvent, ViewportOptions, ViewportRuntime, ViewportWake,
};
use std::time::Duration;

#[test]
fn a_secondary_resize_wakes_the_owner_and_stays_on_its_window() {
    let runtime = ViewportRuntime::new(ViewportOptions::default()).unwrap();
    let mut main = runtime
        .open_window("Main event target", [160, 120])
        .unwrap();
    let mut drawing = runtime
        .open_window("Drawing event target", [180, 140])
        .unwrap();
    let wake = ViewportWake::new().unwrap();
    drain(&mut main);
    drain(&mut drawing);

    let extent = [210, 154];
    drawing.resize(extent).unwrap();
    assert!(
        !main
            .wait_events(&wake, Some(Duration::ZERO))
            .unwrap()
            .native_events
    );
    wake.wake().unwrap();
    let ready = main
        .wait_events_for_windows(&[&drawing], &wake, Some(Duration::from_secs(1)))
        .unwrap();
    assert!(ready.native_events && ready.worker_wake, "{ready:?}");
    assert!(
        drawing
            .poll_events(32)
            .unwrap()
            .contains(&ViewportEvent::Resized(extent))
    );

    drawing.close();
    assert!(
        main.wait_events_for_windows(&[&drawing], &wake, Some(Duration::ZERO))
            .unwrap_err()
            .contains("expected a live view")
    );
}

#[test]
fn event_group_requires_live_native_targets_from_the_same_runtime() {
    let runtime = ViewportRuntime::new(ViewportOptions::default()).unwrap();
    let main = runtime
        .open_window("Main event target", [160, 120])
        .unwrap();
    let wake = ViewportWake::new().unwrap();
    let offscreen = runtime.open_offscreen([32, 32]).unwrap();
    assert!(
        main.wait_events_for_windows(&[&offscreen], &wake, Some(Duration::ZERO))
            .unwrap_err()
            .contains("expected a native event source")
    );
    let other_runtime = ViewportRuntime::new(ViewportOptions::default()).unwrap();
    let foreign = other_runtime
        .open_window("Foreign event target", [180, 140])
        .unwrap();
    assert!(
        main.wait_events_for_windows(&[&foreign], &wake, Some(Duration::ZERO))
            .unwrap_err()
            .contains("expected the runtime")
    );
}

fn drain(window: &mut Viewport) {
    for _ in 0..4 {
        if window.poll_events(32).unwrap().is_empty() {
            return;
        }
    }
    panic!("expected a quiet test window");
}
