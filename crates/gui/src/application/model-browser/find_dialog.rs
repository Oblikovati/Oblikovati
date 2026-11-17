// SPDX-License-Identifier: GPL-2.0-only
//! The Find dialog: build a list of criteria, then choose the row that meets them.
use super::find_criteria::{
    FindCriterion, FindJoin, FindObject, FindRow, NAME, add_grouped, conditions_of,
    first_condition_of, first_value_of, needs_grouping, row_meets, value_choices_of,
};
use super::find_prompts::{
    GROUP_TITLE, NO_MATCH_TITLE, OPEN_TITLE, PickerAnswer, SAVE_TITLE, SavedSearch, group_prompt,
    no_match_message, open_picker, save_prompt,
};
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiModelBrowser, GuiShellAction};
use imgui::Ui;

const SIZE: [f32; 2] = [620., 390.];
const CONTROL_WIDTH: f32 = 150.;
const LIST_HEIGHT: f32 = 130.;
const LOCATION: &str = "Active Document";
const EDGE_MARGIN: f32 = 10.;
const TOP: f32 = 170.;

/// The criterion being put together; it stays apart from the list until Add to list.
#[derive(Clone, Debug, PartialEq, Eq)]
struct PendingCriterion {
    property: &'static str,
    condition: &'static str,
    value: String,
    join: FindJoin,
}
impl PendingCriterion {
    fn of_property(property: &'static str) -> Self {
        Self {
            property,
            condition: first_condition_of(property),
            value: first_value_of(property),
            join: FindJoin::default(),
        }
    }
    fn of_listed(row: &FindCriterion) -> Self {
        Self {
            property: row.property,
            condition: row.condition,
            value: row.value.clone(),
            join: row.join,
        }
    }
    fn criterion(&self) -> FindCriterion {
        FindCriterion {
            property: self.property,
            condition: self.condition,
            value: self.value.clone(),
            join: self.join,
        }
    }
    /// A condition that names a value needs one; "any value" does not.
    fn complete(&self) -> bool {
        !self.value.trim().is_empty() || self.condition == "any value"
    }
}
impl Default for PendingCriterion {
    fn default() -> Self {
        Self::of_property(NAME)
    }
}
/// Open or not, the criteria stay for the session: reopening Find shows them again.
#[derive(Debug, Default)]
pub(crate) struct FindDialog {
    open: crate::dockable_lifecycle::DockableState,
    object: FindObject,
    match_case: bool,
    pending: PendingCriterion,
    criteria: Vec<FindCriterion>,
    selected: Option<usize>,
    /// What Add to list holds back while the grouping question stands.
    grouping: bool,
    opening_prompt: Option<&'static str>,
    saved: Vec<SavedSearch>,
    save_name: String,
    picked: usize,
}
impl DockableWindow for FindDialog {}
impl FindDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("###find-dialog", &self.open)
    }
    /// Open it, with whatever it held; example: the browser menu's Find.
    pub(crate) fn request(&mut self) {
        self.open.set_open(true);
    }
    /// A frame must follow so a prompt opens without waiting for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        self.opening_prompt.is_some()
    }
    pub(crate) fn set_suspended(&self, suspended: bool) {
        self.open.set_suspended(suspended);
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        browser: &GuiModelBrowser,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !self.open.is_open() {
            return;
        }
        let title = format!("Find: {}###find-dialog", self.object.label());
        let mut still_open = true;
        let mut window_open = true;
        // Beside the model, since what it finds is chosen behind it.
        Self::fit_dockable(
            ui,
            &title,
            [ui.io().display_size[0] - SIZE[0] - EDGE_MARGIN, TOP],
            SIZE,
        );
        Self::dockable_window(ui, &title, &mut window_open).build(|| {
            self.contents(ui, state, browser, &mut still_open, actions);
            self.prompts(ui);
        });
        self.open.set_open(still_open && window_open);
    }
    fn contents(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        browser: &GuiModelBrowser,
        still_open: &mut bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.object_row(ui);
        self.criterion_row(ui);
        self.add_row(ui);
        self.criteria_list(ui);
        ui.text_disabled(format!("Search in: {LOCATION}"));
        ui.separator();
        self.buttons(ui, state, browser, still_open, actions);
    }
    fn object_row(&mut self, ui: &Ui) {
        ui.set_next_item_width(CONTROL_WIDTH);
        if let Some(_list) = ui.begin_combo("Find##object", self.object.label()) {
            for (object, label) in FindObject::ALL {
                if ui
                    .selectable_config(label)
                    .selected(object == self.object)
                    .build()
                    && object != self.object
                {
                    self.object = object;
                    self.pending = PendingCriterion::default();
                    self.selected = None;
                }
            }
        }
        ui.same_line();
        ui.checkbox("Match case", &mut self.match_case);
        ui.same_line();
        ui.radio_button("And", &mut self.pending.join, FindJoin::And);
        ui.same_line();
        ui.radio_button("Or", &mut self.pending.join, FindJoin::Or);
    }
    fn criterion_row(&mut self, ui: &Ui) {
        ui.set_next_item_width(CONTROL_WIDTH);
        if let Some(_list) = ui.begin_combo("Property", self.pending.property) {
            for property in self.object.properties() {
                let chosen = *property == self.pending.property;
                if ui.selectable_config(property).selected(chosen).build() && !chosen {
                    let join = self.pending.join;
                    self.pending = PendingCriterion::of_property(property);
                    self.pending.join = join;
                }
            }
        }
        ui.same_line();
        ui.set_next_item_width(CONTROL_WIDTH);
        if let Some(_list) = ui.begin_combo("Condition", self.pending.condition) {
            for condition in conditions_of(self.pending.property) {
                let chosen = *condition == self.pending.condition;
                if ui.selectable_config(condition).selected(chosen).build() {
                    self.pending.condition = condition;
                }
            }
        }
        self.value_field(ui);
    }
    fn value_field(&mut self, ui: &Ui) {
        ui.set_next_item_width(CONTROL_WIDTH);
        let Some(choices) = value_choices_of(self.pending.property) else {
            ui.input_text("Value", &mut self.pending.value).build();
            return;
        };
        if let Some(_list) = ui.begin_combo("Value", &self.pending.value) {
            for choice in choices {
                let chosen = *choice == self.pending.value;
                if ui.selectable_config(choice).selected(chosen).build() {
                    self.pending.value = (*choice).to_owned();
                }
            }
        }
    }
    fn add_row(&mut self, ui: &Ui) {
        let added = self.pending.criterion();
        let listed = self.criteria.iter().any(|row| {
            row.property == added.property
                && row.condition == added.condition
                && row.value == added.value
        });
        let can_add = self.pending.complete() && !listed;
        let pressed = {
            let _muted = ui.begin_disabled(!can_add);
            crate::widget_abi::button(ui, "Add to list", [0.; 2])
        };
        if !pressed {
            return;
        }
        if needs_grouping(&self.criteria, &added) {
            self.grouping = true;
            self.opening_prompt = Some(GROUP_TITLE);
            return;
        }
        // An Or with nothing above it has nothing to group with.
        let mut added = added;
        if self.criteria.is_empty() {
            added.join = FindJoin::And;
        }
        self.criteria.push(added);
        self.selected = None;
    }
    fn criteria_list(&mut self, ui: &Ui) {
        ui.child_window("##find-criteria")
            .size([0., LIST_HEIGHT])
            .border(true)
            .build(|| {
                let mut chosen = None;
                for (index, row) in self.criteria.iter().enumerate() {
                    let label = format!("{}##criterion-{index}", row.row_text());
                    if ui
                        .selectable_config(&label)
                        .selected(self.selected == Some(index))
                        .build()
                    {
                        chosen = Some(index);
                    }
                }
                if let Some(index) = chosen {
                    self.selected = chosen;
                    self.pending = PendingCriterion::of_listed(&self.criteria[index]);
                }
            });
    }
    fn buttons(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        browser: &GuiModelBrowser,
        still_open: &mut bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        {
            let _muted = ui.begin_disabled(self.criteria.is_empty());
            if crate::widget_abi::button(ui, "Find Now", [0.; 2]) {
                self.find_now(state, browser, actions);
            }
            ui.same_line();
            if crate::widget_abi::button(ui, "Save Search", [0.; 2]) {
                self.save_name.clear();
                self.opening_prompt = Some(SAVE_TITLE);
            }
            ui.same_line();
        }
        if crate::widget_abi::button(ui, "Open Search", [0.; 2]) {
            self.picked = 0;
            self.opening_prompt = Some(OPEN_TITLE);
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Close", [0.; 2]) {
            *still_open = false;
        }
    }
    /// The first row met is chosen; with none, a message says so and the dialog stays.
    fn find_now(
        &mut self,
        state: &GuiCommandState,
        browser: &GuiModelBrowser,
        actions: &mut Vec<GuiShellAction>,
    ) {
        match self.first_match(state, browser) {
            Some(action) => actions.push(action),
            None => self.opening_prompt = Some(NO_MATCH_TITLE),
        }
    }
    fn first_match(
        &self,
        state: &GuiCommandState,
        browser: &GuiModelBrowser,
    ) -> Option<GuiShellAction> {
        let document = state.document?;
        match self.object {
            FindObject::Features => browser
                .features
                .iter()
                .find(|row| {
                    let facts = FindRow {
                        name: &row.name,
                        kind: row.kind,
                        suppressed: row.suppressed,
                        parameters: Vec::new(),
                    };
                    row_meets(&self.criteria, &facts, self.match_case)
                })
                .map(|row| GuiShellAction::SelectBrowserFeature {
                    document,
                    feature: row.id,
                }),
            FindObject::Sketches => browser
                .sketches
                .iter()
                .find(|row| {
                    let parameters = state.parameters.iter().map(|known| known.name.as_str());
                    let facts = FindRow {
                        name: &row.name,
                        parameters: parameters.collect(),
                        ..Default::default()
                    };
                    row_meets(&self.criteria, &facts, self.match_case)
                })
                .map(|row| GuiShellAction::SelectBrowserSketch {
                    document,
                    sketch: row.id,
                    toggle: false,
                }),
        }
    }
    fn prompts(&mut self, ui: &Ui) {
        if let Some(title) = self.opening_prompt.take() {
            ui.open_popup(title);
        }
        if let Some(yes) = group_prompt(ui) {
            if yes && self.grouping {
                add_grouped(&mut self.criteria, self.pending.criterion());
            }
            self.grouping = false;
        }
        no_match_message(ui);
        if let Some(name) = save_prompt(ui, &mut self.save_name) {
            self.saved.push(SavedSearch {
                name,
                object: self.object,
                match_case: self.match_case,
                criteria: self.criteria.clone(),
            });
        }
        match open_picker(ui, &self.saved, &mut self.picked) {
            Some(PickerAnswer::Open(index)) => self.open_saved(index),
            Some(PickerAnswer::Delete(index)) => {
                if index < self.saved.len() {
                    self.saved.remove(index);
                }
                self.picked = 0;
            }
            None => {}
        }
    }
    fn open_saved(&mut self, index: usize) {
        let Some(search) = self.saved.get(index) else {
            return;
        };
        self.object = search.object;
        self.match_case = search.match_case;
        self.criteria = search.criteria.clone();
        self.selected = None;
    }
}
