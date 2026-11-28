// SPDX-License-Identifier: GPL-2.0-only
use imgui::StyleColor;
use oblikovati_api::types::*;
use oblikovati_model::theme::ThemePalette;

// Keep chrome surfaces neutral and use the accent for focus and input feedback.
const CHROME_STYLE_BINDINGS: &[(ThemeToken, &[StyleColor])] = &[
    (TOKEN_CHROME_WINDOW_BG, &[StyleColor::WindowBg]),
    (TOKEN_CHROME_PANEL_BG, &[StyleColor::ChildBg]),
    (TOKEN_CHROME_POPUP_BG, &[StyleColor::PopupBg]),
    (TOKEN_CHROME_MENU_BAR_BG, &[StyleColor::MenuBarBg]),
    (
        TOKEN_CHROME_HEADER_BG,
        &[
            StyleColor::TitleBg,
            StyleColor::TitleBgCollapsed,
            StyleColor::TitleBgActive,
            StyleColor::Header,
            StyleColor::TabActive,
            StyleColor::TabUnfocusedActive,
            StyleColor::TableHeaderBg,
            StyleColor::TableRowBgAlt,
        ],
    ),
    // ImGui 1.89 uses Text for the input cursor; it has no independent cursor slot.
    (TOKEN_CHROME_TEXT, &[StyleColor::Text]),
    (TOKEN_CHROME_TEXT_DISABLED, &[StyleColor::TextDisabled]),
    (
        TOKEN_CHROME_BORDER,
        &[
            StyleColor::Border,
            StyleColor::Separator,
            StyleColor::TableBorderLight,
            StyleColor::TableBorderStrong,
        ],
    ),
    // Keep the track recessed: sharing the grab colour made scrollbars invisible (#1471).
    (
        TOKEN_CHROME_CONTROL_BG,
        &[StyleColor::FrameBg, StyleColor::ScrollbarBg],
    ),
    (
        TOKEN_CHROME_CONTROL_HOVER,
        &[StyleColor::FrameBgHovered, StyleColor::SeparatorHovered],
    ),
    (TOKEN_CHROME_CONTROL_ACTIVE, &[StyleColor::FrameBgActive]),
    (
        TOKEN_CHROME_BUTTON,
        &[
            StyleColor::Button,
            StyleColor::Tab,
            StyleColor::TabUnfocused,
        ],
    ),
    (
        TOKEN_CHROME_BUTTON_HOVER,
        &[
            StyleColor::ButtonHovered,
            StyleColor::TabHovered,
            StyleColor::HeaderHovered,
        ],
    ),
    (
        TOKEN_CHROME_BUTTON_ACTIVE,
        &[StyleColor::ButtonActive, StyleColor::HeaderActive],
    ),
    (
        TOKEN_CHROME_ACCENT,
        &[
            StyleColor::CheckMark,
            StyleColor::SliderGrab,
            StyleColor::SliderGrabActive,
            StyleColor::SeparatorActive,
            StyleColor::ScrollbarGrabActive,
            StyleColor::TextSelectedBg,
            StyleColor::NavHighlight,
            StyleColor::DockingPreview,
            StyleColor::ResizeGripHovered,
            StyleColor::ResizeGripActive,
        ],
    ),
    (
        TOKEN_CHROME_SCROLLBAR,
        &[
            StyleColor::ScrollbarGrab,
            StyleColor::ScrollbarGrabHovered,
            StyleColor::ResizeGrip,
        ],
    ),
];

pub(crate) fn configure_chrome(style: &mut imgui::Style) -> Result<(), String> {
    let theme = oblikovati_model::theme::ThemeDefinition::dark()?;
    apply_chrome_theme(&mut style.colors, theme.palette());
    style.window_padding = [10., 10.];
    style.frame_padding = [8., 5.];
    style.item_spacing = [8., 6.];
    style.item_inner_spacing = [6., 4.];
    style.cell_padding = [8., 5.];
    style.indent_spacing = 16.;
    style.scrollbar_size = 10.;
    style.scrollbar_rounding = 9.;
    style.grab_rounding = 4.;
    style.window_rounding = 6.;
    style.child_rounding = 4.;
    style.frame_rounding = 4.;
    style.popup_rounding = 6.;
    style.tab_rounding = 4.;
    style.window_border_size = 1.;
    style.frame_border_size = 1.;
    style.popup_border_size = 1.;
    Ok(())
}

