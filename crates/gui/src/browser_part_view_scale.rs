// SPDX-License-Identifier: GPL-2.0-only
//! Validate and accept a custom annotation ratio for a part view (#4118).
use super::*;
use crate::dockable_window::DockableWindow;
use oblikovati_api::{
    contract::PartDesignView as _,
    types::{
        PART_ANNOTATION_SCALE_AUTO, PART_ANNOTATION_SCALE_CUSTOM, PART_ANNOTATION_SCALE_FIXED,
        PartAnnotationScale,
    },
};
use oblikovati_model::part_design_view::PartDesignView;

#[derive(Debug, Default)]
pub(crate) struct PartViewScaleDialog {
    draft: Option<ScaleDraft>,
    pending: bool,
    dock_state: crate::dockable_lifecycle::DockableState,
}
impl DockableWindow for PartViewScaleDialog {}
#[derive(Debug)]
struct ScaleDraft {
    document: u64,
    view: u64,
    text: String,
    focus: bool,
}
impl PartViewScaleDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Edit Annotation Scale", &self.dock_state)
    }
    pub(crate) fn needs_frame(&self) -> bool {
        self.pending
    }
    pub(crate) fn sync_before_docking(&mut self, state: &GuiCommandState, suspended: bool) {
        let missing = state.browser.as_ref().is_some_and(|browser| {
            self.draft.as_ref().is_some_and(|draft| {
                browser
                    .part_design_views
                    .as_ref()
                    .is_none_or(|views| !views.list().iter().any(|view| view.id() == draft.view))
            })
        });
        if self
            .draft
            .as_ref()
            .is_some_and(|draft| Some(draft.document) != state.document)
            || missing
        {
            self.close();
        }
        self.dock_state.set_suspended(suspended);
    }
    fn close(&mut self) {
        self.draft = None;
        self.pending = false;
        self.dock_state.set_open(false);
    }
    pub(super) fn menu(
        &mut self,
        ui: &Ui,
        document: u64,
        entry: &PartDesignView,
        actions: &mut Vec<GuiShellAction>,
    ) {
        ui.menu("Annotation Scale", || {
            let scale = entry.annotation_scale();
            scale_presets(ui, document, entry, &scale, actions);
            self.custom_choice(ui, document, entry, &scale);
        });
    }
    fn custom_choice(
        &mut self,
        ui: &Ui,
        document: u64,
        entry: &PartDesignView,
        scale: &PartAnnotationScale,
    ) {
        let label = if scale.mode == PART_ANNOTATION_SCALE_CUSTOM {
            format!(
                "Custom ({:.2})",
                f64::from(scale.numerator) / f64::from(scale.denominator)
            )
        } else {
            "Custom".into()
        };
        if ui
            .menu_item_config(&label)
            .selected(scale.mode == PART_ANNOTATION_SCALE_CUSTOM)
            .enabled(entry.active())
            .build()
        {
            self.open(document, entry.id(), scale);
        }
    }
    pub(super) fn open(
        &mut self,
        document: u64,
        view: u64,
        current: &oblikovati_api::types::PartAnnotationScale,
    ) {
        let text = if current.numerator == 0 {
            "Auto".into()
        } else {
            format!("{}:{}", current.numerator, current.denominator)
        };
        self.draft = Some(ScaleDraft {
            document,
            view,
            text,
            focus: true,
        });
        self.pending = true;
        self.dock_state.set_open(true);
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        document: Option<u64>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self
            .draft
            .as_ref()
            .is_some_and(|draft| Some(draft.document) != document)
        {
            self.close();
            return;
        }
        if !self.dock_state.is_open() {
            return;
        }
        let Some(draft) = self.draft.as_mut() else {
            return;
        };
        let focus = std::mem::take(&mut self.pending);
        let mut open = true;
        Self::center_dockable(ui, "Edit Annotation Scale", [350., 150.]);
        let outcome = Self::dockable_window(ui, "Edit Annotation Scale", &mut open)
            .focused(focus)
            .build(|| draw_editor(ui, draft, actions))
            .unwrap_or(false);
        if !open || outcome {
            self.close();
        }
    }
}

