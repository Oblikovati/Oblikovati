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
}

/// Actions returned from the compact single-row Properties identity header.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) struct PropertiesHeaderOutcome {
    pub toggle_preview: bool,
    pub menu_open: bool,
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

/// Draw a compact 30px identity bar with Inter SemiBold title/breadcrumb and right-aligned
/// 24x24 transparent-at-rest Preview Eye and Advanced Settings icon triggers.
pub(crate) fn draw_properties_header(
    ui: &Ui,
    icons: &GuiRibbonIcons,
    title: &str,
    popup_id: &str,
    presets_shown: &mut bool,
    preview: Option<(bool, bool)>,
    extra_items: impl FnOnce(&Ui),
) -> PropertiesHeaderOutcome {
    let mut outcome = PropertiesHeaderOutcome::default();
    let _semi = crate::ui_scale::ui_semibold_font(ui);
    ui.align_text_to_frame_padding();
    ui.text(title);
    drop(_semi);
    let right = ui.window_content_region_max()[0];
    let btn = 24.;
    if let Some((preview_hidden, preview_enabled)) = preview {
        ui.same_line_with_pos((right - btn * 2. - 4.).max(0.));
        outcome.toggle_preview =
            draw_preview_eye_button(ui, icons, preview_hidden, preview_enabled);
    }
    ui.same_line_with_pos((right - btn).max(0.));
    outcome.menu_open =
        draw_advanced_settings_button(ui, icons, popup_id, Some(presets_shown), extra_items);
    let [lo_x, y] = [ui.window_pos()[0], ui.cursor_screen_pos()[1]];
    let width = ui.window_size()[0];
    ui.get_window_draw_list()
        .add_line(
            [lo_x, y],
            [lo_x + width, y],
            crate::theme::border_subtle(ui),
        )
        .build();
    ui.dummy([0., 4.]);
    outcome
}

/// Draw the 24x24 icon-only transparent-at-rest Preview Eye button with hover tooltip.
pub(crate) fn draw_preview_eye_button(
    ui: &Ui,
    icons: &GuiRibbonIcons,
    preview_hidden: bool,
    enabled: bool,
) -> bool {
    let label = if preview_hidden {
        "Show Preview"
    } else {
        "Hide Preview"
    };
    let _disabled = ui.begin_disabled(!enabled);
    let active = enabled && !preview_hidden;
    let _bg = ui.push_style_color(
        StyleColor::Button,
        if active {
            crate::theme::brand_subtle(ui)
        } else {
            [0.; 4]
        },
    );
    let _border_col =
        active.then(|| ui.push_style_color(StyleColor::Border, crate::theme::brand_accent(ui)));
    let _bw = ui.push_style_var(imgui::StyleVar::FrameBorderSize(if active {
        1.
    } else {
        0.
    }));
    let clicked = widget_abi::icon_button(
        ui,
        &format!("##{label}"),
        IconButtonPresentation {
            texture: icons.key(IconGlyph::SketchVisibility, IconSize::Small),
            size: [24., 24.],
            icon: [16., 16.],
            large: false,
            show_caption: false,
        },
    );
    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text(label);
    }
    clicked
}

/// Draw the 24x24 icon-only transparent-at-rest Advanced Settings gear button and popup.
pub(crate) fn draw_advanced_settings_button(
    ui: &Ui,
    icons: &GuiRibbonIcons,
    popup_id: &str,
    presets_shown: Option<&mut bool>,
    extra_items: impl FnOnce(&Ui),
) -> bool {
    let _bg = ui.push_style_color(StyleColor::Button, [0.; 4]);
    let _bw = ui.push_style_var(imgui::StyleVar::FrameBorderSize(0.));
    if widget_abi::icon_button(
        ui,
        "##advanced-settings",
        IconButtonPresentation {
            texture: icons.key(IconGlyph::Preferences, IconSize::Small),
            size: [24., 24.],
            icon: [16., 16.],
            large: false,
            show_caption: false,
        },
    ) {
        ui.open_popup(popup_id);
    }
    if ui.is_item_hovered() {
        ui.tooltip_text("Advanced Settings");
    }
    let Some(_popup) = ui.begin_popup(popup_id) else {
        return false;
    };
    if let Some(shown) = presets_shown
        && ui
            .menu_item_config("Hide Presets")
            .selected(!*shown)
            .build()
    {
        *shown = !*shown;
    }
    extra_items(ui);
    true
}

