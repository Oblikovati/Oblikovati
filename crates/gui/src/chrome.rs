// SPDX-License-Identifier: GPL-2.0-only
use crate::application::ribbon_display::{RibbonDisplay, RibbonPlacement, TemporaryPanel};
use crate::ribbon::{DocumentRibbon, RibbonFrame};
use crate::{GuiCommandState, GuiFrameRequest, GuiShellAction};
use imgui::{Condition, Ui};
/// Vertical space reserved for application controls and the native frame.
pub(crate) const APPLICATION_BAR_HEIGHT: f32 = 32.;
/// Right side reserved for the platform window controls (#4063).
pub(crate) const APPLICATION_BAR_TRAILING_RESERVED: f32 = 150.;
/// Reserve enough space for the chosen quick-access icon scale.
pub(crate) fn application_bar_height(icons: &crate::icons::GuiRibbonIcons) -> f32 {
    APPLICATION_BAR_HEIGHT.max(icons.cell([24., 24.])[1] + 8.)
}
#[path = "chrome/model_click.rs"]
mod model_click;
mod side;
mod tab_click;
use model_click::process_ribbon_key_tips;
mod tab_overflow;
use side::{reserve_ribbon_margins, side_ribbon};
use tab_overflow::tab_row_height;
pub(crate) struct RibbonDrawOwners<'a> {
    pub home_active: bool,
    pub icons: &'a crate::icons::GuiRibbonIcons,
    pub ribbon: &'a mut DocumentRibbon,
    pub parameters: &'a mut crate::parameters::GuiParameterEditor,
    pub files: &'a mut crate::files::GuiDocumentFiles,
    pub theme_editor: &'a mut crate::theme_editor::GuiThemeEditor,
    pub preferences: &'a mut crate::ui_scale::GuiScalePreferences,
    pub precise_input: &'a mut crate::sketch::precise_input::PreciseInputBar,
    pub sketch_pattern: &'a mut crate::sketch::rectangular_pattern::RectangularPatternDialog,
    pub create_block: &'a mut crate::sketch::create_block::CreateBlockDialog,
    pub workspace: &'a mut crate::application::workspace_layout::WorkspaceLayout,
    pub workspaces: &'a mut crate::application::document_creation::workspaces::DocumentWorkspaces,
    pub quick_access: &'a mut crate::application::quick_access_toolbar::QuickAccessToolbar,
    pub plane_selection_active: bool,
}
pub(crate) fn ribbon_band(
    ui: &Ui,
    request: &GuiFrameRequest,
    mut owners: RibbonDrawOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
) -> f32 {
    owners.ribbon.display.appearance = owners.ribbon.customization.appearance();
    let width = request.extent[0] as f32 / request.layout_scale()[0];
    if owners.home_active && request.commands.document.is_some() {
        return home_band(ui, request, &mut owners, width, actions);
    }
    let height_total = request.extent[1] as f32 / request.layout_scale()[1];
    let placement = owners.ribbon.display.placement;
    let mut frame = owners.ribbon.prepare(&request.commands);
    // A moved ribbon leaves only the menu, quick access and status in the top band.
    let bar_height = application_bar_height(owners.icons);
    let tabs_height = tab_row_height(ui);
    let mode = if placement == RibbonPlacement::Top {
        owners.ribbon.display.mode
    } else {
        RibbonDisplay::Tabs
    };
    let height = if mode == RibbonDisplay::Full {
        bar_height
            + 1.
            + tabs_height
            + owners.ribbon.display.appearance.full_row_height() * owners.icons.layout_factor()
    } else {
        bar_height - APPLICATION_BAR_HEIGHT + tabs_height - 24. + mode.band_height()
    };
    owners.ribbon.display.temporary_escape_consumed = false;
    let blocked = ui.io().want_text_input
        || ui.io().key_ctrl
        || ui.io().key_super
        || crate::widget_abi::any_popup_open(ui);
    let grouped = owners.ribbon.display.placement != RibbonPlacement::Top
        || matches!(
            owners.ribbon.display.mode,
            RibbonDisplay::PanelButtons | RibbonDisplay::PanelTitles
        );
    owners.ribbon.tips.set_tools_face_mode(
        owners.ribbon.active == crate::ribbon::RibbonTab::Tools
            && owners.ribbon.display.mode == RibbonDisplay::Full
            && owners.ribbon.display.placement == RibbonPlacement::Top
            && !owners.home_active
            && !owners.ribbon.display.hidden_panels.contains(&(
                crate::ribbon::RibbonTab::Tools,
                crate::ribbon_panels::RibbonPanel::ToolOptions,
            ))
            && request.commands.selected_face_appearances.len() == 1,
    );
    let action =
        process_ribbon_key_tips(ui, request, &mut owners, &frame, blocked, grouped, height);
    let open_menu = apply_key_tip(action, &mut owners, &request.commands, &mut frame, actions);
    owners.quick_access.command_took_right_click = false;
    if let Some(_band) = begin_ribbon_band(ui, width, height) {
        ribbon_contents(ui, request, &mut owners, &frame, open_menu, actions);
        if !owners.quick_access.command_took_right_click {
            request_context_menu_when_clicked(ui, owners.ribbon);
        }
        let tab = owners.ribbon.active;
        let panels = crate::ribbon_panels::panels_for_placement(
            tab,
            &request.commands,
            &owners.ribbon.customization,
            owners.ribbon.display.placement,
        );
        owners.ribbon.display.context_menu(
            ui,
            tab,
            &panels,
            crate::ribbon_panels::supports_custom_panel(tab, &request.commands),
            &mut owners.ribbon.customization,
            (&request.commands, owners.quick_access),
        );
        owners.ribbon.apply_context_actions(tab);
        owners.ribbon.reset_confirmation(ui, &request.commands);
    }
    side_ribbon(
        ui,
        request,
        &mut owners,
        &frame,
        [width, height_total, height],
        actions,
    );
    reserve_ribbon_margins(&mut owners, width);
    if !owners.quick_access.below_ribbon {
        return height;
    }
    owners.quick_access.band_below(
        ui,
        ([0., height], width),
        owners.icons,
        &request.commands,
        actions,
        &owners.ribbon.tips,
    );
    height + owners.quick_access.below_height(owners.icons)
}