fn draw_editor(ui: &Ui, draft: &mut ScaleDraft, actions: &mut Vec<GuiShellAction>) -> bool {
    if std::mem::take(&mut draft.focus) {
        ui.set_keyboard_focus_here();
    }
    let invalid = draft.text != "Auto" && parse_ratio(&draft.text).is_none();
    let red = invalid.then(|| ui.push_style_color(StyleColor::Text, [1., 0.25, 0.25, 1.]));
    ui.input_text("Scale", &mut draft.text)
        .auto_select_all(true)
        .build();
    drop(red);
    let ratio = parse_ratio(&draft.text);
    let disabled = ui.begin_disabled(ratio.is_none());
    let accept = crate::widget_abi::button(ui, "OK", [80., 0.]);
    drop(disabled);
    ui.same_line();
    let cancel = crate::widget_abi::button(ui, "Cancel", [80., 0.])
        || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape));
    ui.same_line();
    let help_disabled = ui.begin_disabled(true);
    crate::widget_abi::button(ui, "Help", [80., 0.]);
    drop(help_disabled);
    if accept && let Some((numerator, denominator)) = ratio {
        actions.push(GuiShellAction::BrowserChoice {
            document: draft.document,
            choice: crate::GuiBrowserChoice::PartViewAnnotationScale {
                id: draft.view,
                numerator,
                denominator,
                custom: true,
            },
        });
        return true;
    }
    cancel
}
fn scale_presets(
    ui: &Ui,
    document: u64,
    entry: &PartDesignView,
    scale: &PartAnnotationScale,
    actions: &mut Vec<GuiShellAction>,
) {
    if ui
        .menu_item_config("Auto")
        .selected(scale.mode == PART_ANNOTATION_SCALE_AUTO)
        .enabled(entry.active())
        .build()
    {
        push_scale(actions, document, entry.id(), 0, 0);
    }
    for (label, numerator, denominator) in [
        ("1:1", 1, 1),
        ("1:2", 1, 2),
        ("1:5", 1, 5),
        ("1:10", 1, 10),
        ("2:1", 2, 1),
        ("5:1", 5, 1),
        ("10:1", 10, 1),
    ] {
        let selected = scale.mode == PART_ANNOTATION_SCALE_FIXED
            && scale.numerator == numerator
            && scale.denominator == denominator;
        if ui
            .menu_item_config(label)
            .selected(selected)
            .enabled(entry.active())
            .build()
        {
            push_scale(actions, document, entry.id(), numerator, denominator);
        }
    }
}
fn push_scale(
    actions: &mut Vec<GuiShellAction>,
    document: u64,
    id: u64,
    numerator: u32,
    denominator: u32,
) {
    actions.push(GuiShellAction::BrowserChoice {
        document,
        choice: crate::GuiBrowserChoice::PartViewAnnotationScale {
            id,
            numerator,
            denominator,
            custom: false,
        },
    });
}
fn parse_ratio(value: &str) -> Option<(u32, u32)> {
    let (numerator, denominator) = value.trim().split_once(':')?;
    let numerator = numerator.trim().parse::<u32>().ok()?;
    let denominator = denominator.trim().parse::<u32>().ok()?;
    (numerator > 0 && denominator > 0).then_some((numerator, denominator))
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn accepts_positive_ratios_and_rejects_zero_or_invalid_text() {
        assert_eq!(parse_ratio("3:2"), Some((3, 2)));
        for value in ["0", "1:0", "abc", "0:1", "1:2:3"] {
            assert_eq!(parse_ratio(value), None);
        }
    }
    #[test]
    fn document_switch_closes_the_editor_and_suspension_preserves_its_draft() {
        let mut dialog = PartViewScaleDialog::default();
        let scale = PartAnnotationScale {
            mode: PART_ANNOTATION_SCALE_AUTO,
            numerator: 0,
            denominator: 0,
        };
        dialog.open(1, 2, &scale);
        let mut state = GuiCommandState {
            document: Some(1),
            ..Default::default()
        };
        dialog.sync_before_docking(&state, true);
        assert!(dialog.draft.is_some());
        assert!(!dialog.dock_state.is_open());
        dialog.sync_before_docking(&state, false);
        assert!(dialog.dock_state.is_open());
        state.document = Some(3);
        dialog.sync_before_docking(&state, false);
        assert!(dialog.draft.is_none());
        assert!(!dialog.dock_state.is_open());
        assert!(!dialog.needs_frame());
    }
}
