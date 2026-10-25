// SPDX-License-Identifier: GPL-2.0-only
//! Go Dark theme values for the shared Dear ImGui shell.

use imgui::{Context, Style, StyleColor};

/// Apply the Go application's default Dark chrome before any frame is drawn.
pub(crate) fn configure(context: &mut Context) {
    let style = context.style_mut();
    style.use_dark_colors();
    configure_geometry(style);
    apply_surface_palette(style);
    apply_control_palette(style);
    apply_navigation_palette(style);
}

fn configure_geometry(style: &mut Style) {
    style.window_padding = [6., 4.];
    style.frame_padding = [5., 3.];
    style.item_spacing = [6., 3.];
    style.item_inner_spacing = [4., 3.];
    style.window_border_size = 1.;
    style.frame_border_size = 1.;
    style.window_rounding = 0.;
    style.child_rounding = 0.;
    style.frame_rounding = 2.;
    style.grab_rounding = 2.;
    style.tab_rounding = 2.;
    style.scrollbar_rounding = 2.;
    style.scrollbar_size = 12.;
    style.indent_spacing = 12.;
}

fn apply_surface_palette(style: &mut Style) {
    style[StyleColor::Text] = rgba(0.773, 0.773, 0.773, 1.);
    style[StyleColor::TextDisabled] = rgba(0.49, 0.49, 0.49, 1.);
    style[StyleColor::WindowBg] = rgba(0.106, 0.106, 0.106, 1.);
    style[StyleColor::ChildBg] = rgba(0.106, 0.106, 0.106, 1.);
    style[StyleColor::PopupBg] = rgba(0.106, 0.106, 0.106, 1.);
    style[StyleColor::MenuBarBg] = rgba(0.086, 0.086, 0.086, 1.);
    style[StyleColor::Border] = rgba(0.114, 0.114, 0.114, 1.);
    style[StyleColor::Separator] = rgba(0.114, 0.114, 0.114, 1.);
    style[StyleColor::TitleBg] = rgba(0.129, 0.129, 0.129, 1.);
    style[StyleColor::TitleBgCollapsed] = rgba(0.129, 0.129, 0.129, 1.);
}

fn apply_control_palette(style: &mut Style) {
    style[StyleColor::FrameBg] = rgba(0.141, 0.141, 0.141, 1.);
    style[StyleColor::FrameBgHovered] = rgba(0.18, 0.18, 0.18, 1.);
    style[StyleColor::FrameBgActive] = rgba(0.506, 0.376, 0.89, 1.);
    style[StyleColor::Button] = rgba(0.18, 0.18, 0.18, 1.);
    style[StyleColor::ButtonHovered] = rgba(0.24, 0.24, 0.24, 1.);
    style[StyleColor::ButtonActive] = rgba(0.506, 0.376, 0.89, 1.);
    style[StyleColor::CheckMark] = rgba(0.506, 0.376, 0.89, 1.);
    style[StyleColor::SliderGrab] = rgba(0.506, 0.376, 0.89, 1.);
    style[StyleColor::SliderGrabActive] = rgba(0.592, 0.376, 0.902, 1.);
    style[StyleColor::TextSelectedBg] = rgba(0.506, 0.376, 0.89, 0.5);
}

fn apply_navigation_palette(style: &mut Style) {
    let accent = rgba(0.506, 0.376, 0.89, 1.);
    style[StyleColor::TitleBgActive] = accent;
    style[StyleColor::TabActive] = accent;
    style[StyleColor::TabHovered] = accent;
    style[StyleColor::HeaderHovered] = accent;
    style[StyleColor::HeaderActive] = accent;
    style[StyleColor::SeparatorHovered] = accent;
    style[StyleColor::SeparatorActive] = accent;
    style[StyleColor::ScrollbarGrab] = rgba(0.247, 0.247, 0.247, 1.);
    style[StyleColor::ScrollbarGrabHovered] = accent;
    style[StyleColor::ScrollbarGrabActive] = accent;
}

fn rgba(red: f32, green: f32, blue: f32, alpha: f32) -> [f32; 4] {
    [red, green, blue, alpha]
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn go_dark_palette_sets_the_chrome_and_accent_values() {
        let mut context = Context::create();
        configure(&mut context);
        let style = context.style();
        assert_eq!(style[StyleColor::WindowBg], rgba(0.106, 0.106, 0.106, 1.));
        assert_eq!(style[StyleColor::TabActive], rgba(0.506, 0.376, 0.89, 1.));
    }
}
