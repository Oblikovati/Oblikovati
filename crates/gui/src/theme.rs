// SPDX-License-Identifier: GPL-2.0-only
use imgui::StyleColor;
use oblikovati_api::types::*;
use oblikovati_model::theme::ThemePalette;

// Go aa8297d1 head/ui/theme_apply.go is the binding oracle. Dear ImGui 1.89 calls
// TabSelected/TabDimmed/TabDimmedSelected TabActive/TabUnfocused/TabUnfocusedActive.
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
            StyleColor::Header,
        ],
    ),
    // ImGui 1.89 uses Text for the input cursor; it has no independent cursor slot.
    (TOKEN_CHROME_TEXT, &[StyleColor::Text]),
    (TOKEN_CHROME_TEXT_DISABLED, &[StyleColor::TextDisabled]),
    (
        TOKEN_CHROME_BORDER,
        &[StyleColor::Border, StyleColor::Separator],
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
    (TOKEN_CHROME_BUTTON_HOVER, &[StyleColor::ButtonHovered]),
    (
        TOKEN_CHROME_BUTTON_ACTIVE,
        &[StyleColor::ButtonActive, StyleColor::TabUnfocusedActive],
    ),
    (
        TOKEN_CHROME_ACCENT,
        &[
            StyleColor::TitleBgActive,
            StyleColor::TabActive,
            StyleColor::TabHovered,
            StyleColor::CheckMark,
            StyleColor::SliderGrab,
            StyleColor::SliderGrabActive,
            StyleColor::HeaderHovered,
            StyleColor::HeaderActive,
            StyleColor::SeparatorActive,
            StyleColor::ScrollbarGrabActive,
            StyleColor::TextSelectedBg,
        ],
    ),
    (
        TOKEN_CHROME_SCROLLBAR,
        &[StyleColor::ScrollbarGrab, StyleColor::ScrollbarGrabHovered],
    ),
];

pub(crate) fn apply_chrome_theme(
    colors: &mut [[f32; 4]; StyleColor::COUNT],
    palette: &ThemePalette,
) {
    for (token, slots) in CHROME_STYLE_BINDINGS {
        let color = &palette[token];
        for slot in *slots {
            colors[*slot as usize] = [color.r, color.g, color.b, color.a];
        }
    }
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
        assert_eq!(slots.len(), 35);
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
                    assert_eq!(colors[*slot as usize], [rgba.r, rgba.g, rgba.b, rgba.a]);
                }
            }
            assert_eq!(colors[StyleColor::PlotLines as usize], [0.123; 4]);
        }
    }
}