pub(crate) fn is_high_contrast(ui: &imgui::Ui) -> bool {
    let surface = ui.style_color(StyleColor::WindowBg);
    surface[..3].iter().all(|value| *value < 0.01)
}

pub(crate) fn is_light_theme(ui: &imgui::Ui) -> bool {
    ui.style_color(StyleColor::WindowBg)[0] > 0.5
}

pub(crate) fn surface_subtle(ui: &imgui::Ui) -> [f32; 4] {
    hex_rgb(if is_high_contrast(ui) {
        0x14141A
    } else if is_light_theme(ui) {
        0xF0F2F6
    } else {
        0x1F222A
    })
}

pub(crate) fn border_subtle(ui: &imgui::Ui) -> [f32; 4] {
    hex_rgb(if is_high_contrast(ui) {
        0x525266
    } else if is_light_theme(ui) {
        0xE4E7EE
    } else {
        0x22252E
    })
}

pub(crate) fn text_secondary(ui: &imgui::Ui) -> [f32; 4] {
    hex_rgb(if is_high_contrast(ui) {
        0xE2E4EC
    } else if is_light_theme(ui) {
        0x4B5263
    } else {
        0x9CA3B2
    })
}

pub(crate) fn text_muted(ui: &imgui::Ui) -> [f32; 4] {
    hex_rgb(if is_high_contrast(ui) {
        0xC4C7D4
    } else if is_light_theme(ui) {
        0x636B7E
    } else {
        0x7C8496
    })
}

pub(crate) fn brand_accent(ui: &imgui::Ui) -> [f32; 4] {
    hex_rgb(if is_high_contrast(ui) {
        0xA599FF
    } else if is_light_theme(ui) {
        0x5B46E8
    } else {
        0x8172FA
    })
}

pub(crate) fn brand_subtle(ui: &imgui::Ui) -> [f32; 4] {
    let mut color = brand_accent(ui);
    color[3] = if is_high_contrast(ui) {
        0.24
    } else if is_light_theme(ui) {
        0.10
    } else {
        0.16
    };
    color
}

pub(crate) fn selection_accent(ui: &imgui::Ui) -> [f32; 4] {
    hex_rgb(if is_high_contrast(ui) {
        0x2DD4BF
    } else if is_light_theme(ui) {
        0x0D9488
    } else {
        0x14B8A6
    })
}

pub(crate) fn selection_subtle(ui: &imgui::Ui) -> [f32; 4] {
    let mut color = selection_accent(ui);
    color[3] = if is_high_contrast(ui) {
        0.28
    } else if is_light_theme(ui) {
        0.14
    } else {
        0.20
    };
    color
}

pub(crate) fn success_accent(ui: &imgui::Ui) -> [f32; 4] {
    hex_rgb(if is_high_contrast(ui) {
        0x4ADE80
    } else if is_light_theme(ui) {
        0x16A34A
    } else {
        0x22C55E
    })
}

pub(crate) fn success_fill(ui: &imgui::Ui) -> [f32; 4] {
    hex_rgb(if is_high_contrast(ui) {
        0x4ADE80
    } else {
        0x15803D
    })
}

pub(crate) fn error_accent(ui: &imgui::Ui) -> [f32; 4] {
    hex_rgb(if is_high_contrast(ui) {
        0xFB7185
    } else if is_light_theme(ui) {
        0xE11D48
    } else {
        0xF43F5E
    })
}

pub(crate) fn secondary_button(ui: &imgui::Ui, label: &str, size: [f32; 2]) -> bool {
    let _fill = ui.push_style_color(StyleColor::Button, surface_subtle(ui));
    let _border = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
    crate::widget_abi::button(ui, label, size)
}

