// SPDX-License-Identifier: GPL-2.0-only
use crate::dockable_window::DockableWindow;
use crate::{GuiAtlas, GuiCommandState, GuiFrameRequest, GuiShellAction};
use imgui::{Context, FontConfig, FontGlyphRanges, FontId, FontSource, TextureId, Ui};
use oblikovati_model::ui_scale::UiScaleOptions;

// ImGui's default font is 13 pixels. Refuse unbounded atlas allocations at the input boundary.
const DEFAULT_FONT_PIXELS: f32 = 13.;
const MAX_FONT_PIXELS: f32 = 256.;
const INTER_REGULAR: &[u8] = include_bytes!("../assets/fonts/Inter-Regular.ttf");
const INTER_MEDIUM: &[u8] = include_bytes!("../assets/fonts/Inter-Medium.ttf");
const INTER_SEMIBOLD: &[u8] = include_bytes!("../assets/fonts/Inter-SemiBold.ttf");
const NUMERIC_FACE: &[u8] = include_bytes!("../assets/fonts/JetBrainsMono-Medium.ttf");
const CHROME_GLYPH_RANGES: &[u32] = &[
    0x0020, 0x007E, // Basic Latin
    0x00B0, 0x00B1, // degree °, plus-minus ±
    0x00D7, 0x00D8, // multiply ×, diameter Ø
    0x2013, 0x2014, // en-dash, em-dash
    0x2022, 0x2022, // bullet •
    0x2026, 0x2026, // ellipsis …
    0x2039, 0x203A, // single angle quotes ‹ ›
    0x2190, 0x2193, // arrows ← ↑ → ↓
    0x2212, 0x2212, // minus −
    0x22EE, 0x22EE, // vertical ellipsis ⋮
    0x2300, 0x2300, // diameter ⌀
    0x25B4, 0x25B4, // small up triangle ▴
    0x25B8, 0x25B8, // small right triangle ▸
    0x25BC, 0x25BC, // down triangle ▼
    0x25BE, 0x25BE, // small down triangle ▾
    0x2713, 0x2713, // check mark ✓
    0,
];
pub(crate) fn font_raster_scale(request: &GuiFrameRequest) -> Result<f32, String> {
    let scale = request.layout_scale().into_iter().fold(0., f32::max);
    let pixels = DEFAULT_FONT_PIXELS * scale;
    if !pixels.is_finite() || pixels <= 0. || pixels > MAX_FONT_PIXELS {
        return Err(format!(
            "GUI font size {pixels} pixels for layout scale {:?}: expected finite size in (0, {MAX_FONT_PIXELS}]",
            request.layout_scale()
        ));
    }
    Ok(scale)
}
pub(crate) fn rebuild_font_atlas(context: &mut Context, scale: f32) -> GuiAtlas {
    context.fonts().clear();
    install_editor_fonts(context, scale);
    context.io_mut().font_global_scale = scale.recip();
    let atlas = crate::engine::copy_atlas(context);
    // imgui-rs rebuild clears TexID; bind the new owned atlas before the next frame (#3850).
    context.fonts().tex_id = TextureId::new(crate::snapshot::FONT_TEXTURE_ID);
    atlas
}
fn chrome_font_config(size_pixels: f32, glyph_ranges: FontGlyphRanges) -> FontConfig {
    FontConfig {
        size_pixels,
        oversample_h: if size_pixels <= 26. { 2 } else { 1 },
        oversample_v: 1,
        pixel_snap_h: true,
        glyph_ranges,
        ..FontConfig::default()
    }
}
const UI_SYMBOL_FALLBACK_RANGES: &[u32] = &[
    0x22EE, 0x22EE, // vertical ellipsis ⋮
    0x25B4, 0x25B4, // small up triangle ▴
    0x25B8, 0x25B8, // small right triangle ▸
    0x25BE, 0x25BE, // small down triangle ▾
    0,
];
fn ui_symbol_fallback(size_pixels: f32) -> FontSource<'static> {
    FontSource::TtfData {
        data: NUMERIC_FACE,
        size_pixels,
        config: Some(chrome_font_config(
            size_pixels,
            FontGlyphRanges::from_slice(UI_SYMBOL_FALLBACK_RANGES),
        )),
    }
}
fn install_editor_fonts(context: &mut Context, scale: f32) {
    let base_pixels = DEFAULT_FONT_PIXELS * scale;
    let content_config = FontConfig {
        size_pixels: base_pixels,
        oversample_h: 1,
        oversample_v: 1,
        pixel_snap_h: true,
        ..Default::default()
    };
    let chrome_ranges = FontGlyphRanges::from_slice(CHROME_GLYPH_RANGES);
    let base_chrome = chrome_font_config(base_pixels, chrome_ranges.clone());
    let symbols = FontConfig {
        glyph_ranges: FontGlyphRanges::from_slice(&[0x2022, 0x2022, 0x2300, 0x2300, 0]),
        ..content_config.clone()
    };
    // Slot 0: Primary UI (Inter Regular 400)
    context.fonts().add_font(&[
        FontSource::TtfData {
            data: INTER_REGULAR,
            size_pixels: base_pixels,
            config: Some(base_chrome.clone()),
        },
        ui_symbol_fallback(base_pixels),
        sketch_symbols(symbols.clone()),
    ]);
    // Slots 1..=4: Sketch content faces (Liberation Sans Regular/Bold/Italic/BoldItalic)
    for face in [
        include_bytes!("../assets/fonts/LiberationSans-Regular.ttf").as_slice(),
        include_bytes!("../assets/fonts/LiberationSans-Bold.ttf").as_slice(),
        include_bytes!("../assets/fonts/LiberationSans-Italic.ttf").as_slice(),
        include_bytes!("../assets/fonts/LiberationSans-BoldItalic.ttf").as_slice(),
    ] {
        context.fonts().add_font(&[
            FontSource::TtfData {
                data: face,
                size_pixels: base_pixels,
                config: Some(content_config.clone()),
            },
            sketch_symbols(symbols.clone()),
        ]);
    }
    // Slot 5: Numeric / parameter value face (JetBrains Mono Medium 500)
    context.fonts().add_font(&[
        FontSource::TtfData {
            data: NUMERIC_FACE,
            size_pixels: base_pixels,
            config: Some(base_chrome.clone()),
        },
        sketch_symbols(symbols),
    ]);
    // Slot 6: Ribbon group caption (Inter Medium 500, 10px)
    context.fonts().add_font(&[
        FontSource::TtfData {
            data: INTER_MEDIUM,
            size_pixels: 10. * scale,
            config: Some(chrome_font_config(10. * scale, chrome_ranges.clone())),
        },
        ui_symbol_fallback(10. * scale),
    ]);
    // Slot 7: Status & key-tip micro monospace (JetBrains Mono Medium 500, 11px min)
    context.fonts().add_font(&[FontSource::TtfData {
        data: NUMERIC_FACE,
        size_pixels: 11. * scale,
        config: Some(chrome_font_config(11. * scale, chrome_ranges.clone())),
    }]);
    // Slot 8: Inter Medium 500 (active tabs, buttons, filenames)
    context.fonts().add_font(&[
        FontSource::TtfData {
            data: INTER_MEDIUM,
            size_pixels: base_pixels,
            config: Some(base_chrome.clone()),
        },
        ui_symbol_fallback(base_pixels),
    ]);
    // Slot 9: Inter SemiBold 600 (section headings, breadcrumbs)
    context.fonts().add_font(&[
        FontSource::TtfData {
            data: INTER_SEMIBOLD,
            size_pixels: base_pixels,
            config: Some(base_chrome),
        },
        ui_symbol_fallback(base_pixels),
    ]);
    // Slot 10: Inter SemiBold 600 14px (dialog titles)
    context.fonts().add_font(&[
        FontSource::TtfData {
            data: INTER_SEMIBOLD,
            size_pixels: 14. * scale,
            config: Some(chrome_font_config(14. * scale, chrome_ranges)),
        },
        ui_symbol_fallback(14. * scale),
    ]);
    // Slot 11: Inter SemiBold 600 20px (Home display heading)
    context.fonts().add_font(&[FontSource::TtfData {
        data: INTER_SEMIBOLD,
        size_pixels: 20. * scale,
        config: Some(FontConfig {
            size_pixels: 20. * scale,
            oversample_h: 1,
            oversample_v: 1,
            pixel_snap_h: true,
            ..FontConfig::default()
        }),
    }]);
}

