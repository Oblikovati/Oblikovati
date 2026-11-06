// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::{self, ThemeToken};
#[derive(Clone, Copy, Debug)]
pub(super) enum BlenderPath {
    Ui,
    Widget(&'static str),
    Topbar,
    Console,
    Gradient,
    View3d,
    ViewSpace,
    WidgetState,
}
const UI: &[&str] = &["Theme", "user_interface", "ThemeUserInterface"];
const VIEW: &[&str] = &["Theme", "view_3d", "ThemeView3D"];
const SPACE: &[&str] = &[
    "Theme",
    "view_3d",
    "ThemeView3D",
    "space",
    "ThemeSpaceGradient",
];
impl BlenderPath {
    pub(super) fn nodes(self) -> Vec<&'static str> {
        match self {
            Self::Ui => UI.to_vec(),
            Self::Widget(group) => [UI, &[group, "ThemeWidgetColors"]].concat(),
            Self::Topbar => vec![
                "Theme",
                "topbar",
                "ThemeTopBar",
                "space",
                "ThemeSpaceGeneric",
            ],
            Self::Console => vec!["Theme", "console", "ThemeConsole"],
            Self::Gradient => [SPACE, &["gradients", "ThemeGradientColors"]].concat(),
            Self::View3d => VIEW.to_vec(),
            Self::ViewSpace => SPACE.to_vec(),
            Self::WidgetState => [UI, &["wcol_state", "ThemeWidgetStateColors"]].concat(),
        }
    }
}
pub(super) enum Source {
    Direct(BlenderPath, &'static str, bool),
    Mix(ThemeToken, f32),
    Alpha(ThemeToken, f32),
    SketchGreen,
}
pub(super) struct Binding {
    pub token: ThemeToken,
    pub source: Source,
}
macro_rules! slot {
    ($token:ident, $source:expr) => {
        Binding {
            token: types::$token,
            source: $source,
        }
    };
}
use BlenderPath::*;
use Source::{Alpha, Direct, Mix, SketchGreen};
// Go aa8297d1 theme/blendermap.go; order is the API's AllThemeTokens display order.
pub(super) const THEME_BINDINGS: &[Binding] = &[
    slot!(TOKEN_CHROME_WINDOW_BG, Direct(Ui, "editor_border", true)),
    slot!(TOKEN_CHROME_PANEL_BG, Direct(Ui, "panel_back", false)),
    slot!(
        TOKEN_CHROME_POPUP_BG,
        Direct(Widget("wcol_menu_back"), "inner", false)
    ),
    slot!(TOKEN_CHROME_MENU_BAR_BG, Direct(Topbar, "header", false)),
    slot!(TOKEN_CHROME_HEADER_BG, Direct(Ui, "panel_header", false)),
    slot!(
        TOKEN_CHROME_TEXT,
        Direct(Widget("wcol_regular"), "text", false)
    ),
    slot!(
        TOKEN_CHROME_TEXT_DISABLED,
        Alpha(types::TOKEN_CHROME_TEXT, 0.5)
    ),
    slot!(
        TOKEN_CHROME_BORDER,
        Direct(Widget("wcol_regular"), "outline", false)
    ),
    slot!(
        TOKEN_CHROME_CONTROL_BG,
        Direct(Widget("wcol_text"), "inner", false)
    ),
    slot!(
        TOKEN_CHROME_CONTROL_HOVER,
        Mix(types::TOKEN_CHROME_CONTROL_BG, 0.10)
    ),
    slot!(
        TOKEN_CHROME_CONTROL_ACTIVE,
        Mix(types::TOKEN_CHROME_CONTROL_BG, 0.18)
    ),
    slot!(
        TOKEN_CHROME_BUTTON,
        Direct(Widget("wcol_tool"), "inner", false)
    ),
    slot!(
        TOKEN_CHROME_BUTTON_HOVER,
        Mix(types::TOKEN_CHROME_BUTTON, 0.10)
    ),
    slot!(
        TOKEN_CHROME_BUTTON_ACTIVE,
        Direct(Widget("wcol_tool"), "inner_sel", false)
    ),
    slot!(
        TOKEN_CHROME_ACCENT,
        Direct(Widget("wcol_regular"), "inner_sel", true)
    ),
    slot!(
        TOKEN_CHROME_SCROLLBAR,
        Direct(Widget("wcol_scroll"), "item", false)
    ),
    slot!(TOKEN_CHROME_DANGER, Direct(Console, "line_error", true)),
    slot!(TOKEN_VIEWPORT_BG, Direct(Gradient, "high_gradient", true)),
    slot!(TOKEN_GRID_MINOR, Direct(View3d, "grid", false)),
    slot!(TOKEN_GRID_MAJOR, Alpha(types::TOKEN_GRID_MINOR, 2.)),
    slot!(TOKEN_GRID_AXIS, Alpha(types::TOKEN_GRID_MINOR, 3.)),
    slot!(TOKEN_SKETCH_GEOMETRY, Direct(View3d, "wire_edit", false)),
    slot!(TOKEN_SKETCH_SELECTED, Direct(View3d, "edge_select", false)),
    slot!(
        TOKEN_SKETCH_CANDIDATE,
        Direct(View3d, "editmesh_active", false)
    ),
    slot!(TOKEN_SKETCH_PREVIEW, Direct(View3d, "transform", false)),
    slot!(TOKEN_DIMENSION_DRIVING, Direct(ViewSpace, "text", false)),
    slot!(
        TOKEN_DIMENSION_DRIVEN,
        Direct(WidgetState, "inner_driven", false)
    ),
    slot!(TOKEN_DIMENSION_SKETCH, SketchGreen),
    slot!(TOKEN_SNAP_GLYPH, Direct(View3d, "vertex_select", false)),
    slot!(
        TOKEN_VIEWPORT_ACTIVE_BORDER,
        Direct(Ui, "editor_outline_active", false)
    ),
    slot!(TOKEN_PLANE_FAINT, Direct(Ui, "gizmo_primary", false)),
    slot!(TOKEN_PLANE_HOVER, Direct(Ui, "gizmo_hi", false)),
    slot!(TOKEN_PLANE_SELECTED, Direct(View3d, "object_active", false)),
    slot!(TOKEN_PLANE_FILL, Direct(View3d, "face", false)),
    slot!(
        TOKEN_SELECTION_HIGHLIGHT,
        Direct(View3d, "object_selected", false)
    ),
    slot!(TOKEN_ICON_PRIMARY, Direct(Ui, "icon_collection", false)),
    slot!(TOKEN_ICON_SECONDARY, Direct(Ui, "icon_object", false)),
    slot!(TOKEN_ICON_TERTIARY, Direct(Ui, "icon_modifier", false)),
    slot!(TOKEN_ICON_BACKGROUND, Direct(Ui, "panel_sub_back", false)),
];