pub(crate) fn property_cancel(ui: &imgui::Ui) -> bool {
    secondary_button(ui, "Cancel", [68., 28.])
}

pub(crate) fn property_apply_new(ui: &imgui::Ui, enabled: bool) -> bool {
    let _disabled = ui.begin_disabled(!enabled);
    let accent = success_accent(ui);
    let hover = success_fill(ui);
    let _fill = ui.push_style_color(StyleColor::Button, surface_subtle(ui));
    let _hover = ui.push_style_color(StyleColor::ButtonHovered, hover);
    let _active = ui.push_style_color(StyleColor::ButtonActive, hover);
    let _border = ui.push_style_color(StyleColor::Border, accent);
    let _text = ui.push_style_color(StyleColor::Text, accent);
    let _bw = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
    let clicked = crate::widget_abi::button(ui, "+##apply-and-create-new", [28., 28.]);
    if ui.is_item_hovered() {
        let [lo, hi] = [ui.item_rect_min(), ui.item_rect_max()];
        let label = "+";
        let [tw, th] = ui.calc_text_size(label);
        let on_accent = if is_high_contrast(ui) {
            [0., 0., 0., 1.]
        } else {
            [1., 1., 1., 1.]
        };
        ui.get_window_draw_list().add_text(
            [
                ((lo[0] + hi[0] - tw) * 0.5).floor(),
                ((lo[1] + hi[1] - th) * 0.5).floor(),
            ],
            on_accent,
            label,
        );
        ui.tooltip_text("Apply and Create New");
    }
    clicked
}

pub(crate) fn active_tab_underline(ui: &imgui::Ui) {
    tab_underline_at(ui, ui.item_rect_min(), ui.item_rect_max());
}

pub(crate) fn tab_underline_at(ui: &imgui::Ui, [left, _]: [f32; 2], [right, bottom]: [f32; 2]) {
    let bottom = bottom.min(crate::widget_abi::window_draw_clip_bottom(ui));
    ui.get_window_draw_list()
        .add_rect(
            [left, bottom - 2.],
            [right, bottom],
            ui.style_color(StyleColor::NavHighlight),
        )
        .filled(true)
        .build();
}

pub(crate) fn property_window_border(ui: &imgui::Ui) -> imgui::ColorStackToken<'_> {
    ui.push_style_color(StyleColor::Border, strong_border(ui))
}

/// Resolve the Studio strong outline for the active surface, such as a dark asset swatch.
pub(crate) fn strong_border(ui: &imgui::Ui) -> [f32; 4] {
    let surface = ui.style_color(StyleColor::WindowBg);
    let ink = if surface[..3].iter().all(|value| *value < 0.01) {
        0xFFFFFF
    } else if surface[0] > 0.5 {
        0xAEB6C6
    } else {
        0x3F4556
    };
    hex_rgb(ink)
}

pub(crate) fn floating_property_shadow(ui: &imgui::Ui) {
    if crate::widget_abi::window_is_docked(ui) {
        return;
    }
    let top = ui.window_pos();
    let size = ui.window_size();
    let bottom: [f32; 2] = std::array::from_fn(|axis| top[axis] + size[axis]);
    let draw = ui.get_background_draw_list();
    for radius in [10., 7., 4.] {
        draw.add_rect(
            [top[0] - radius, top[1] - radius + 3.],
            [bottom[0] + radius, bottom[1] + radius + 3.],
            [0., 0., 0., 0.09],
        )
        .rounding(radius + 3.)
        .filled(true)
        .build();
    }
}

pub(crate) fn property_label(ui: &imgui::Ui, label: &str) {
    let value_start = ui.cursor_pos()[0] + ui.content_region_avail()[0] * 0.4;
    ui.align_text_to_frame_padding();
    ui.text(label);
    ui.same_line_with_pos(value_start);
    ui.set_next_item_width(-1.);
}

