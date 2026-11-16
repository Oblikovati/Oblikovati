// SPDX-License-Identifier: GPL-2.0-only
//! Diagnostic frame timing for Ready-state hover (#4043).
use crate::HeadWork;
use oblikovati_viewport::{ViewportEvent, ViewportInput};
use std::time::{Duration, Instant};

#[test]
#[ignore = "measure native Ready hover frames in an optimized build"]
fn measure_ready_hover_frames_for_dense_sketch() {
    let mut world = super::dense_sketch_world();
    let point = world.sketch_pixel_of([0., 0.]);
    let app = world.app();
    let extent = app.window.native.drawable_extent().unwrap();
    assert_eq!(extent, [1440, 900]);
    eprintln!(
        "ready_hover device={} extent={extent:?}",
        app.window.native.runtime.device_name()
    );
    let samples = collect_frames(app, point);
    assert!(
        app.owner
            .gui_command_state()
            .unwrap()
            .sketch_pick_cue
            .is_some()
    );
    report_frames(&samples);
    world.close();
}

fn collect_frames(
    app: &mut super::super::super::world::NativeApplication,
    point: [i32; 2],
) -> Vec<Duration> {
    let mut samples = Vec::with_capacity(120);
    for index in 0..140 {
        app.window.scripted = vec![ViewportEvent::Input(ViewportInput::Motion {
            position: [point[0] + index % 20, point[1] + 3],
            modifiers: Default::default(),
        })];
        let start = Instant::now();
        let summary = crate::frames::run_frames(
            &mut app.window,
            crate::HeadOptions { frames: Some(1) },
            &mut app.owner,
        )
        .unwrap();
        assert_eq!(summary.completed, 1);
        if index >= 20 {
            samples.push(start.elapsed());
        }
    }
    samples
}

fn report_frames(samples: &[Duration]) {
    for (index, duration) in samples.iter().enumerate() {
        eprintln!(
            "ready_hover_sample index={index} frame_ns={}",
            duration.as_nanos()
        );
    }
    let mut ordered = samples.to_vec();
    ordered.sort_unstable();
    let over_budget = samples
        .iter()
        .filter(|frame| frame.as_secs_f64() > 1. / 60.)
        .count();
    eprintln!(
        "ready_hover n={} p50={:?} p95={:?} p99={:?} max={:?} over_16_67ms={over_budget}",
        ordered.len(),
        ordered[60],
        ordered[113],
        ordered[118],
        ordered[119]
    );
}
