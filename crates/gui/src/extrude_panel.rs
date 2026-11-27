// SPDX-License-Identifier: GPL-2.0-only
//! Floating layout and controls for the Extrude properties window.
use super::{ExtrudeDialog, GuiCommandState, GuiShellAction};
use imgui::{Key, StyleColor, Ui};

const WIDTH: f32 = 280.;
const CLEAR_WIDTH: f32 = 44.;
pub(super) const FOOTER_HEIGHT: f32 = 44.;
const DEFAULT_HEIGHT: f32 = 340.;

pub(super) fn fit_properties(
    ui: &Ui,
    area: crate::application::document_creation::workspaces::WorkspaceArea,
    height: f32,
) {
    crate::widget_abi::fit_viewport_properties(ui, "Extrude", area, [WIDTH, height]);
}

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
        fit_properties(ui, area, height);
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
        let profile_count = self
            .edit_feature
            .and_then(|feature| state.browser.as_ref()?.extrusion_edits.get(&feature))
            .map_or(self.additional_profiles.len() + 1, |seed| {
                seed.profile_count
            });
        let label = if self.sketch.is_none() {
            "Select Profile".to_owned()
        } else if profile_count > 1 {
            format!("{profile_count} Profiles")
        } else if state.extrude_preview_active || self.preview_hidden || self.edit_feature.is_some()
        {
            "1 Profile".to_owned()
        } else {
            format!("Profile {}", self.profile.saturating_add(1))
        };
        let clear = self.sketch.filter(|_| self.edit_feature.is_none());
        let changed_profile = self.profile_combo(ui, &label, clear.is_some());
        let cleared = clear.is_some_and(|sketch| {
            ui.same_line();
            if !crate::widget_abi::button(
                ui,
                "Clear##extrude-profile-clear",
                [CLEAR_WIDTH, ui.frame_height()],
            ) {
                return false;
            }
            self.await_profile_pick(sketch);
            true
        });
        changed_sketch || changed_profile || cleared
    }

    fn profile_combo(&mut self, ui: &Ui, label: &str, can_clear: bool) -> bool {
        if can_clear {
            let spacing = ui.clone_style().item_spacing[0];
            ui.set_next_item_width((ui.content_region_avail()[0] - CLEAR_WIDTH - spacing).max(1.));
        }
        let _profile_disabled = ui.begin_disabled(self.sketch.is_none());
        let selected = self.sketch.is_some();
        let rounding = ui.push_style_var(imgui::StyleVar::FrameRounding(11.));
        let _bw = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
        let tint = ui.push_style_color(
            StyleColor::FrameBg,
            if selected {
                crate::theme::selection_subtle(ui)
            } else {
                crate::theme::surface_subtle(ui)
            },
        );
        let _edge = selected
            .then(|| ui.push_style_color(StyleColor::Border, crate::theme::selection_accent(ui)));
        let choices = ui.begin_combo("##profile", label);
        drop(_edge);
        drop(tint);
        drop(_bw);
        drop(rounding);
        choices.is_some_and(|_choices| {
            let _numeric = crate::ui_scale::numeric_font(ui);
            ui.text("Profile index");
            let changed = ui.input_int("##profile-index", &mut self.profile).build();
            if changed {
                self.additional_profiles
                    .retain(|&index| index != i64::from(self.profile));
                self.profile_pick_request = self.profile_pick_request.wrapping_add(1);
            }
            changed
        })
    }

    pub(super) fn controls(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        document: u64,
        interaction: (bool, bool),
        actions: &mut Vec<GuiShellAction>,
    ) {
        let (edited, shortcuts) = interaction;
        let preview = self.valid_preview_request(state, document, edited);
        let enabled = preview
            .as_ref()
            .is_some_and(|request| self.creation_action(request, false).dispatchable(state));
        let request = preview.as_ref().filter(|_| enabled);
        ui.separator();
        let enter = shortcuts
            && !self.enter_applies_value_only
            && (ui.is_key_pressed_no_repeat(Key::Enter)
                || ui.is_key_pressed_no_repeat(Key::KeypadEnter));
        let commit = crate::theme::property_accept(ui, enabled) || enter;
        if commit && self.commit_request(request, false, actions) {
            return;
        }

        ui.same_line_with_spacing(0., 6.);
        self.cancel_control(ui, shortcuts, actions);
        let right = ui.window_content_region_max()[0];
        ui.same_line_with_pos(right - 28.);
        let keep_sketch = !self.release_sketch_on_apply;
        let apply_request = preview.as_ref().filter(|request| {
            self.edit_feature.is_none()
                && self
                    .creation_action(request, keep_sketch)
                    .dispatchable(state)
        });
        let apply_more = crate::theme::property_apply_new(ui, apply_request.is_some());
        if apply_more && self.commit_request(apply_request, keep_sketch, actions) {
            self.requested = true;
            self.start_without_sketch = true;
        }
    }

    pub(super) fn toggle_preview(&mut self, actions: &mut Vec<GuiShellAction>) {
        self.preview_hidden = !self.preview_hidden;
        if self.preview_hidden {
            self.previewed = false;
            actions.push(GuiShellAction::PreviewExtrude(None));
        } else {
            // Rebuild through the current request and validation path.
            self.attempted = None;
        }
    }

    pub(super) fn commit_request(
        &mut self,
        request: Option<&crate::GuiExtrudePreview>,
        keep_sketch: bool,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        let Some(request) = request else {
            return false;
        };
        if let Some(edit) = self.edit_request(request) {
            self.submitted_edit = Some(edit.clone());
            actions.push(GuiShellAction::EditExtrudeDistance(edit));
        } else {
            self.presets.submit(request);
            let mut submitted = request.clone();
            submitted.generation = None;
            self.submitted = Some(submitted);
            actions.push(self.creation_action(request, keep_sketch));
        }
        let taper = self.taper.take();
        self.set_document(None);
        self.taper = taper;
        self.edit_feature = None;
        self.previewed = false;
        true
    }
    fn cancel_control(&mut self, ui: &Ui, shortcuts: bool, actions: &mut Vec<GuiShellAction>) {
        if crate::theme::property_cancel(ui)
            || (shortcuts && ui.is_key_pressed_no_repeat(Key::Escape))
        {
            self.close(actions);
        }
    }
}
