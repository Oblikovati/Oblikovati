// SPDX-License-Identifier: GPL-2.0-only
use crate::dockable_window::DockableWindow;
use crate::{GuiAtlas, GuiCommandState, GuiFrameRequest, GuiShellAction};
use imgui::{Context, FontConfig, FontGlyphRanges, FontId, FontSource, TextureId, Ui};
use oblikovati_model::ui_scale::UiScaleOptions;

// ImGui's default font is 13 pixels. Refuse unbounded atlas allocations at the input boundary.
const DEFAULT_FONT_PIXELS: f32 = 13.;
const MAX_FONT_PIXELS: f32 = 256.;
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
fn install_editor_fonts(context: &mut Context, scale: f32) {
    let config = FontConfig {
        size_pixels: DEFAULT_FONT_PIXELS * scale,
        oversample_h: 1,
        oversample_v: 1,
        pixel_snap_h: true,
        ..Default::default()
    };
    let symbols = FontConfig {
        glyph_ranges: FontGlyphRanges::from_slice(&[0x2022, 0x2022, 0x2300, 0x2300, 0]),
        ..config.clone()
    };
    context.fonts().add_font(&[
        FontSource::TtfData {
            data: include_bytes!("../assets/fonts/Inter-Regular.ttf"),
            size_pixels: config.size_pixels,
            config: Some(FontConfig {
                glyph_ranges: FontGlyphRanges::from_slice(&[
                    0x20, 0xff, 0x2014, 0x2014, 0x203a, 0x203a, 0x25bc, 0x25bc, 0,
                ]),
                ..config.clone()
            }),
        },
        sketch_symbols(symbols.clone()),
    ]);
    for face in [
        include_bytes!("../assets/fonts/LiberationSans-Regular.ttf").as_slice(),
        include_bytes!("../assets/fonts/LiberationSans-Bold.ttf").as_slice(),
        include_bytes!("../assets/fonts/LiberationSans-Italic.ttf").as_slice(),
        include_bytes!("../assets/fonts/LiberationSans-BoldItalic.ttf").as_slice(),
    ] {
        context.fonts().add_font(&[
            FontSource::TtfData {
                data: face,
                size_pixels: config.size_pixels,
                config: Some(config.clone()),
            },
            sketch_symbols(symbols.clone()),
        ]);
    }
    context.fonts().add_font(&[
        FontSource::TtfData {
            data: include_bytes!("../assets/fonts/JetBrainsMono-Regular.ttf"),
            size_pixels: config.size_pixels,
            config: Some(config),
        },
        sketch_symbols(symbols),
    ]);
    context.fonts().add_font(&[FontSource::TtfData {
        data: include_bytes!("../assets/fonts/Inter-Regular.ttf"),
        size_pixels: 10. * scale,
        config: Some(FontConfig {
            size_pixels: 10. * scale,
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
            .build(|| {
                ui.text("UI");
                ui.separator();
                ui.text("Text size");
                scale_choices(
                    ui,
                    options,
                    actions,
                    #[cfg(test)]
                    &mut self.scale_200_bounds,
                );
                self.part_draft.draw(ui, state.part_options, actions);
                close_requested = self.sketch_draft.draw(ui, state, actions);
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
        (height - 32.).clamp(1., 500.),
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
    for percent in [75_u16, 100, 125, 150, 175, 200] {
        let scale = f32::from(percent) / 100.;
        let selected = ui.radio_button_bool(format!("{percent}%"), options.font() == scale);
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