pub(crate) fn brand_action_colors(ui: &imgui::Ui) -> [[f32; 4]; 4] {
    let background = ui.style_color(StyleColor::WindowBg);
    let contrast = background[..3].iter().all(|value| *value < 0.01);
    let (fill, hover, active) = if contrast {
        (0xa599ff, 0xbfb5ff, 0x8f80ff)
    } else if background[0] > 0.5 {
        (0x5b46e8, 0x4d38d6, 0x402cc0)
    } else {
        (0x6754e8, 0x7866f2, 0x5845d6)
    };
    [
        hex_rgb(fill),
        hex_rgb(hover),
        hex_rgb(active),
        if contrast { [0., 0., 0., 1.] } else { [1.; 4] },
    ]
}

pub(crate) fn primary_button(ui: &imgui::Ui, label: &str, size: [f32; 2]) -> bool {
    let [fill, hover, active, text] = brand_action_colors(ui);
    let _fill = ui.push_style_color(StyleColor::Button, fill);
    let _hover = ui.push_style_color(StyleColor::ButtonHovered, hover);
    let _active = ui.push_style_color(StyleColor::ButtonActive, active);
    let _text = ui.push_style_color(StyleColor::Text, text);
    crate::widget_abi::button(ui, label, size)
}

pub(crate) fn property_accept(ui: &imgui::Ui, enabled: bool) -> bool {
    let alpha = ui.clone_style().alpha;
    let _disabled = ui.begin_disabled(!enabled);
    if enabled || ui.style_color(StyleColor::WindowBg)[0] <= 0.5 {
        return primary_button(ui, "OK", [72., 28.]);
    }
    let _alpha = ui.push_style_var(imgui::StyleVar::Alpha(alpha));
    let _fill = ui.push_style_color(StyleColor::Button, hex_rgb(0xE2E5EC));
    let _ink = ui.push_style_color(StyleColor::Text, hex_rgb(0x6B7280));
    crate::widget_abi::button(ui, "OK", [72., 28.])
}

pub(crate) fn hex_rgb(hex: u32) -> [f32; 4] {
    let channel = |shift| ((hex >> shift) & 0xff_u32) as f32 / 255.;
    [channel(16), channel(8), channel(0), 1.]
}

pub(crate) fn apply_chrome_theme(
    colors: &mut [[f32; 4]; StyleColor::COUNT],
    palette: &ThemePalette,
) -> bool {
    let previous = *colors;
    for (token, slots) in CHROME_STYLE_BINDINGS {
        let color = &palette[token];
        for slot in *slots {
            colors[*slot as usize] = [color.r, color.g, color.b, color.a];
        }
    }
    // A theme's translucent menu colour let the rows behind a menu read through its own rows.
    colors[StyleColor::PopupBg as usize][3] = 1.;
    colors[StyleColor::TextSelectedBg as usize][3] *= 0.28;
    colors[StyleColor::DockingPreview as usize][3] *= 0.28;
    *colors != previous
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::theme::ThemeDefinition;
    use std::collections::BTreeSet;

    #[test]
    fn every_bound_style_slot_has_one_semantic_owner() {
        let mut slots = BTreeSet::new();
        for (_, bindings) in CHROME_STYLE_BINDINGS {
            for slot in *bindings {
                assert!(slots.insert(*slot as usize), "duplicate slot {slot:?}");
            }
        }
        assert_eq!(slots.len(), 44);
    }

    #[test]
    fn restyling_copies_exact_theme_channels_and_preserves_unbound_slots() {
        let mut colors = [[0.123; 4]; StyleColor::COUNT];
        for theme in [
            ThemeDefinition::dark().unwrap(),
            ThemeDefinition::light().unwrap(),
        ] {
            apply_chrome_theme(&mut colors, theme.palette());
            for (token, slots) in CHROME_STYLE_BINDINGS {
                let rgba = theme.color(token);
                for slot in *slots {
                    let alpha = match slot {
                        StyleColor::PopupBg => 1.,
                        StyleColor::TextSelectedBg | StyleColor::DockingPreview => rgba.a * 0.28,
                        _ => rgba.a,
                    };
                    assert_eq!(colors[*slot as usize], [rgba.r, rgba.g, rgba.b, alpha]);
                }
            }
            assert_eq!(colors[StyleColor::PlotLines as usize], [0.123; 4]);
        }
    }
}
