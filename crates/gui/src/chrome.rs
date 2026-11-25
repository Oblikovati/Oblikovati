// SPDX-License-Identifier: GPL-2.0-only
use crate::application::ribbon_display::{RibbonDisplay, RibbonPlacement, TemporaryPanel};
use crate::ribbon::{DocumentRibbon, RibbonFrame};
use crate::{GuiCommandState, GuiFrameRequest, GuiShellAction};
use imgui::{Condition, TabItemFlags, Ui};
/// Vertical space reserved for application controls and the native frame.
pub(crate) const APPLICATION_BAR_HEIGHT: f32 = 34.;
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
    let height_total = request.extent[1] as f32 / request.layout_scale()[1];
    let scrollbar = owners.ribbon.panels.scrollbar_height(ui);
    let placement = owners.ribbon.display.placement;
    // A moved ribbon leaves only the menu, quick access and status in the top band.
    let row_growth =
        application_bar_height(owners.icons) - APPLICATION_BAR_HEIGHT + tab_row_height(ui) - 24.;
    let height = row_growth
        + if placement == RibbonPlacement::Top {
            owners.ribbon.display.mode.band_height(scrollbar)
                + if owners.ribbon.display.mode == RibbonDisplay::Full {
                    (20. + owners.ribbon.display.appearance.command_height())
                        * owners.icons.layout_factor()
                        - 80.
                } else {
                    0.
                }
        } else {
            RibbonDisplay::Tabs.band_height(scrollbar)
        };
    owners.ribbon.display.temporary_escape_consumed = false;
    let mut frame = owners.ribbon.prepare(&request.commands);
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
    crate::widget_abi::cursor_position(ui, [0., tab_top]);
    ribbon_menu_row(ui, request, owners, actions);
    if owners.ribbon.display.placement == RibbonPlacement::Top {
        ui.same_line();
        tab_overflow::draw(ui, owners.ribbon, frame);
        let control = [
            ui.window_size()[0] - 160. - crate::application::ribbon_display::PANEL_SELECTOR_WIDTH,
            tab_top,
        ];
        owners
            .ribbon
            .display
            .control(ui, control, &owners.ribbon.tips, open_display_menu);
        // #3856: adopt selector changes before the row takes its visibility snapshot.
        let panels = crate::ribbon_panels::available_panels(
            owners.ribbon.active,
            &request.commands,
            &owners.ribbon.customization,
        );
        owners
            .ribbon
            .display
            .panel_selector(ui, owners.ribbon.active, &panels);
        ribbon_panel_row(ui, request, owners, actions, tab_top + tab_row_height(ui));
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
    crate::widget_abi::cursor_position(ui, [8., 4.]);
    let width = (ui.window_size()[0] - APPLICATION_BAR_TRAILING_RESERVED - 8.).max(1.);
    let _padding = ui.push_style_var(imgui::StyleVar::WindowPadding([0., 0.]));
    ui.child_window("##application-bar-controls")
        .size([width, application_bar_height(owners.icons) - 4.])
        .scroll_bar(false)
        .scrollable(false)
        .build(|| {
            if !owners.quick_access.below_ribbon {
                owners.quick_access.row(
                    ui,
                    owners.icons,
                    &request.commands,
                    actions,
                    &owners.ribbon.tips,
                );
            }
            application_document_context(ui, &request.commands, !owners.quick_access.below_ribbon);
        });
}
fn application_document_context(ui: &Ui, state: &GuiCommandState, follows_toolbar: bool) {
    let title = state
        .documents
        .iter()
        .find(|item| Some(item.id) == state.document)
        .map_or("Oblikovati", |document| document.name.as_str());
    if ui.content_region_avail()[0] > ui.calc_text_size(title)[0] + 16. {
        if follows_toolbar {
            ui.same_line();
        }
        ui.text_disabled(title);
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
    ui.child_window("##ribbon-menu-row")
        .size([width, tab_row_height(ui)])
        .menu_bar(true)
        .build(|| {
            ui.menu_bar(|| {
                owners
                    .files
                    .menu(ui, &request.commands, actions, owners.ribbon.tips.tabs());
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
fn ribbon_tabs(
    ui: &Ui,
    ribbon: &mut DocumentRibbon,
    frame: &RibbonFrame,
    visible: &[crate::ribbon::RibbonTab],
    changed: bool,
) {
    let Some(_tabs) = ui.tab_bar("##document-ribbon-tabs") else {
        return;
    };
    let mut hovered = false;
    let mut selector_tab = None;
    let hidden_active = !visible.contains(&ribbon.active);
    let context_click = tab_click::context_button_event(ui);
    let restore = (changed || context_click).then_some(ribbon.active);
    let requested = frame.select.or(restore);
    for tab in visible {
        let selected = requested == Some(*tab);
        let flags = if selected {
            TabItemFlags::SET_SELECTED
        } else {
            TabItemFlags::empty()
        };
        let item = ui.tab_item_with_flags(tab.label(), None, flags);
        if ribbon.display.selector_pointer_press(ui) || ribbon.display.context_tab_press(ui) {
            selector_tab = Some(*tab);
        }
        let tip = crate::ribbon_key_tips::tab_tip(*tab).to_string();
        if ribbon.tips.tabs() && ribbon.tips.matching(&tip) {
            crate::ribbon_key_tips::badge(ui, &tip);
        }
        hovered |= ui.is_item_hovered();
        // On release: the press itself is a click outside any panel still open, and closes it.
        let released = ui.is_item_hovered() && ui.is_mouse_released(imgui::MouseButton::Left);
        let clicked = ui.is_item_hovered() && ui.is_mouse_clicked(imgui::MouseButton::Left);
        let through_popup = ribbon.tab_popup_press.released(ui, *tab);
        let double_click = clicked && ui.is_mouse_double_clicked(imgui::MouseButton::Left);
        tab_click::temporary_expansion_press(ribbon, *tab, clicked, double_click);
        if clicked && ribbon.active != *tab {
            ribbon.display.switch_from_open_compact_panel();
        }
        let doubled = double_click && ribbon.double_click_tab == Some(*tab);
        if clicked && !double_click {
            ribbon.double_click_tab = (ribbon.active == *tab).then_some(*tab);
        }
        if double_click {
            ribbon.double_click_tab = None;
        }
        if (hidden_active && clicked) || through_popup {
            ribbon.select_tab(*tab);
        }
        if item.is_some()
            && (!hidden_active || clicked || selected)
            && (requested.is_none() || clicked || selected)
        {
            ribbon.active = *tab;
        }
        drop(item);
        if doubled {
            ribbon.display.advance_from_tab();
        }
        if released && ribbon.display.take_tab_release() {
            ribbon.display.open_temporary(TemporaryPanel::WholeTab);
        }
    }
    ribbon.tab_popup_press.finish_frame(ui);
    if let Some(tab) = selector_tab.or(context_click.then_some(ribbon.active)) {
        ribbon.select_tab(tab);
    }
    if hovered {
        ribbon.scroll_tabs(&frame.tabs, ui.io().mouse_wheel);
    }
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
