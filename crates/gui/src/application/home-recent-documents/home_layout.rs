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
                let [win_x, win_y] = ui.window_pos();
                let win_h = ui.window_size()[1];
                let divider_x = (win_x + rail_width - 8.).floor() + 0.5;
                ui.get_window_draw_list()
                    .add_line(
                        [divider_x, win_y],
                        [divider_x, win_y + win_h],
                        crate::theme::border_subtle(ui),
                    )
                    .build();
                self.rail(ui, state, actions, icons, rail_width - 28.);
                ui.same_line_with_pos(rail_width + 6.);
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
            ui.dummy([0., 8.]);
            {
                let _disabled = ui.begin_disabled(state.transaction_open);
                if crate::theme::primary_button(ui, "New Part##home-new-part", [width, 32.]) {
                    actions.push(GuiShellAction::NewPart);
                }
            }
            ui.dummy([0., 6.]);
            self.requests.new_document |= crate::theme::secondary_button(ui, "New", [width, 30.]);
            self.requests.open_document |= crate::theme::secondary_button(ui, "Open", [width, 30.]);
            if let Some(project) = state.projects.iter().find(|project| project.active) {
                self.project_card(ui, project, width);
            }
        });
    }

    fn project_card(&mut self, ui: &Ui, project: &crate::GuiProject, width: f32) {
        ui.dummy([0., 8.]);
        ui.text_colored(crate::theme::text_muted(ui), "Project");
        let name = super::recent_rows::fit_recent_cell(&project.name, width - 12., |text| {
            ui.calc_text_size(text)[0]
        });
        let label = format!("{name}###home-project");
        self.requests.open_projects |= crate::theme::secondary_button(ui, &label, [width, 34.]);
        if ui.is_item_hovered() {
            ui.tooltip_text(&project.name);
        }
    }

    pub(super) fn recent_toolbar(&mut self, ui: &Ui, icons: &GuiRibbonIcons) {
        {
            let _display = crate::ui_scale::display_font(ui);
            ui.text("Recent");
        }
        ui.dummy([0., 4.]);
        self.view_controls(ui);
        if ui.content_region_avail()[0] >= 300. {
            ui.same_line_with_spacing(0., 8.);
        }
        self.search_controls(ui, icons);
    }

    pub(super) fn view_controls(&mut self, ui: &Ui) {
        if view_button(ui, "List View", !self.grid) {
            self.grid = false;
        }
        ui.same_line_with_spacing(0., 0.);
        if view_button(ui, "Grid View", self.grid) {
            self.grid = true;
        }
        ui.same_line_with_spacing(0., 8.);
        self.filter_control(ui);
        ui.same_line_with_spacing(0., 6.);
        self.columns_control(ui);
    }
}

fn view_button(ui: &Ui, label: &str, active: bool) -> bool {
    let fill = if active {
        crate::theme::brand_subtle(ui)
    } else {
        crate::theme::surface_subtle(ui)
    };
    let border = if active {
        crate::theme::brand_accent(ui)
    } else {
        ui.style_color(StyleColor::Border)
    };
    let _fill = ui.push_style_color(StyleColor::Button, fill);
    let _edge = ui.push_style_color(StyleColor::Border, border);
    let _bw = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
    ui.button(label)
}
