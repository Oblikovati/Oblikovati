// SPDX-License-Identifier: GPL-2.0-only
//! Feature Properties: a feature's name, when it is suppressed, and how it looks.
use crate::dockable_window::DockableWindow;
use crate::{GuiBrowserFeature, GuiCommandState, GuiModelBrowser, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::FeatureId;

const TITLE: &str = "Feature Properties";
const SIZE: [f32; 2] = [460., 380.];
const CONTROL_WIDTH: f32 = 210.;
const AS_BODY: &str = "As Body";
const COMPARISONS: [&str; 6] = [
    "equals",
    "not equals",
    "less than",
    "less than or equal",
    "greater than",
    "greater than or equal",
];

/// When a feature is suppressed by a parameter's value; the owner reads the value and name.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiFeatureCondition {
    pub parameter: String,
    /// A place in the dialog's list of comparisons.
    pub comparison: usize,
    pub value: String,
}
/// What the dialog changes on a feature when it is accepted.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiFeatureEdit {
    pub feature: FeatureId,
    pub name: String,
    pub always_suppressed: bool,
    pub condition: Option<GuiFeatureCondition>,
}
#[derive(Debug)]
struct Editing {
    feature: FeatureId,
    name: String,
    always: bool,
    conditional: bool,
    parameter: usize,
    comparison: usize,
    value: String,
    adaptive_sketch: bool,
    adaptive_parameters: bool,
    appearance: Option<String>,
}
impl Editing {
    fn of(feature: &GuiBrowserFeature) -> Self {
        Self {
            feature: feature.id,
            name: feature.name.clone(),
            always: feature.suppressed,
            conditional: false,
            parameter: 0,
            comparison: 0,
            value: String::new(),
            adaptive_sketch: false,
            adaptive_parameters: false,
            appearance: None,
        }
    }
    /// A condition with nothing to compare with is not finished.
    fn acceptable(&self) -> bool {
        !self.name.trim().is_empty() && (!self.conditional || !self.value.trim().is_empty())
    }
    fn edit(&self, parameter_names: &[String]) -> GuiFeatureEdit {
        let name = parameter_names.get(self.parameter).cloned();
        GuiFeatureEdit {
            feature: self.feature,
            name: self.name.trim().to_owned(),
            always_suppressed: self.always,
            condition: name
                .filter(|_| self.conditional)
                .map(|parameter| GuiFeatureCondition {
                    parameter,
                    comparison: self.comparison,
                    value: self.value.trim().to_owned(),
                }),
        }
    }
}
#[derive(Debug, Default)]
pub(crate) struct FeatureProperties {
    editing: Option<Editing>,
    opening: bool,
    document: Option<u64>,
    dock_state: crate::dockable_lifecycle::DockableState,
}
impl DockableWindow for FeatureProperties {}
impl FeatureProperties {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>(TITLE, &self.dock_state)
    }
    /// Open on a feature as it stands; example: from its row's Properties.
    pub(crate) fn request(&mut self, feature: &GuiBrowserFeature) {
        self.editing = Some(Editing::of(feature));
        self.opening = true;
        self.dock_state.set_open(true);
    }
    pub(crate) fn needs_frame(&self) -> bool {
        self.opening
    }
    pub(crate) fn sync_before_docking(&mut self, state: &GuiCommandState, suspended: bool) {
        if !self.opening && self.document.is_some() && self.document != state.document {
            self.editing = None;
            self.dock_state.set_open(false);
            self.document = None;
        }
        self.dock_state.set_suspended(suspended);
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        browser: &GuiModelBrowser,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if std::mem::take(&mut self.opening) {
            self.document = state.document;
        }
        if self.document.is_none() || self.document != state.document {
            self.editing = None;
            self.dock_state.set_open(false);
            self.document = None;
        }
        if self.editing.is_none() {
            return;
        }
        Self::center_dockable(ui, TITLE, SIZE);
        let mut open = true;
        let mut closed = false;
        Self::dockable_window(ui, TITLE, &mut open).build(|| {
            let editing = self
                .editing
                .as_mut()
                .expect("expected feature properties draft");
            let parameters: Vec<String> = state
                .parameters
                .iter()
                .map(|row| row.name.clone())
                .collect();
            ui.input_text("Name", &mut editing.name).build();
            suppress_group(ui, editing, &parameters);
            adaptive_group(ui, editing);
            appearance_row(ui, editing, browser);
            ui.separator();
            closed = answer_row(ui, state, editing, &parameters, actions)
                || ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape);
        });
        if closed || !open {
            self.editing = None;
            self.dock_state.set_open(false);
            self.document = None;
        }
    }
}
/// A row of a list: choosing it closes the list and not the dialog under it.
fn list_row(ui: &Ui, label: &str, selected: bool) -> bool {
    ui.selectable_config(label).selected(selected).build()
}
fn suppress_group(ui: &Ui, editing: &mut Editing, parameters: &[String]) {
    ui.text("Suppress");
    if ui.checkbox("Always", &mut editing.always) && editing.always {
        editing.conditional = false;
    }
    if ui.checkbox("If", &mut editing.conditional) && editing.conditional {
        editing.always = false;
    }
    let _muted = ui.begin_disabled(!editing.conditional);
    ui.set_next_item_width(CONTROL_WIDTH);
    let shown = parameters.get(editing.parameter).map_or("", String::as_str);
    if let Some(_list) = ui.begin_combo("Parameter", shown) {
        for (index, name) in parameters.iter().enumerate() {
            if list_row(ui, name, index == editing.parameter) {
                editing.parameter = index;
            }
        }
    }
    ui.set_next_item_width(CONTROL_WIDTH);
    if let Some(_list) = ui.begin_combo("Comparison", COMPARISONS[editing.comparison]) {
        for (index, name) in COMPARISONS.iter().enumerate() {
            if list_row(ui, name, index == editing.comparison) {
                editing.comparison = index;
            }
        }
    }
    ui.set_next_item_width(CONTROL_WIDTH);
    ui.input_text("Value##suppress", &mut editing.value).build();
}
fn adaptive_group(ui: &Ui, editing: &mut Editing) {
    ui.text("Adaptive");
    ui.checkbox("Sketch", &mut editing.adaptive_sketch);
    ui.same_line();
    ui.checkbox("Parameters", &mut editing.adaptive_parameters);
    ui.same_line();
    let _muted = ui.begin_disabled(true);
    let mut planes = false;
    ui.checkbox("From/To Planes", &mut planes);
}
fn appearance_row(ui: &Ui, editing: &mut Editing, browser: &GuiModelBrowser) {
    ui.set_next_item_width(CONTROL_WIDTH);
    let shown = editing.appearance.as_deref().unwrap_or(AS_BODY);
    let Some(_list) = ui.begin_combo("Feature Appearance", shown) else {
        return;
    };
    if list_row(ui, AS_BODY, editing.appearance.is_none()) {
        editing.appearance = None;
    }
    for choice in &browser.appearances {
        let chosen = editing.appearance.as_deref() == Some(choice.name.as_str());
        if list_row(ui, &choice.name, chosen) {
            editing.appearance = Some(choice.name.clone());
        }
    }
}
/// True when the dialog closes.
fn answer_row(
    ui: &Ui,
    state: &GuiCommandState,
    editing: &Editing,
    parameters: &[String],
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    let accepted = {
        let _muted = ui.begin_disabled(!editing.acceptable());
        crate::widget_abi::button(ui, "OK", [80., 0.])
    };
    ui.same_line();
    if crate::widget_abi::button(ui, "Cancel", [80., 0.]) {
        return true;
    }
    if accepted {
        actions.push(GuiShellAction::EditFeature {
            document: state.document.unwrap_or(0),
            edit: editing.edit(parameters),
        });
    }
    accepted
}

