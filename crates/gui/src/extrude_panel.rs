// SPDX-License-Identifier: GPL-2.0-only
//! Floating layout and controls for the Extrude properties window.
use super::{ExtrudeDialog, GuiCommandState, GuiShellAction};
use crate::icons::{GuiRibbonIcons, IconGlyph, IconSize};
use imgui::{Key, StyleColor, Ui};

const WIDTH: f32 = 280.;
pub(super) const FOOTER_HEIGHT: f32 = 44.;
const DEFAULT_HEIGHT: f32 = 340.;

#[derive(Default)]
pub(super) struct PropertiesSize {
    content: Option<f32>,
    pending: bool,
}
impl PropertiesSize {
    pub(super) fn fit(
        &mut self,
        ui: &Ui,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
    ) {
        let height = self.content.unwrap_or(DEFAULT_HEIGHT);
        crate::widget_abi::fit_viewport_properties(ui, "Extrude", area, [WIDTH, height]);
        self.pending = false;
    }

    pub(super) fn measure(&mut self, chrome: f32, properties: f32) {
        let height = (chrome + properties + FOOTER_HEIGHT).ceil();
        self.pending = self.content != Some(height);
        self.content = Some(height);
    }

    pub(super) fn needs_frame(&self) -> bool {
        self.pending
    }
}

impl ExtrudeDialog {
    pub(super) fn geometry_controls(&mut self, ui: &Ui, state: &GuiCommandState) -> bool {
        let _disabled = ui.begin_disabled(self.edit_feature.is_some());
        let changed_sketch = self.sketch_choice(ui, state);
        crate::theme::property_label(ui, "Profiles");
        let label = if state.extrude_preview_active {
            "1 Profile".to_owned()
        } else {
            format!("Profile {}", self.profile.saturating_add(1))
        };
        let rounding = ui.push_style_var(imgui::StyleVar::FrameRounding(11.));
        let tint = ui.push_style_color(StyleColor::FrameBg, [0.05, 0.58, 0.53, 0.22]);
        let choices = ui.begin_combo("##profile", &label);
        drop(tint);
        drop(rounding);
        let changed_profile = choices.is_some_and(|_choices| {
            let _numeric = crate::ui_scale::numeric_font(ui);
            ui.text("Profile index");
            ui.input_int("##profile-index", &mut self.profile).build()
        });
        changed_sketch || changed_profile
    }

    pub(super) fn controls(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        document: u64,
        interaction: (bool, bool),
        presentation: &GuiRibbonIcons,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let (edited, shortcuts) = interaction;
        let preview = self.valid_preview_request(state, document, edited);
        let enabled = preview
            .as_ref()
            .is_some_and(|request| self.creation_action(request).dispatchable(state));
        let request = preview.as_ref().filter(|_| enabled);
        ui.separator();
        let commit = crate::theme::property_accept(ui, enabled)
            || (shortcuts && ui.is_key_pressed_no_repeat(Key::Enter));
        if commit && self.commit_request(request, actions) {
            return;
        }

        ui.same_line_with_spacing(0., 6.);
        self.cancel_control(ui, shortcuts, actions);
        ui.same_line_with_spacing(0., 6.);
        let disabled = ui.begin_disabled(!enabled);
        self.preview_control(
            ui,
            request,
            presentation,
            state.extrude_preview_active,
            actions,
        );
        drop(disabled);

        let right = ui.window_content_region_max()[0];
        ui.same_line_with_pos(right - 28.);
        let disabled = ui.begin_disabled(!enabled || self.edit_feature.is_some());
        let _base = ui.push_style_color(StyleColor::Button, [0.13, 0.77, 0.37, 1.]);
        let _hovered = ui.push_style_color(StyleColor::ButtonHovered, [0.29, 0.87, 0.50, 1.]);
        let _ink = ui.push_style_color(StyleColor::Text, [0.05, 0.15, 0.08, 1.]);
        let apply_more = crate::widget_abi::button(ui, "+##apply-create-new", [28., 28.]);
        drop(disabled);
        if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
            ui.tooltip_text("Apply & Create New");
        }
        if apply_more && self.commit_request(request, actions) {
            self.requested = true;
        }
    }

    fn preview_control(
        &mut self,
        ui: &Ui,
        request: Option<&crate::GuiExtrudePreview>,
        icons: &GuiRibbonIcons,
        preview_active: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let pressed = crate::widget_abi::icon_button(
            ui,
            "Preview",
            crate::widget_abi::IconButtonPresentation {
                texture: icons.key(IconGlyph::SketchVisibility, IconSize::Small),
                size: [74., 28.],
                icon: [16.; 2],
                large: false,
                show_caption: true,
            },
        );
        if ui.is_item_hovered() {
            let tooltip = if preview_active {
                "Hide preview"
            } else {
                "Show preview"
            };
            ui.tooltip_text(tooltip);
        }
        if !pressed {
            return;
        }
        if preview_active {
            self.preview_hidden = true;
            self.previewed = false;
            actions.push(GuiShellAction::PreviewExtrude(None));
        } else if let Some(request) = request {
            self.preview_hidden = false;
            let action = self.edit_request(request).map_or_else(
                || GuiShellAction::PreviewExtrude(Some(request.clone())),
                GuiShellAction::PreviewExtrudeEdit,
            );
            actions.push(action);
            self.previewed = true;
        }
    }

    fn cancel_control(&mut self, ui: &Ui, shortcuts: bool, actions: &mut Vec<GuiShellAction>) {
        let _outline = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
        if crate::widget_abi::button(ui, "Cancel", [62., 28.])
            || (shortcuts && ui.is_key_pressed_no_repeat(Key::Escape))
        {
            self.close(actions);
        }
    }
}