fn home_band(
    ui: &Ui,
    request: &GuiFrameRequest,
    owners: &mut RibbonDrawOwners<'_>,
    width: f32,
    actions: &mut Vec<GuiShellAction>,
) -> f32 {
    owners.workspace.reserve_for_ribbon(0., 0.);
    let top = application_bar_height(owners.icons) + 1.;
    let height = top + tab_row_height(ui);
    if let Some(_band) = begin_ribbon_band(ui, width, height) {
        application_controls_row(ui, request, owners, actions);
        crate::widget_abi::cursor_position(ui, [0., top]);
        ribbon_menu_row(ui, request, owners, actions);
    }
    if !owners.quick_access.below_ribbon {
        return height;
    }
    owners.quick_access.band_below(
        ui,
        ([0., height], width),
        owners.icons,
        &request.commands,
        actions,
        &owners.ribbon.tips,
    );
    height + owners.quick_access.below_height(owners.icons)
}
fn apply_key_tip(
    action: Option<crate::ribbon_key_tips::KeyTipAction>,
    owners: &mut RibbonDrawOwners<'_>,
    state: &GuiCommandState,
    frame: &mut RibbonFrame,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    use crate::ribbon_key_tips::KeyTipAction;
    match action {
        Some(KeyTipAction::SelectTab(tab)) => {
            owners.ribbon.select_tab(tab);
            frame.select = Some(tab);
            if owners.ribbon.display.mode == RibbonDisplay::Tabs {
                owners
                    .ribbon
                    .display
                    .open_temporary(TemporaryPanel::WholeTab);
            }
        }
        Some(KeyTipAction::CycleDisplay) => owners.ribbon.display.advance_from_tab(),
        Some(KeyTipAction::OpenDisplayMenu) => return true,
        Some(KeyTipAction::OpenFileMenu) => owners.files.request_menu(),
        Some(KeyTipAction::ApplicationOptions) => {
            if state.ui_scale.is_some() {
                owners.preferences.request();
            }
        }
        Some(KeyTipAction::DocumentSettings) => {
            if !state.transaction_open
                && let Some(id) = state.document.filter(|_| state.document_units.is_some())
            {
                owners.preferences.request_document(id);
            }
        }
        Some(KeyTipAction::OpenAdditional(panel)) => {
            let tab = owners.ribbon.active;
            if !owners.home_active && !owners.ribbon.display.hidden_panels.contains(&(tab, panel)) {
                owners.ribbon.panels.request_additional(tab, panel);
            }
        }
        Some(KeyTipAction::OpenGroup(panel)) => {
            if !owners.home_active
                && !owners
                    .ribbon
                    .display
                    .hidden_panels
                    .contains(&(owners.ribbon.active, panel))
                && crate::ribbon_panels::panels_for(owners.ribbon.active, state).contains(&panel)
            {
                owners.ribbon.display.request_keyboard_group(panel);
            }
        }
        Some(KeyTipAction::QuickAccess(index)) => {
            owners.quick_access.activate_tip(index, state, actions)
        }
        Some(KeyTipAction::StartSketch) => {
            let action = GuiShellAction::ChooseSketchPlane;
            if !owners.home_active && action.enabled(state) {
                owners.ribbon.display.temporary = None;
                actions.push(action);
            }
        }
        Some(KeyTipAction::Extrude)
            if !owners.home_active
                && crate::application::quick_access_toolbar::extrude_can_start(state) =>
        {
            owners.ribbon.display.temporary = None;
            actions.push(crate::extrude_dialog::start_action(state));
        }
        Some(KeyTipAction::MoveBody) => {
            let action = GuiShellAction::OpenMoveBody;
            if !owners.home_active && action.enabled(state) {
                owners.ribbon.panels.close_unpinned_additional();
                actions.push(action);
            }
        }
        Some(KeyTipAction::Hole) => {
            let action = GuiShellAction::OpenHole;
            if !owners.home_active && action.enabled(state) {
                owners.ribbon.display.temporary = None;
                actions.push(action);
            }
        }
        Some(KeyTipAction::Extrude) | None => {}
    }
    false
}
fn request_context_menu_when_clicked(ui: &Ui, ribbon: &mut DocumentRibbon) {
    let hovered = ui.is_window_hovered_with_flags(imgui::WindowHoveredFlags::CHILD_WINDOWS);
    if hovered && ui.is_mouse_clicked(imgui::MouseButton::Right) {
        ribbon
            .display
            .request_context_menu(ribbon.panels.context_title.take());
    }
}
fn begin_ribbon_band(ui: &Ui, width: f32, height: f32) -> Option<imgui::WindowToken<'_>> {
    // #3798: chrome owns a fixed band; wheel input must never move its commands.
    crate::widget_abi::window_geometry(ui, [0., 0.], [width, height], Condition::Always);
    // Keep focused ribbon commands below the window controls in the shared bar (#4063).
    ui.window("##application-ribbon")
        .flags(imgui::WindowFlags::NO_NAV_FOCUS)
        .bring_to_front_on_focus(false)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .collapsible(false)
        .scroll_bar(false)
        .scrollable(false)
        .begin()
}
fn ribbon_contents(
    ui: &Ui,
    request: &GuiFrameRequest,
    owners: &mut RibbonDrawOwners<'_>,
    frame: &RibbonFrame,
    open_display_menu: bool,
    actions: &mut Vec<GuiShellAction>,
) {
    application_controls_row(ui, request, owners, actions);
    let tab_top = application_bar_height(owners.icons) + 1.;
    let row_height = tab_row_height(ui);
    crate::widget_abi::cursor_position(ui, [0., tab_top]);
    ribbon_menu_row(ui, request, owners, actions);
    if owners.ribbon.display.placement == RibbonPlacement::Top {
        ui.same_line();
        tab_overflow::draw(ui, owners.ribbon, frame);
        let control = [
            ui.window_size()[0] - tab_overflow::RIBBON_CONTROL_WIDTH,
            tab_top + (row_height - 22.) * 0.5,
        ];
        let display = &mut owners.ribbon.display;
        display.control(
            ui,
            control,
            owners.icons,
            &owners.ribbon.tips,
            open_display_menu,
        );
        // #3856: adopt selector changes before the row takes its visibility snapshot.
        let panels = crate::ribbon_panels::available_panels(
            owners.ribbon.active,
            &request.commands,
            &owners.ribbon.customization,
        );
        let center = tab_top + row_height * 0.5;
        display.panel_selector(ui, owners.ribbon.active, &request.commands, &panels, center);
        ribbon_panel_row(ui, request, owners, actions, tab_top + row_height);
    }
    crate::theme_editor::panel(
        ui,
        request.commands.theme.as_ref(),
        owners.theme_editor,
        actions,
    );
}
fn application_controls_row(
    ui: &Ui,
    request: &GuiFrameRequest,
    owners: &mut RibbonDrawOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    crate::widget_abi::cursor_position(ui, [10., 4.]);
    let width = (ui.window_size()[0] - APPLICATION_BAR_TRAILING_RESERVED - 10.).max(1.);
    let _padding = ui.push_style_var(imgui::StyleVar::WindowPadding([0., 0.]));
    ui.child_window("##application-bar-controls")
        .size([width, application_bar_height(owners.icons) - 4.])
        .scroll_bar(false)
        .scrollable(false)
        .build(|| {
            owners
                .icons
                .brand_mark(ui, !owners.quick_access.below_ribbon);
            if !owners.quick_access.below_ribbon {
                owners.quick_access.row(
                    ui,
                    owners.icons,
                    &request.commands,
                    actions,
                    &owners.ribbon.tips,
                );
            }
            application_document_context(
                ui,
                &request.commands,
                application_bar_height(owners.icons),
            );
        });
}
fn application_document_context(ui: &Ui, state: &GuiCommandState, height: f32) {
    let title = state
        .documents
        .iter()
        .find(|item| Some(item.id) == state.document)
        .map_or_else(
            || "Oblikovati".to_owned(),
            |document| document_title(state, document),
        );
    let left = ui.item_rect_max()[0] + 12.;
    let right = ui.window_pos()[0] + ui.window_size()[0] - 8.;
    let screen_center = ui.io().display_size[0] / 2.;
    let title = crate::application::home_recent_documents::recent_rows::fit_recent_cell(
        &title,
        right - left,
        |text| ui.calc_text_size(text)[0],
    );
    let [width, text_height] = ui.calc_text_size(&title);
    if right > left + width {
        let x = (screen_center - width / 2.).clamp(left, right - width);
        ui.get_window_draw_list().add_text(
            [x, (height - text_height) / 2.],
            ui.style_color(imgui::StyleColor::Text),
            title,
        );
    }
}
fn document_title(
    state: &GuiCommandState,
    document: &oblikovati_api::wire::DocumentInfo,
) -> String {
    let changed = if document.dirty { "* " } else { "" };
    let model_state = state.document_model_states.get(&document.id);
    match model_state {
        Some(active) => format!("{changed}{} <{active}> — Oblikovati", document.name),
        None => format!("{changed}{} — Oblikovati", document.name),
    }
}
fn ribbon_menu_row(
    ui: &Ui,
    request: &GuiFrameRequest,
    owners: &mut RibbonDrawOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    let style = ui.clone_style();
    let width = (ui.calc_text_size("File")[0]
        + style.frame_padding[0] * 2.
        + style.item_spacing[0]
        + style.window_padding[0] * 2.)
        .max(40.);
    let height = tab_row_height(ui);
    let _padding = tab_overflow::tab_item_padding(ui);
    ui.child_window("##ribbon-menu-row")
        .size([width, height])
        .menu_bar(true)
        .build(|| {
            ui.menu_bar(|| {
                owners.files.menu(
                    ui,
                    &request.commands,
                    actions,
                    owners.ribbon.tips.tabs(),
                    owners.icons,
                );
            });
        });
}
fn ribbon_panel_row(
    ui: &Ui,
    request: &GuiFrameRequest,
    owners: &mut RibbonDrawOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
    top: f32,
) {
    crate::widget_abi::cursor_position(ui, [8., top]);
    let ribbon = &mut *owners.ribbon;
    let mut icons = owners.icons.clone();
    icons.appearance = ribbon.display.appearance;
    crate::ribbon_panels::draw_panels(
        ui,
        ribbon.active,
        &request.commands,
        crate::ribbon_panels::RibbonPanelOwners {
            home_active: owners.home_active,
            icons: &icons,
            parameters: owners.parameters,
            preferences: owners.preferences,
            editor: owners.theme_editor,
            workspace: owners.workspace,
            workspaces: owners.workspaces,
            quick_access: owners.quick_access,
            precise_input: owners.precise_input,
            sketch_pattern: owners.sketch_pattern,
            create_block: owners.create_block,
            customization: &mut ribbon.customization,
            team_web: &mut ribbon.team_web,
            active_tab: ribbon.active,
            plane_selection_active: owners.plane_selection_active,
            hole_tip: ribbon.tips.model_commands() && ribbon.tips.matching("HO"),
            modify_group_tip: ribbon.tips.model_commands() && ribbon.tips.matching("Y"),
            sketch_group_tip: ribbon.tips.model_commands() && ribbon.tips.matching("K"),
            create_group_tip: ribbon.tips.create_group_tip(ribbon.display.mode),
            key_tips_active: ribbon.tips.active(),
            sketch_tip: ribbon.tips.model_commands() && ribbon.tips.matching("SK"),
            extrude_tip: ribbon.tips.model_commands() && ribbon.tips.matching("EX"),
            tools_application_tip: ribbon.tips.tools_face_tip("OA"),
            tools_document_tip: ribbon.tips.tools_face_tip("OD"),
        },
        &mut ribbon.panels,
        &mut ribbon.display,
        actions,
    );
}
pub(crate) fn command_button(
    ui: &Ui,
    label: &str,
    action: GuiShellAction,
    state: &GuiCommandState,
    size: [f32; 2],
    actions: &mut Vec<GuiShellAction>,
) {
    command_control(ui, label, action, state, actions, || {
        crate::widget_abi::button(ui, label, size)
    });
}
pub(crate) fn command_control(
    ui: &Ui,
    label: &str,
    action: GuiShellAction,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    draw: impl FnOnce() -> bool,
) {
    let disabled = ui.begin_disabled(!action.enabled(state));
    let clicked = draw();
    drop(disabled);
    if ui.is_item_hovered()
        && let Some(next) = action.next_label(state)
    {
        ui.tooltip_text(format!("{label} {next}"));
    }
    if clicked && action.dispatchable(state) {
        actions.push(action);
    }
}