#[cfg(test)]
mod tests {
    #[test]
    fn document_change_closes_nested_window_before_docking() {
        let mut dialog = FeatureProperties {
            editing: Some(editing()),
            document: Some(1),
            ..Default::default()
        };
        dialog.dock_state.set_open(true);
        dialog.sync_before_docking(
            &GuiCommandState {
                document: Some(2),
                ..Default::default()
            },
            false,
        );
        assert!(dialog.editing.is_none());
        assert!(!dialog.dock_state.is_requested());
    }

    use super::*;
    fn editing() -> Editing {
        Editing::of(&GuiBrowserFeature {
            id: FeatureId(3),
            name: "Extrude1".into(),
            suppressed: false,
            extended: None,
            kind: "Extrusion",
        })
    }
    #[test]
    fn a_condition_needs_a_value_before_it_can_be_accepted() {
        let mut row = editing();
        assert!(row.acceptable());
        row.conditional = true;
        assert!(!row.acceptable());
        row.value = "5".into();
        assert!(row.acceptable());
    }
    #[test]
    fn an_edit_carries_the_condition_only_while_if_is_checked() {
        let mut row = editing();
        row.value = "5".into();
        let names = ["d0".to_string()];
        assert_eq!(row.edit(&names).condition, None);
        row.conditional = true;
        row.comparison = 5;
        let condition = row.edit(&names).condition.unwrap();
        assert_eq!(condition.parameter, "d0");
        assert_eq!(COMPARISONS[condition.comparison], "greater than or equal");
    }
}
