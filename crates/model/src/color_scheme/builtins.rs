// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::OVERRIDE_COLOR_SOURCE;
// The studio schemes use the handoff's background, selection, and work-feature colors.
// Sky retains the palette from Go aa8297d1 model/colorscheme/scheme.go.
type SchemeRecipe = (&'static str, BackgroundTypeEnum, [[u8; 3]; 6]);
const BUILTIN_SCHEME_RECIPES: [SchemeRecipe; 4] = [
    (
        "Default",
        GRADIENT_BACKGROUND,
        [
            [20, 22, 27],
            [20, 22, 27],
            [30, 33, 42],
            [245, 158, 11],
            [20, 184, 166],
            [45, 212, 191],
        ],
    ),
    (
        "Presentation",
        GRADIENT_BACKGROUND,
        [
            [244, 246, 249],
            [244, 246, 249],
            [228, 232, 240],
            [217, 119, 6],
            [13, 148, 136],
            [20, 184, 166],
        ],
    ),
    (
        "High Contrast",
        ONE_COLOR_BACKGROUND,
        [
            [0, 0, 0],
            [0, 0, 0],
            [0, 0, 0],
            [251, 191, 36],
            [45, 212, 191],
            [94, 234, 212],
        ],
    ),
    (
        "Sky",
        GRADIENT_BACKGROUND,
        [
            [120, 160, 210],
            [150, 190, 235],
            [225, 235, 245],
            [255, 170, 0],
            [20, 110, 200],
            [0, 150, 110],
        ],
    ),
];
pub(super) fn schemes() -> Vec<ColorScheme> {
    BUILTIN_SCHEME_RECIPES
        .into_iter()
        .map(|(name, mode, colors)| scheme(name, mode, colors))
        .collect()
}
fn scheme(name: &str, background_type: BackgroundTypeEnum, colors: [[u8; 3]; 6]) -> ColorScheme {
    let [
        screen,
        top,
        bottom,
        highlight,
        primary_select,
        secondary_select,
    ] = colors.map(opaque_color);
    ColorScheme {
        name: name.into(),
        background_type,
        screen,
        top,
        bottom,
        highlight,
        primary_select,
        secondary_select,
    }
}
fn opaque_color([r, g, b]: [u8; 3]) -> Color {
    Color {
        r,
        g,
        b,
        opacity: 1.,
        source: OVERRIDE_COLOR_SOURCE,
    }
}
