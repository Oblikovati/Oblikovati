// SPDX-License-Identifier: GPL-2.0-only
use super::{GuiFrameEngine, GuiFrameRequest, SCALE_SETTLING_FRAMES};

#[allow(dead_code, unsafe_code)]
#[path = "../../../../test-utilities/rust/drawn_text.rs"]
mod drawn_text;
#[allow(unsafe_code)]
#[path = "../../../../test-utilities/rust/drawn_windows.rs"]
mod drawn_windows;
#[allow(dead_code, unsafe_code)]
#[path = "../../../../test-utilities/rust/quick_access_icons.rs"]
mod quick_access_icons;

/// Locate an icon by its rendered texture. Example: `quick_access_point(engine, input, "Save")`.
pub(super) fn quick_access_point(
    engine: &mut GuiFrameEngine,
    input: &mut GuiFrameRequest,
    label: &str,
) -> [f32; 2] {
    draw_current_controls(engine, input);
    let center = quick_access_icons::standard_icon_center(label)
        .unwrap_or_else(|| panic!("{label:?}: expected a rendered quick access icon"));
    engine_pointer_pixels(input, center)
}

/// Locate an inline caption. Example: `ribbon_button_point(engine, input, "FinishSketch")`.
pub(super) fn ribbon_button_point(
    engine: &mut GuiFrameEngine,
    input: &mut GuiFrameRequest,
    caption: &str,
) -> [f32; 2] {
    draw_current_controls(engine, input);
    let frame = drawn_text::rendered_frame();
    let run = frame
        .runs
        .iter()
        .find(|run| run.text.split_whitespace().collect::<String>() == caption)
        .unwrap_or_else(|| panic!("{caption:?}: expected the inline ribbon caption"));
    engine_pointer_pixels(input, run.center())
}

fn draw_current_controls(engine: &mut GuiFrameEngine, input: &mut GuiFrameRequest) {
    input.events.clear();
    for _ in 0..SCALE_SETTLING_FRAMES {
        assert!(engine.frame(input).unwrap().actions.is_empty());
    }
}

fn engine_pointer_pixels(input: &GuiFrameRequest, [x, y]: [f32; 2]) -> [f32; 2] {
    [
        x * input.framebuffer_scale[0],
        y * input.framebuffer_scale[1],
    ]
}
