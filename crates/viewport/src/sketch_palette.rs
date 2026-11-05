// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::theme::{Rgba, ThemeDefinition};
/// Application sketch colours stay independent of geometry and document history.
#[derive(Clone, Debug, PartialEq)]
pub struct ViewportSketchPalette {
    pub(crate) geometry: Rgba,
    pub(crate) preview: Rgba,
}
impl ViewportSketchPalette {
    /// Copy the same semantic tokens as Go's sketchThemeColors; example: `ViewportSketchPalette::from_theme(library.active())`.
    pub fn from_theme(theme: &ThemeDefinition) -> Self {
        Self {
            geometry: theme.color(&oblikovati_api::types::TOKEN_SKETCH_GEOMETRY),
            preview: theme.color(&oblikovati_api::types::TOKEN_SKETCH_PREVIEW),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn palettes_copy_exact_go_tokens_and_ignore_unrelated_chrome_edits() {
        for theme in [
            ThemeDefinition::dark().unwrap(),
            ThemeDefinition::light().unwrap(),
        ] {
            let palette = ViewportSketchPalette::from_theme(&theme);
            assert_eq!(
                palette.geometry,
                theme.color(&oblikovati_api::types::TOKEN_SKETCH_GEOMETRY)
            );
            assert_eq!(
                palette.preview,
                theme.color(&oblikovati_api::types::TOKEN_SKETCH_PREVIEW)
            );
            let mut custom = theme.duplicate("Workshop").unwrap();
            custom
                .set_color(
                    &oblikovati_api::types::TOKEN_CHROME_ACCENT,
                    Rgba {
                        r: 1.,
                        g: 0.,
                        b: 0.,
                        a: 0.5,
                    },
                )
                .unwrap();
            assert_eq!(ViewportSketchPalette::from_theme(&custom), palette);
        }
    }
}
