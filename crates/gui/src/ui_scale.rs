// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiAtlas, GuiCommandState, GuiFrameRequest, GuiShellAction};
use imgui::{Context, FontConfig, FontSource, TextureId, Ui};
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
    context.fonts().add_font(&[FontSource::DefaultFontData {
        config: Some(FontConfig {
            size_pixels: DEFAULT_FONT_PIXELS * scale,
            oversample_h: 1,
            oversample_v: 1,
            pixel_snap_h: true,
            ..Default::default()
        }),
    }]);
    context.io_mut().font_global_scale = scale.recip();
    let atlas = crate::engine::copy_atlas(context);
    // imgui-rs rebuild clears TexID; bind the new owned atlas before the next frame (#3850).
    context.fonts().tex_id = TextureId::new(crate::snapshot::FONT_TEXTURE_ID);
    atlas
}
#[derive(Default)]
pub(crate) struct GuiScalePreferences {
    open: bool,
}
impl GuiScalePreferences {
    /// Open the scale panel; example: the File menu's Options entry.
    pub(crate) fn request(&mut self) {
        self.open = true;
    }
    pub(crate) fn menu(&mut self, ui: &Ui, state: &GuiCommandState) {
        ui.menu("Tools", || {
            if ui
                .menu_item_config("Application Options...")
                .enabled(state.ui_scale.is_some())
                .build()
            {
                self.open = true;
            }
        });
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(options) = state.ui_scale.filter(|_| self.open) else {
            return;
        };
        preference_geometry(ui);
        let escape = ui
            .window("Application Options")
            .opened(&mut self.open)
            .collapsible(false)
            .build(|| {
                ui.text("UI");
                ui.separator();
                ui.text("Text size");
                scale_choices(ui, options, actions);
                sketch_choices(ui, state, actions);
                ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape)
            })
            .unwrap_or(false);
        if escape {
            self.open = false;
        }
    }
}
fn preference_geometry(ui: &Ui) {
    let [width, height] = ui.io().display_size;
    let size = [
        (width - 32.).clamp(1., 420.),
        (height - 32.).clamp(1., 340.),
    ];
    crate::widget_abi::fit_dialog(
        ui,
        "Application Options",
        [(width - size[0]) * 0.5, (height - size[1]) * 0.5],
        size,
    );
}
/// Each choice goes to the owner, which keeps it across sessions.
fn sketch_choices(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(options) = &state.sketch_options else {
        return;
    };
    ui.separator();
    ui.text("Sketch");
    let mut shown = options.heads_up_display;
    if ui.checkbox("Sketch HUD", &mut shown) {
        actions.push(GuiShellAction::SetSketchOptions(
            oblikovati_model::application_options::SketchApplicationOptions {
                heads_up_display: shown,
                ..options.clone()
            },
        ));
    }
}
fn scale_choices(ui: &Ui, options: UiScaleOptions, actions: &mut Vec<GuiShellAction>) {
    for percent in [75_u16, 100, 125, 150, 175, 200] {
        let scale = f32::from(percent) / 100.;
        if ui.radio_button_bool(format!("{percent}%"), options.font() == scale) {
            // Every choice is finite and within the model bounds; retain the independently stored icon factor.
            if let Ok(updated) = UiScaleOptions::new(scale, options.icon()) {
                actions.push(GuiShellAction::SetUiScale(updated));
            }
        }
    }
}
