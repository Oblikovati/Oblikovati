// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn each_activity_reason_keeps_the_next_frame_due() {
    assert!(!GuiFrameActivity::default().needs_frame());
    for activity in [
        GuiFrameActivity {
            queued_input: true,
            ..Default::default()
        },
        GuiFrameActivity {
            pending_layout: true,
            ..Default::default()
        },
        GuiFrameActivity {
            interaction: true,
            ..Default::default()
        },
    ] {
        assert!(activity.needs_frame());
    }
}
#[test]
fn real_window_layout_settles_without_a_fixed_number_of_warmup_frames() {
    let mut context = imgui::Context::create();
    context.set_ini_filename(None);
    context.io_mut().display_size = [400., 240.];
    context.io_mut().delta_time = 1. / 60.;
    context.fonts().build_rgba32_texture();
    let mut activity = layout_frame(&mut context);
    assert!(activity.pending_layout);
    let mut attempts = 0;
    while activity.needs_frame() {
        assert!(attempts < 8, "layout did not settle: {activity:?}");
        activity = layout_frame(&mut context);
        attempts += 1;
    }
    assert!(attempts > 0);
}
fn layout_frame(context: &mut imgui::Context) -> GuiFrameActivity {
    let ui = context.frame();
    ui.window("Measured layout")
        .build(|| ui.text("Content determines size"));
    context.render();
    crate::context_owner::frame_activity().unwrap()
}