fn sketch_symbols(config: FontConfig) -> FontSource<'static> {
    FontSource::TtfData {
        data: include_bytes!("../assets/fonts/OblikovatiSketchSymbols.ttf"),
        size_pixels: config.size_pixels,
        config: Some(config),
    }
}

/// Pick the editor face, for example `sketch_content_font(ui, true, true)` for Bold Italic.
pub(crate) fn sketch_content_font(ui: &Ui, bold: bool, italic: bool) -> Option<FontId> {
    // Atlas installation adds the chrome face first, then the four content faces in this order.
    let index = 1 + usize::from(bold) + 2 * usize::from(italic);
    ui.fonts().fonts().get(index).copied()
}

pub(crate) fn numeric_font(ui: &Ui) -> Option<imgui::FontStackToken<'_>> {
    ui.fonts().fonts().get(5).map(|font| ui.push_font(*font))
}

pub(crate) fn group_caption_font(ui: &Ui) -> Option<imgui::FontStackToken<'_>> {
    ui.fonts().fonts().get(6).map(|font| ui.push_font(*font))
}

pub(crate) fn mono_micro_font(ui: &Ui) -> Option<imgui::FontStackToken<'_>> {
    ui.fonts().fonts().get(7).map(|font| ui.push_font(*font))
}

