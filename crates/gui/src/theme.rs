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
    style.frame_rounding = 4.;
    style.popup_rounding = 4.;
    style.tab_rounding = 4.;
    style.popup_border_size = 1.;
    Ok(())
}

pub(crate) fn active_tab_underline(ui: &imgui::Ui) {
    let [left, _] = ui.item_rect_min();
    let [right, bottom] = ui.item_rect_max();
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

pub(crate) fn property_label(ui: &imgui::Ui, label: &str) {
    let value_start = ui.cursor_pos()[0] + ui.content_region_avail()[0] * 0.4;
    ui.align_text_to_frame_padding();
    ui.text(label);
    ui.same_line_with_pos(value_start);
    ui.set_next_item_width(-1.);
}

pub(crate) fn primary_button(ui: &imgui::Ui, label: &str, size: [f32; 2]) -> bool {
    let background = ui.style_color(StyleColor::WindowBg);
    let contrast = background[..3].iter().all(|value| *value < 0.01);
    let (fill, hover, active) = if contrast {
        (0xa599ff, 0xbfb5ff, 0x8f80ff)
    } else if background[0] > 0.5 {
        (0x5b46e8, 0x4d38d6, 0x402cc0)
    } else {
        (0x6754e8, 0x7866f2, 0x5845d6)
    };
    let _fill = ui.push_style_color(StyleColor::Button, hex_rgb(fill));
    let _hover = ui.push_style_color(StyleColor::ButtonHovered, hex_rgb(hover));
    let _active = ui.push_style_color(StyleColor::ButtonActive, hex_rgb(active));
    let _text = ui.push_style_color(
        StyleColor::Text,
        if contrast { [0., 0., 0., 1.] } else { [1.; 4] },
    );
    crate::widget_abi::button(ui, label, size)
}
fn hex_rgb(hex: u32) -> [f32; 4] {
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
