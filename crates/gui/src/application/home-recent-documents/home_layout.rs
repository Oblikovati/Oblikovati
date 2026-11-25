// SPDX-License-Identifier: GPL-2.0-only
//! Home's action rail and selection controls share the desktop chrome palette.
use super::*;
use crate::icons::GuiRibbonIcons;
use imgui::StyleColor;

impl HomeWorkspace {
    /// Draw the workspace over the document area while it is active.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        actions: &mut Vec<GuiShellAction>,
        icons: &GuiRibbonIcons,
    ) {
        self.sync_before_docking();
        if !self.active {
            return;
        }
        crate::widget_abi::window_geometry(
            ui,
            [area.left, area.top],
            [area.right - area.left, area.bottom - area.top],
            imgui::Condition::Always,
        );
        ui.window("##home-workspace")
            .flags(
                imgui::WindowFlags::NO_DECORATION
                    | imgui::WindowFlags::NO_MOVE
                    | imgui::WindowFlags::NO_DOCKING
                    | imgui::WindowFlags::NO_SAVED_SETTINGS
                    | imgui::WindowFlags::NO_FOCUS_ON_APPEARING
                    | imgui::WindowFlags::NO_BRING_TO_FRONT_ON_FOCUS,
            )
            .build(|| {
                let rail_width = ((area.right - area.left) * 0.27).clamp(122., 220.);
                self.rail(ui, state, actions, icons, rail_width - 24.);
                ui.same_line_with_pos(rail_width);
                ui.group(|| self.recent_area(ui, state, actions, icons));
                self.open_options.panel(ui, actions);
                self.properties_notice(ui);
            });
        self.side_panel_window(ui, area);
    }
    fn rail(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        icons: &GuiRibbonIcons,
        width: f32,
    ) {
        ui.group(|| {
            icons.brand_wordmark(ui, width);
            ui.dummy([0., 6.]);
            if state.document.is_none() {
                if crate::theme::primary_button(ui, "New Part##home-new-part", [width, 30.]) {
                    actions.push(GuiShellAction::NewPart);
                }
                self.requests.new_document |= ui.button_with_size("New", [width, 30.]);
            } else {
                self.requests.new_document |= crate::theme::primary_button(ui, "New", [width, 30.]);
            }
            self.requests.open_document |= ui.button_with_size("Open", [width, 30.]);
            if let Some(project) = state.projects.iter().find(|project| project.active) {
                self.project_card(ui, project, width);
            }
        });
    }

    fn project_card(&mut self, ui: &Ui, project: &crate::GuiProject, width: f32) {
        ui.dummy([0., 6.]);
        ui.text_disabled("Project");
        let name = super::recent_rows::fit_recent_cell(&project.name, width - 12., |text| {
            ui.calc_text_size(text)[0]
        });
        let label = format!("{name}###home-project");
        self.requests.open_projects |= ui.button_with_size(label, [width, 34.]);
        if ui.is_item_hovered() {
            ui.tooltip_text(&project.name);
        }
    }

    pub(super) fn view_controls(&mut self, ui: &Ui) {
        if view_button(ui, "List View", !self.grid) {
            self.grid = false;
        }
        ui.same_line();
        if view_button(ui, "Grid View", self.grid) {
            self.grid = true;
        }
        ui.same_line();
        self.filter_control(ui);
        ui.same_line();
        self.columns_control(ui);
    }
}

fn view_button(ui: &Ui, label: &str, active: bool) -> bool {
    let accent = ui.style_color(StyleColor::NavHighlight);
    let background = ui.style_color(StyleColor::Button);
    let fill: [f32; 4] =
        std::array::from_fn(|index| background[index] * 0.84 + accent[index] * 0.16);
    let _fill = active.then(|| ui.push_style_color(StyleColor::Button, fill));
    let _edge = active.then(|| ui.push_style_color(StyleColor::Border, accent));
    ui.button(label)
}