pub(crate) fn ui_medium_font(ui: &Ui) -> Option<imgui::FontStackToken<'_>> {
    ui.fonts().fonts().get(8).map(|font| ui.push_font(*font))
}

pub(crate) fn ui_semibold_font(ui: &Ui) -> Option<imgui::FontStackToken<'_>> {
    ui.fonts().fonts().get(9).map(|font| ui.push_font(*font))
}

pub(crate) fn dialog_title_font(ui: &Ui) -> Option<imgui::FontStackToken<'_>> {
    ui.fonts().fonts().get(10).map(|font| ui.push_font(*font))
}

pub(crate) fn display_font(ui: &Ui) -> Option<imgui::FontStackToken<'_>> {
    ui.fonts().fonts().get(11).map(|font| ui.push_font(*font))
}

#[derive(Default)]
pub(crate) struct GuiScalePreferences {
    open: crate::dockable_lifecycle::DockableState,
    document_units: crate::document_units::GuiDocumentUnits,
    part_draft: crate::part_options_ui::PartOptionsDraft,
    sketch_draft: crate::sketch_options_ui::SketchOptionsDraft,
    #[cfg(test)]
    pub(crate) scale_200_bounds: Option<[[f32; 2]; 2]>,
}
impl DockableWindow for GuiScalePreferences {}
impl GuiScalePreferences {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Application Options", &self.open)?;
        self.sketch_draft.register_dockable(registry)?;
        self.document_units.register_dockable(registry)
    }
    /// Open the scale panel; example: the File menu's Options entry.
    pub(crate) fn request(&mut self) {
        self.open.set_open(true);
    }
    pub(crate) fn request_document(&mut self, document: u64) {
        self.document_units.request(document);
    }
    pub(crate) fn acknowledge_document_units(&mut self, document: u64, result: Result<(), String>) {
        self.document_units.acknowledge(document, result);
    }
    pub(crate) fn validate_constraint_settings(
        &self,
        action: &GuiShellAction,
    ) -> Result<(), String> {
        self.sketch_draft.validate_constraint_settings(action)
    }
    pub(crate) fn acknowledge_constraint_settings(
        &mut self,
        action: &GuiShellAction,
        active_document: Option<u64>,
        result: Result<(), String>,
    ) {
        self.sketch_draft
            .acknowledge_constraint_settings(action, active_document, result);
    }
    pub(crate) fn sync_before_docking(&mut self, state: &GuiCommandState, home_active: bool) {
        self.document_units.sync_before_docking(state, home_active);
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        home_active: bool,
        icons: &crate::icons::GuiRibbonIcons,
    ) {
        self.document_units.panel(ui, state, actions, home_active);
        let Some(options) = state.ui_scale.filter(|_| self.open.is_open()) else {
            self.part_draft.clear();
            self.sketch_draft.clear();
            return;
        };
        preference_geometry(ui);
        let mut open = self.open.is_open();
        let mut close_requested = false;
        let escape = Self::dockable_window(ui, "Application Options", &mut open)
            .scroll_bar(false)
            .scrollable(false)
            .build(|| {
                let body = crate::widget_abi::child_region(
                    ui,
                    "##application-options-body",
                    [0., -48.],
                    0.,
                    imgui::WindowFlags::empty(),
                );
                if body.visible {
                    {
                        let _semi = ui_semibold_font(ui);
                        ui.text("UI");
                    }
                    ui.separator();
                    ui.text_colored(crate::theme::text_secondary(ui), "Text size");
                    scale_choices(
                        ui,
                        options,
                        actions,
                        #[cfg(test)]
                        &mut self.scale_200_bounds,
                    );
                    self.part_draft.draw(ui, state.part_options);
                    self.sketch_draft.draw(ui, state);
                }
                drop(body);
                ui.separator();
                ui.dummy([0., 2.]);
                let dirty = self.part_draft.has_changes(state.part_options)
                    || self.sketch_draft.has_changes(state.sketch_options.as_ref());
                let conflict = self.part_draft.has_conflict(state.part_options)
                    || self
                        .sketch_draft
                        .has_conflict(state.sketch_options.as_ref());
                if crate::theme::property_accept(ui, !conflict) {
                    self.part_draft.apply(state.part_options, actions);
                    self.sketch_draft
                        .apply(state.sketch_options.as_ref(), actions);
                    close_requested = true;
                }
                ui.same_line_with_spacing(0., 6.);
                {
                    let _disabled = ui.begin_disabled(!dirty || conflict);
                    if crate::theme::secondary_button(ui, "Apply", [68., 28.]) {
                        self.part_draft.apply(state.part_options, actions);
                        self.sketch_draft
                            .apply(state.sketch_options.as_ref(), actions);
                    }
                }
                ui.same_line_with_spacing(0., 6.);
                if crate::theme::property_cancel(ui) {
                    self.part_draft.clear();
                    self.sketch_draft.clear();
                    close_requested = true;
                }
                ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape)
            })
            .unwrap_or(false);
        self.sketch_draft.draw_hud(ui);
        self.sketch_draft.draw_constraint(ui, state, actions, icons);
        if escape || close_requested {
            self.open.set_open(false);
        }
        if !open {
            self.open.set_open(false);
        }
        if !self.open.is_open() {
            self.part_draft.clear();
            self.sketch_draft.clear();
        }
    }
}
fn preference_geometry(ui: &Ui) {
    let [width, height] = ui.io().display_size;
    let size = [
        (width - 32.).clamp(1., 420.),
        (height - 32.).clamp(1., 660.),
    ];
    GuiScalePreferences::fit_dockable(
        ui,
        "Application Options",
        [(width - size[0]) * 0.5, (height - size[1]) * 0.5],
        size,
    );
}
fn scale_choices(
    ui: &Ui,
    options: UiScaleOptions,
    actions: &mut Vec<GuiShellAction>,
    #[cfg(test)] scale_200_bounds: &mut Option<[[f32; 2]; 2]>,
) {
    if ui.radio_button_bool("Automatic", options.saved_font().is_none()) {
        actions.push(GuiShellAction::SetUiScale(options.with_automatic_font()));
    }
    for percent in [75_u16, 100, 125, 150, 175, 200] {
        let scale = f32::from(percent) / 100.;
        let selected =
            ui.radio_button_bool(format!("{percent}%"), options.saved_font() == Some(scale));
        #[cfg(test)]
        if percent == 200 {
            *scale_200_bounds = Some([ui.item_rect_min(), ui.item_rect_max()]);
        }
        if selected {
            // Every choice is finite and within the model bounds; retain the independently stored icon factor.
            if let Ok(updated) = UiScaleOptions::new(scale, options.icon()) {
                actions.push(GuiShellAction::SetUiScale(updated));
            }
        }
    }
}
