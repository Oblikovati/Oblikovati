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

#[test]
#[ignore = "measure batched native Ready hover frames in an optimized build"]
fn measure_ready_hover_motion_bursts() {
    let mut world = super::dense_sketch_world();
    world.pump(Vec::new());
    let point = world.sketch_pixel_of([0., 0.]);
    world.pump(vec![ViewportEvent::Input(ViewportInput::Motion {
        position: [point[0], point[1] + 3],
        modifiers: Default::default(),
    })]);
    let app = world.app();
    let extent = app.window.native.drawable_extent().unwrap();
    assert_eq!(extent, [1440, 900]);
    eprintln!(
        "ready_hover_burst device={} extent={extent:?}",
        app.window.native.runtime.device_name()
    );
    let mut durations = Vec::new();
    let mut delivered = Vec::new();
    for frame in 0..40 {
        app.window.passed.clear();
        app.window.scripted = (0..16)
            .map(|index| {
                ViewportEvent::Input(ViewportInput::Motion {
                    position: [point[0] + (index + frame) % 12, point[1] + 3],
                    modifiers: Default::default(),
                })
            })
            .collect();
        let start = Instant::now();
        let summary = crate::frames::run_frames(
            &mut app.window,
            crate::HeadOptions { frames: Some(1) },
            &mut app.owner,
        )
        .unwrap();
        assert_eq!(summary.completed, 1);
        if frame >= 10 {
            durations.push(start.elapsed());
            delivered.push(app.window.passed.len());
        }
    }
    let state = app.owner.gui_command_state().unwrap();
    assert!(
        state.sketch_at_ready,
        "burst must leave the sketch at Ready"
    );
    assert!(
        state.sketch_pick_cue.is_some(),
        "burst must end over sketch geometry"
    );
    durations.sort();
    eprintln!(
        "ready_hover_burst events=16 n={} p50={:?} p95={:?} max={:?} delivered={delivered:?}",
        durations.len(),
        durations[15],
        durations[28],
        durations[29]
    );
    world.close();
}