/// Draw a Teal-tinted selection status pill for Profiles, Axes, Faces, or Edges.
pub(crate) fn draw_pick_pill(ui: &Ui, label: &str, active: bool) {
    let [x, y] = ui.cursor_screen_pos();
    let [tw, th] = ui.calc_text_size(label);
    let pad_x = 10.;
    let height = ui.frame_height();
    let width = (tw + pad_x * 2.).max(64.);
    let fill = if active {
        crate::theme::selection_subtle(ui)
    } else {
        crate::theme::surface_subtle(ui)
    };
    let border = if active {
        crate::theme::selection_accent(ui)
    } else {
        ui.style_color(StyleColor::Border)
    };
    let ink = if active {
        ui.style_color(StyleColor::Text)
    } else {
        crate::theme::text_secondary(ui)
    };
    let draw = ui.get_window_draw_list();
    draw.add_rect([x, y], [x + width, y + height], fill)
        .rounding(11.)
        .filled(true)
        .build();
    draw.add_rect([x, y], [x + width, y + height], border)
        .rounding(11.)
        .build();
    draw.add_text(
        [(x + pad_x).floor(), (y + (height - th) * 0.5).floor()],
        ink,
        label,
    );
    ui.dummy([width, height]);
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
    let is_surface = mode.is_surface();
    crate::theme::property_label(ui, "Type");
    let half = ((ui.content_region_avail()[0] - 4.) * 0.5).max(40.);
    let h = ui.frame_height();
    if draw_segment_choice(
        ui,
        &format!("Solid##{id_prefix}-solid"),
        !is_surface,
        [half, h],
    ) {
        *mode = if boolean_available {
            FeatureOutputMode::Join
        } else {
            FeatureOutputMode::NewSolid
        };
    }
    ui.same_line_with_spacing(0., 4.);
    if draw_segment_choice(
        ui,
        &format!("Surface##{id_prefix}-surface"),
        is_surface,
        [half, h],
    ) {
        *mode = FeatureOutputMode::Surface;
    }
    if mode.is_surface() {
        crate::theme::property_label(ui, "Body Name");
        ui.align_text_to_frame_padding();
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
        ui.align_text_to_frame_padding();
        ui.text(solid_name);
    }
    *mode != before
}

fn draw_segment_choice(ui: &Ui, label: &str, active: bool, size: [f32; 2]) -> bool {
    let fill = if active {
        crate::theme::brand_subtle(ui)
    } else {
        crate::theme::surface_subtle(ui)
    };
    let border = if active {
        crate::theme::brand_accent(ui)
    } else {
        ui.style_color(StyleColor::Border)
    };
    let _fill = ui.push_style_color(StyleColor::Button, fill);
    let _border = ui.push_style_color(StyleColor::Border, border);
    let _bw = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
    widget_abi::button(ui, label, size)
}

/// Draw a red validation error badge (`!`) next to an invalid field, with hover tooltip.
pub(crate) fn draw_error_badge(ui: &Ui, message: &str) {
    ui.same_line();
    let _color = widget_abi::push_style_color(ui, StyleColor::Text, crate::theme::error_accent(ui));
    ui.text("!");
    if ui.is_item_hovered() {
        ui.tooltip_text(message);
    }
}

/// Draw the unified 44px Properties footer row: Primary Indigo OK + bordered Cancel.
pub(crate) fn draw_properties_footer(ui: &Ui, ok_enabled: bool) -> PropertiesFooterOutcome {
    ui.separator();
    ui.dummy([0., 2.]);
    let mut outcome = PropertiesFooterOutcome::default();
    outcome.ok = crate::theme::property_accept(ui, ok_enabled);
    ui.same_line_with_spacing(0., 6.);
    outcome.cancel = crate::theme::property_cancel(ui);
    outcome
}
