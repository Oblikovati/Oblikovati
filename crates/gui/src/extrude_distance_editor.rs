// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    GuiCommandState, GuiExtrudeDirection, GuiInspectAction, GuiParameterRow, GuiShellAction,
    parameters::GuiParameterEditor,
};
use imgui::Ui;
use oblikovati_model::feature::FeatureId;

const MENU: &str = "extrude-distance-editor";
const PARAMETERS: &str = "Available Parameters";
const FEATURE_PROMPT: &str = "Select a feature and then one of its dimensions.";
const EDIT_BUTTON: &str = "Edit Distance A";

pub(super) fn draw_value(ui: &Ui, distance: &mut String) -> bool {
    let label = "Distance A";
    let style = ui.clone_style();
    let reserved = ui.calc_text_size(label)[0]
        + ui.calc_text_size(EDIT_BUTTON)[0]
        + style.item_inner_spacing[0]
        + style.item_spacing[0]
        + style.frame_padding[0] * 2.;
    ui.set_next_item_width((ui.content_region_avail()[0] - reserved).max(1.));
    ui.input_text(label, distance).build()
}

#[derive(Default)]
pub(super) struct DistanceEditor {
    needs_parameters: bool,
    picking_dimension: bool,
    issue: Option<String>,
}

#[derive(Clone, Copy)]
enum DistanceTool {
    Measure,
    Dimension,
    Parameters,
    Tolerance(u64),
    List,
}

impl DistanceEditor {
    pub(super) fn uses_parameters(&self) -> bool {
        self.needs_parameters
    }

    pub(super) fn prompt(&self) -> Option<&'static str> {
        self.picking_dimension.then_some(FEATURE_PROMPT)
    }

    pub(super) fn draw(
        &mut self,
        ui: &Ui,
        distance: &mut String,
        edited_feature: Option<FeatureId>,
        state: &GuiCommandState,
        parameters: &mut GuiParameterEditor,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        let Some(document) = state.document else {
            return false;
        };
        ui.same_line();
        if ui.small_button(EDIT_BUTTON) {
            self.needs_parameters = true;
            ui.open_popup(MENU);
        }
        if let Some(tool) = menu_item(ui, parameter_rows(parameters, document), distance) {
            self.apply_tool(ui, tool, parameters, document, actions);
        }
        if let Some(issue) = &self.issue {
            ui.text_wrapped(issue);
        }
        self.pick_distance(
            ui,
            parameter_rows(parameters, document),
            state,
            edited_feature,
        )
        .is_some_and(|name| {
            let changed = *distance != name;
            *distance = name;
            changed
        })
    }

    fn apply_tool(
        &mut self,
        ui: &Ui,
        tool: DistanceTool,
        parameters: &mut GuiParameterEditor,
        document: u64,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.picking_dimension = matches!(tool, DistanceTool::Dimension);
        self.issue = None;
        match tool {
            DistanceTool::Measure => {
                actions.push(GuiShellAction::Inspect(GuiInspectAction::StartMeasure))
            }
            DistanceTool::Parameters => parameters.open(Some(document)),
            DistanceTool::Tolerance(parameter) => {
                self.issue = parameters.open_tolerance(document, parameter).err();
            }
            DistanceTool::List => ui.open_popup(PARAMETERS),
            DistanceTool::Dimension => {}
        }
    }

    fn pick_distance(
        &mut self,
        ui: &Ui,
        rows: &[GuiParameterRow],
        state: &GuiCommandState,
        edited_feature: Option<FeatureId>,
    ) -> Option<String> {
        if let Some(_list) = ui.begin_popup(PARAMETERS) {
            ui.text(PARAMETERS);
            return choose_parameter(ui, rows.iter());
        }
        let selected = state.browser.as_ref()?.selected_feature?;
        if !self.picking_dimension || Some(selected) == edited_feature {
            return None;
        }
        let picked = feature_dimension(ui, rows, selected)?;
        self.picking_dimension = false;
        Some(picked)
    }
}

fn parameter_rows(parameters: &GuiParameterEditor, document: u64) -> &[GuiParameterRow] {
    parameters
        .snapshot(document)
        .map_or(&[], |snapshot| snapshot.rows.as_slice())
}

fn menu_item(ui: &Ui, rows: &[GuiParameterRow], distance: &str) -> Option<DistanceTool> {
    let _menu = ui.begin_popup(MENU)?;
    let parameter = rows
        .iter()
        .find(|row| {
            row.is_length && row.is_editable() && row.detail.parameter_info.name == distance.trim()
        })
        .map(|row| row.parameter);
    [
        ("Measure", Some(DistanceTool::Measure)),
        ("Select Feature Dimension", Some(DistanceTool::Dimension)),
        ("Parameters", Some(DistanceTool::Parameters)),
        ("Tolerance", parameter.map(DistanceTool::Tolerance)),
        ("List Parameters", Some(DistanceTool::List)),
    ]
    .into_iter()
    .find_map(|(label, tool)| {
        ui.menu_item_config(label)
            .enabled(tool.is_some())
            .build()
            .then_some(tool)
            .flatten()
    })
}

fn feature_dimension(ui: &Ui, rows: &[GuiParameterRow], feature: FeatureId) -> Option<String> {
    let _combo = ui.begin_combo("##feature-dimension", "Select a dimension")?;
    choose_parameter(
        ui,
        rows.iter()
            .filter(|row| row.feature_consumers.iter().any(|(id, _)| *id == feature)),
    )
}

fn choose_parameter<'a>(
    ui: &Ui,
    rows: impl Iterator<Item = &'a GuiParameterRow>,
) -> Option<String> {
    let mut any = false;
    for row in rows {
        any = true;
        let _id = ui.push_id_usize(row.parameter as usize);
        let name = &row.detail.parameter_info.name;
        let label = format!("{name}   {}", row.model_value_text);
        let _disabled = ui.begin_disabled(!row.is_length);
        let selected = ui.selectable(label);
        if selected {
            return Some(name.clone());
        }
    }
    if !any {
        ui.text("No parameters available");
    }
    None
}

pub(crate) struct ExtrudePanelControls<'a> {
    pub icons: &'a crate::icons::GuiRibbonIcons,
    pub parameters: &'a mut crate::parameters::GuiParameterEditor,
    pub popup_was_open: bool,
}

impl super::ExtrudeDialog {
    pub(super) fn draw_extent(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        icons: &crate::icons::GuiRibbonIcons,
        parameters: &mut crate::parameters::GuiParameterEditor,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        let mut edited = false;
        if self.reference.numeric() {
            edited |= draw_value(ui, &mut self.distance);
            edited |= self.distance_editor.draw(
                ui,
                &mut self.distance,
                self.edit_feature,
                state,
                parameters,
                actions,
            );
            edited |= self.direction_style.draw(
                ui,
                &mut self.direction,
                self.edit_feature.is_some(),
                icons,
            );
            if self.direction == GuiExtrudeDirection::Asymmetric {
                edited |= super::direction::draw_second_distance(
                    ui,
                    &mut self.distance,
                    &mut self.second_distance,
                );
            }
        }
        if self.edit_feature.is_none() {
            edited |= self.reference.draw(ui);
        }
        if !self.reference.numeric() {
            self.distance_editor = Default::default();
        }
        edited
    }
    pub(crate) fn distance_prompt(&self) -> Option<&'static str> {
        self.distance_editor.prompt()
    }
    pub(crate) fn uses_parameter_metadata(&self) -> bool {
        self.distance_editor.uses_parameters()
    }
}
