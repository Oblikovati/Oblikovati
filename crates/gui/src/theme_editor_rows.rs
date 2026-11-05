// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::*;
// Go aa8297d1 theme/token.go owns the editor labels and group vocabulary.
pub(super) struct ThemeColorRow {
    pub token: ThemeToken,
    pub label: &'static str,
    pub group: &'static str,
}
pub(super) const THEME_COLOR_ROWS: &[ThemeColorRow] = &[
    ThemeColorRow {
        token: TOKEN_CHROME_WINDOW_BG,
        label: "Window background",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_PANEL_BG,
        label: "Panel background",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_POPUP_BG,
        label: "Popup background",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_MENU_BAR_BG,
        label: "Menu bar",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_HEADER_BG,
        label: "Header / title",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_TEXT,
        label: "Text",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_TEXT_DISABLED,
        label: "Text (disabled)",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_BORDER,
        label: "Border",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_CONTROL_BG,
        label: "Control background",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_CONTROL_HOVER,
        label: "Control (hover)",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_CONTROL_ACTIVE,
        label: "Control (active)",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_BUTTON,
        label: "Button",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_BUTTON_HOVER,
        label: "Button (hover)",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_BUTTON_ACTIVE,
        label: "Button (active)",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_ACCENT,
        label: "Accent",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_SCROLLBAR,
        label: "Scrollbar",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_CHROME_DANGER,
        label: "Danger / required",
        group: "Chrome",
    },
    ThemeColorRow {
        token: TOKEN_VIEWPORT_BG,
        label: "Background",
        group: "Viewport 2D",
    },
    ThemeColorRow {
        token: TOKEN_GRID_MINOR,
        label: "Grid (minor)",
        group: "Viewport 2D",
    },
    ThemeColorRow {
        token: TOKEN_GRID_MAJOR,
        label: "Grid (major)",
        group: "Viewport 2D",
    },
    ThemeColorRow {
        token: TOKEN_GRID_AXIS,
        label: "Grid (axis)",
        group: "Viewport 2D",
    },
    ThemeColorRow {
        token: TOKEN_SKETCH_GEOMETRY,
        label: "Sketch geometry",
        group: "Viewport 2D",
    },
    ThemeColorRow {
        token: TOKEN_SKETCH_SELECTED,
        label: "Sketch (selected)",
        group: "Viewport 2D",
    },
    ThemeColorRow {
        token: TOKEN_SKETCH_CANDIDATE,
        label: "Sketch (candidate)",
        group: "Viewport 2D",
    },
    ThemeColorRow {
        token: TOKEN_SKETCH_PREVIEW,
        label: "Sketch preview",
        group: "Viewport 2D",
    },
    ThemeColorRow {
        token: TOKEN_DIMENSION_DRIVING,
        label: "Dimension (driving)",
        group: "Viewport 2D",
    },
    ThemeColorRow {
        token: TOKEN_DIMENSION_DRIVEN,
        label: "Dimension (driven)",
        group: "Viewport 2D",
    },
    ThemeColorRow {
        token: TOKEN_DIMENSION_SKETCH,
        label: "Dimension (sketch)",
        group: "Viewport 2D",
    },
    ThemeColorRow {
        token: TOKEN_SNAP_GLYPH,
        label: "Snap glyph",
        group: "Viewport 2D",
    },
    ThemeColorRow {
        token: TOKEN_VIEWPORT_ACTIVE_BORDER,
        label: "Active view border",
        group: "Viewport 2D",
    },
    ThemeColorRow {
        token: TOKEN_PLANE_FAINT,
        label: "Work plane",
        group: "Gizmos 3D",
    },
    ThemeColorRow {
        token: TOKEN_PLANE_HOVER,
        label: "Work plane (hover)",
        group: "Gizmos 3D",
    },
    ThemeColorRow {
        token: TOKEN_PLANE_SELECTED,
        label: "Work plane (selected)",
        group: "Gizmos 3D",
    },
    ThemeColorRow {
        token: TOKEN_PLANE_FILL,
        label: "Work plane fill (alpha = opacity)",
        group: "Gizmos 3D",
    },
    ThemeColorRow {
        token: TOKEN_SELECTION_HIGHLIGHT,
        label: "Selection highlight",
        group: "Gizmos 3D",
    },
    ThemeColorRow {
        token: TOKEN_ICON_PRIMARY,
        label: "Icon (primary linework)",
        group: "Icons",
    },
    ThemeColorRow {
        token: TOKEN_ICON_SECONDARY,
        label: "Icon (secondary accent)",
        group: "Icons",
    },
    ThemeColorRow {
        token: TOKEN_ICON_TERTIARY,
        label: "Icon (tertiary detail)",
        group: "Icons",
    },
    ThemeColorRow {
        token: TOKEN_ICON_BACKGROUND,
        label: "Icon background plate",
        group: "Icons",
    },
];
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn every_canonical_token_has_one_named_editor_row() {
        let actual: std::collections::BTreeSet<_> = THEME_COLOR_ROWS
            .iter()
            .map(|row| row.token.clone())
            .collect();
        let expected = oblikovati_model::theme::theme_tokens().cloned().collect();
        assert_eq!(actual, expected);
        assert_eq!(actual.len(), THEME_COLOR_ROWS.len());
        assert!(
            oblikovati_model::theme::theme_tokens()
                .eq(THEME_COLOR_ROWS.iter().map(|row| &row.token))
        );
        assert!(
            THEME_COLOR_ROWS
                .iter()
                .all(|row| !row.label.is_empty() && !row.group.is_empty())
        );
    }
}
