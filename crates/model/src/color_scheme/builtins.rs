// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::OVERRIDE_COLOR_SOURCE;
// Go aa8297d1 model/colorscheme/scheme.go is the palette oracle.
type SchemeRecipe = (&'static str, BackgroundTypeEnum, [[u8; 3]; 6]);
const BUILTIN_SCHEME_RECIPES: [SchemeRecipe; 4] = [
    (
        "Default",
        GRADIENT_BACKGROUND,
        [
            [45, 48, 54],
            [60, 64, 72],
            [30, 32, 36],
            [255, 196, 0],
            [60, 160, 255],
            [0, 200, 160],
        ],
    ),
    (
        "Presentation",
        GRADIENT_BACKGROUND,
        [
            [235, 238, 242],
            [245, 247, 250],
            [205, 212, 222],
            [255, 140, 0],
            [0, 120, 215],
            [0, 160, 120],
        ],
    ),
    (
        "High Contrast",
        ONE_COLOR_BACKGROUND,
        [
            [0, 0, 0],
            [0, 0, 0],
            [0, 0, 0],
            [255, 255, 0],
            [0, 255, 255],
            [255, 0, 255],
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
