// SPDX-License-Identifier: GPL-2.0-only
//! Stage the two HUD input modes until the child window is accepted.
use crate::dockable_window::DockableWindow;
use imgui::Ui;
use oblikovati_model::application_options::SketchApplicationOptions;

#[derive(Clone, Debug, PartialEq)]
struct HudDraft {
    pointer_enabled: bool,
    pointer_cartesian: bool,
    dimension_enabled: bool,
    dimension_cartesian: bool,
}
impl HudDraft {
    fn from_options(options: &SketchApplicationOptions) -> Self {
        Self {
            pointer_enabled: options.pointer_input,
            pointer_cartesian: options.pointer_input_cartesian,
            dimension_enabled: options.dimension_input,
            dimension_cartesian: options.dimension_input_cartesian,
        }
    }
    fn apply(&self, options: &mut SketchApplicationOptions) {
        options.pointer_input = self.pointer_enabled;
        options.pointer_input_cartesian = self.pointer_cartesian;
        options.dimension_input = self.dimension_enabled;
        options.dimension_input_cartesian = self.dimension_cartesian;
    }
}

#[derive(Default)]
pub(crate) struct HudSettings {
    open: crate::dockable_lifecycle::DockableState,
    draft: Option<HudDraft>,
    focus_requested: bool,
}
impl DockableWindow for HudSettings {}
impl HudSettings {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("HUD Settings", &self.open)
    }
    pub(crate) fn request(&mut self, options: &SketchApplicationOptions) {
        self.draft = Some(HudDraft::from_options(options));
        self.open.set_open(true);
        self.focus_requested = true;
    }
    pub(crate) fn close(&mut self) {
        self.open.set_open(false);
        self.draft = None;
        self.focus_requested = false;
    }
    pub(crate) fn panel(&mut self, ui: &Ui, options: &mut SketchApplicationOptions) {
        if !self.open.is_open() {
            return;
        }
        let Some(draft) = self.draft.as_mut() else {
            self.close();
            return;
        };
        Self::center_dockable(ui, "HUD Settings", [390., 280.]);
        let mut open = true;
        let focus = std::mem::take(&mut self.focus_requested);
        let outcome = Self::dockable_window(ui, "HUD Settings", &mut open)
            .focused(focus)
            .build(|| draw_hud(ui, draft))
            .unwrap_or_default();
        if outcome.accept {
            draft.apply(options);
        }
        if !open || outcome.close {
            self.close();
        }
    }
}

#[derive(Default)]
struct HudOutcome {
    accept: bool,
    close: bool,
}
fn draw_hud(ui: &Ui, draft: &mut HudDraft) -> HudOutcome {
    ui.checkbox("Enable Pointer Input", &mut draft.pointer_enabled);
    coordinate_modes(
        ui,
        "Pointer",
        draft.pointer_enabled,
        &mut draft.pointer_cartesian,
    );
    ui.checkbox("Enable Dimension Input", &mut draft.dimension_enabled);
    coordinate_modes(
        ui,
        "Dimension",
        draft.dimension_enabled,
        &mut draft.dimension_cartesian,
    );
    let accept = ui.button("OK");
    ui.same_line();
    let cancel = ui.button("Cancel");
    let escape = ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape);
    HudOutcome {
        accept,
        close: accept || cancel || escape,
    }
}
fn coordinate_modes(ui: &Ui, name: &str, enabled: bool, cartesian: &mut bool) {
    let disabled = ui.begin_disabled(!enabled);
    if ui.radio_button_bool(format!("{name} Cartesian Coordinates"), *cartesian) {
        *cartesian = true;
    }
    if ui.radio_button_bool(format!("{name} Polar Coordinates"), !*cartesian) {
        *cartesian = false;
    }
    disabled.end();
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn accepting_hud_draft_preserves_other_sketch_options() {
        let original = SketchApplicationOptions::default();
        let mut draft = HudDraft::from_options(&original);
        draft.pointer_enabled = false;
        draft.dimension_cartesian = true;
        let mut options = SketchApplicationOptions {
            axes_visible: false,
            ..original
        };
        draft.apply(&mut options);
        assert!(!options.pointer_input && options.dimension_input_cartesian);
        assert!(!options.axes_visible && options.dimension_input);
    }
}
