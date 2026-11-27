// SPDX-License-Identifier: GPL-2.0-only
//! Shared dockable Feature Properties shell controls (`@inherits_common_feature_properties`).
pub(crate) mod presets;
pub(crate) mod selector;

use crate::icons::{GuiRibbonIcons, IconGlyph, IconSize};
use crate::widget_abi::{self, IconButtonPresentation};
use imgui::{StyleColor, Ui};
use oblikovati_model::feature::sketch_prism::PrismBoolean;
pub(crate) use presets::SessionPresets;
pub(crate) use selector::SelectorStyle;

/// Standard Boolean / Solid / Surface output choice shared by Part Create Properties panels.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum FeatureOutputMode {
    #[default]
    NewSolid,
    Join,
    Cut,
    Intersect,
    Surface,
}

impl FeatureOutputMode {
    /// Convert to a solid Boolean operation when applicable.
    pub fn boolean(self) -> Option<PrismBoolean> {
        match self {
            Self::Join => Some(PrismBoolean::Join),
            Self::Cut => Some(PrismBoolean::Cut),
            Self::Intersect => Some(PrismBoolean::Intersect),
            Self::NewSolid | Self::Surface => None,
        }
    }

    /// Return whether this output mode builds a surface shell rather than a solid.
    pub fn is_surface(self) -> bool {
        self == Self::Surface
    }
}

/// Actions returned from a Properties footer row.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) struct PropertiesFooterOutcome {
    pub ok: bool,
    pub cancel: bool,
    pub toggle_preview: bool,
}

const BOOLEAN_CHOICES: [(FeatureOutputMode, &str, IconGlyph); 4] = [
    (FeatureOutputMode::Join, "Join", IconGlyph::ExtrudeJoin),
    (FeatureOutputMode::Cut, "Cut", IconGlyph::ExtrudeCut),
    (
        FeatureOutputMode::Intersect,
        "Intersect",
        IconGlyph::ExtrudeIntersect,
    ),
    (
        FeatureOutputMode::NewSolid,
        "New Solid",
        IconGlyph::ExtrudeNewSolid,
    ),
];

/// Draw the shared Properties header row with right-aligned Advanced Settings menu.
/// Returns whether the popup is currently open.
pub(crate) fn draw_properties_header(
    ui: &Ui,
    icons: &GuiRibbonIcons,
    popup_id: &str,
    presets_shown: &mut bool,
    extra_items: impl FnOnce(&Ui),
) -> bool {
    ui.align_text_to_frame_padding();
    ui.text("Properties");
    let icon = icons.logical[IconSize::Small.index()];
    let label = "Advanced Settings";
    let width = icon[0] + ui.calc_text_size(label)[0] + 16.;
    ui.same_line_with_pos((ui.window_content_region_max()[0] - width).max(0.));
    let button = IconButtonPresentation {
        texture: icons.key(IconGlyph::Preferences, IconSize::Small),
        size: [width, icon[1] + 6.],
        icon,
        large: false,
        show_caption: true,
    };
    if widget_abi::icon_button(ui, label, button) {
        ui.open_popup(popup_id);
    }
    let Some(_popup) = ui.begin_popup(popup_id) else {
        return false;
    };
    if ui
        .menu_item_config("Hide Presets")
        .selected(!*presets_shown)
        .build()
    {
        *presets_shown = !*presets_shown;
    }
    extra_items(ui);
    true
}

/// Draw the shared Output section (Solid/Surface toggle, Boolean selector, and Body Name).
/// Returns true if `mode` changed.
pub(crate) fn draw_output_section(
    ui: &Ui,
    icons: &GuiRibbonIcons,
    style: &mut SelectorStyle,
    id_prefix: &str,
    mode: &mut FeatureOutputMode,
    boolean_available: bool,
    (solid_name, surface_name): (&str, &str),
) -> bool {
    if !ui.collapsing_header("Output", imgui::TreeNodeFlags::DEFAULT_OPEN) {
        return false;
    }
    let before = *mode;
    let mut is_surface = mode.is_surface();
    let surface_label = if is_surface {
        "Output: Surface"
    } else {
        "Output: Solid"
    };
    if widget_abi::button(ui, surface_label, [0.; 2]) {
        is_surface = !is_surface;
        *mode = if is_surface {
            FeatureOutputMode::Surface
        } else if boolean_available {
            FeatureOutputMode::Join
        } else {
            FeatureOutputMode::NewSolid
        };
    }
    if mode.is_surface() {
        crate::theme::property_label(ui, "Body Name");
        ui.text(surface_name);
        return *mode != before;
    }
    if boolean_available {
        let popup = format!("##{id_prefix}-boolean-style");
        let combo = format!("##{id_prefix}-boolean");
        style.draw_style_menu(ui, "Boolean", &popup);
        if let Some(chosen) = style.choose(ui, icons, &combo, *mode, &BOOLEAN_CHOICES, |_| true) {
            *mode = chosen;
        }
    } else if mode.boolean().is_some() {
        *mode = FeatureOutputMode::NewSolid;
    }
    if *mode == FeatureOutputMode::NewSolid {
        crate::theme::property_label(ui, "Body Name");
        ui.text(solid_name);
    }
    *mode != before
}

/// Draw a red validation error badge (`!`) next to an invalid field, with hover tooltip.
pub(crate) fn draw_error_badge(ui: &Ui, message: &str) {
    ui.same_line();
    let _color = widget_abi::push_style_color(ui, StyleColor::Text, [0.92, 0.22, 0.22, 1.0]);
    ui.text("!");
    if ui.is_item_hovered() {
        ui.tooltip_text(message);
    }
}

/// Draw the shared OK / Cancel / Preview eye footer row.
pub(crate) fn draw_properties_footer(
    ui: &Ui,
    ok_enabled: bool,
    preview_hidden: bool,
) -> PropertiesFooterOutcome {
    ui.separator();
    let mut outcome = PropertiesFooterOutcome::default();
    {
        let _disabled = ui.begin_disabled(!ok_enabled);
        outcome.ok = widget_abi::button(ui, "OK", [0.; 2]);
    }
    ui.same_line();
    outcome.cancel = widget_abi::button(ui, "Cancel", [0.; 2]);
    ui.same_line();
    let eye = if preview_hidden {
        "Show Preview"
    } else {
        "Hide Preview"
    };
    outcome.toggle_preview = widget_abi::button(ui, eye, [0.; 2]);
    outcome
}
