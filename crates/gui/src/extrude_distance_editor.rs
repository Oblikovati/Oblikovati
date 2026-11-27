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
const EDIT_BUTTON_WIDTH: f32 = 24.;

fn draw_value(ui: &Ui, distance: &mut String, unit: Option<&str>, error: Option<[f32; 4]>) -> bool {
    crate::theme::property_label(ui, "Distance A");
    let available = ui.content_region_avail()[0];
    let spacing = ui.clone_style().item_spacing[0];
    ui.set_next_item_width((available - EDIT_BUTTON_WIDTH - spacing).max(1.));
    let _numeric = crate::ui_scale::numeric_font(ui);
    let _border = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
    let _error = error.map(|color| ui.push_style_color(imgui::StyleColor::Border, color));
    let edited = ui
        .input_text("##distance-a", distance)
        .auto_select_all(true)
        .build();
    if let Some(unit) =
        unit.filter(|_| distance.is_empty() || distance.trim().parse::<f64>().is_ok())
    {
        draw_unit_suffix(ui, distance, unit);
    }
    edited
}
fn draw_unit_suffix(ui: &Ui, value: &str, unit: &str) {
    let [end, _] = ui.item_rect_max();
    let [start, top] = ui.item_rect_min();
    let padding = ui.clone_style().frame_padding;
    let suffix_start = end - padding[0] - ui.calc_text_size(unit)[0];
    if suffix_start > start + 2. * padding[0] + ui.calc_text_size(value)[0] {
        ui.get_window_draw_list().add_text(
            [suffix_start, top + padding[1]],
            ui.style_color(imgui::StyleColor::TextDisabled),
            unit,
        );
    }
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
    Tolerance(Option<u64>),
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
        let engineering = state
            .extrude_distance_validation
            .as_ref()
            .filter(|current| {
                current.document == document
                    && Some(current.generation) == state.document_generation
                    && current.distance == *distance
            })
            .and_then(|current| current.engineering.as_ref());
        if let Some(tool) = menu_item(
            ui,
            parameter_rows(parameters, document),
            distance,
            engineering.is_some(),
        ) {
            if matches!(tool, DistanceTool::Tolerance(None)) {
                if let Some(engineering) = engineering {
                    parameters.open_tolerance_inspection(engineering.clone());
                }
            } else {
                parameters.close_tolerance_inspection();
            }
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
            if changed {
                parameters.close_tolerance_inspection();
            }
            changed
        })
    }

    fn menu_button(&mut self, ui: &Ui, icons: &crate::icons::GuiRibbonIcons) {
        ui.same_line();
        if icons.navigation_button(
            ui,
            EDIT_BUTTON,
            crate::icons::IconGlyph::Parameters,
            [EDIT_BUTTON_WIDTH, ui.frame_height()],
        ) {
            self.needs_parameters = true;
            ui.open_popup(MENU);
        }
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
            DistanceTool::Tolerance(Some(parameter)) => {
                self.issue = parameters.open_tolerance(document, parameter).err();
            }
            DistanceTool::List => ui.open_popup(PARAMETERS),
            DistanceTool::Dimension | DistanceTool::Tolerance(None) => {}
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

fn menu_item(
    ui: &Ui,
    rows: &[GuiParameterRow],
    distance: &str,
    inspectable: bool,
) -> Option<DistanceTool> {
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
        (
            "Tolerance",
            (parameter.is_some() || inspectable).then_some(DistanceTool::Tolerance(parameter)),
        ),
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
    pub area: crate::application::document_creation::workspaces::WorkspaceArea,
}

impl super::ExtrudeDialog {
    pub(super) fn breadcrumb(&self, ui: &Ui, state: &GuiCommandState) {
        let Some(browser) = &state.browser else {
            return;
        };
        let feature = browser
            .features
            .iter()
            .find(|row| Some(row.id) == self.edit_feature)
            .map_or("New Extrusion", |row| row.name.as_str());
        if let Some(sketch) = browser
            .sketches
            .iter()
            .find(|row| Some(row.id) == self.sketch)
        {
            ui.text(format!("{feature} › {}", sketch.name));
        }
    }

    fn distance_error_ink(&self, state: &GuiCommandState) -> Option<[f32; 4]> {
        let validation = self.current_distance_validation(state, state.document?)?;
        validation.error.as_ref()?;
        Some(
            state
                .theme
                .as_ref()
                .map_or([0.96, 0.25, 0.37, 1.], |theme| {
                    let color = &theme.palette()[&oblikovati_api::types::TOKEN_CHROME_DANGER];
                    [color.r, color.g, color.b, color.a]
                }),
        )
    }

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
            let error = self.distance_error_ink(state);
            let unit = state
                .document_units
                .as_ref()
                .map(|units| units.length_unit.as_str());
            edited |= draw_value(ui, &mut self.distance, unit, error);
            if edited {
                parameters.close_tolerance_inspection();
            }
            self.distance_editor.menu_button(ui, icons);
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
