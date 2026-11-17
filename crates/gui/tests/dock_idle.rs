// SPDX-License-Identifier: GPL-2.0-only
//! Idle dock reconciliation must not schedule frames indefinitely.
use oblikovati_gui::{GuiCommandState, GuiFrameEngine, GuiFrameRequest};

#[test]
fn settled_docks_do_not_schedule_idle_frames() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let request = GuiFrameRequest {
        extent: [640, 360],
        events: Vec::new(),
        commands: GuiCommandState::default(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1., 1.],
    };
    for _ in 0..32 {
        if !engine.frame(&request).unwrap().activity.needs_frame() {
            break;
        }
    }
    let started = std::time::Instant::now();
    for _ in 0..20 {
        assert!(!engine.frame(&request).unwrap().activity.needs_frame());
    }
    eprintln!("20 idle GUI frames: {:?}", started.elapsed());
}
