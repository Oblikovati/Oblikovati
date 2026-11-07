// SPDX-License-Identifier: GPL-2.0-only
use crate::application::ribbon_display::{RibbonDisplay, RibbonPlacement, TemporaryPanel};
use crate::ribbon::{DocumentRibbon, RibbonFrame};
use crate::{GuiCommandState, GuiFrameRequest, GuiShellAction};
use imgui::{Condition, TabItemFlags, Ui};

pub(crate) struct RibbonDrawOwners<'a> {
    pub home_active: bool,
    pub icons: &'a crate::icons::GuiRibbonIcons,
    pub ribbon: &'a mut DocumentRibbon,
    pub parameters: &'a mut crate::parameters::GuiParameterEditor,
    pub files: &'a mut crate::files::GuiDocumentFiles,
    pub theme_editor: &'a mut crate::theme_editor::GuiThemeEditor,
    pub preferences: &'a mut crate::ui_scale::GuiScalePreferences,
    pub precise_input: &'a mut crate::sketch::precise_input::PreciseInputBar,
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
    let width = request.extent[0] as f32 / request.layout_scale()[0];
    let height_total = request.extent[1] as f32 / request.layout_scale()[1];
    let scrollbar = owners.ribbon.panels.scrollbar_height(ui);
    let placement = owners.ribbon.display.placement;
    // A moved ribbon leaves only the menu, quick access and status in the top band.
    let height = if placement == RibbonPlacement::Top {
        owners.ribbon.display.mode.band_height(scrollbar)
            + if owners.ribbon.display.mode == RibbonDisplay::Full {
                80. * (owners.icons.layout_factor() - 1.)
            } else {
                0.
            }
    } else {
        RibbonDisplay::Tabs.band_height(scrollbar)
    };
    let frame = owners.ribbon.prepare(&request.commands);
    owners.quick_access.command_took_right_click = false;
    if let Some(_band) = begin_ribbon_band(ui, width, height) {
        ribbon_contents(ui, request, &mut owners, &frame, actions);
        if !owners.quick_access.command_took_right_click {
            request_context_menu_when_clicked(ui, owners.ribbon);
        }
        let tab = owners.ribbon.active;
        let panels = crate::ribbon_panels::panels_for(tab, &request.commands);
        owners.ribbon.display.context_menu(ui, tab, &panels);
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
    owners
        .quick_access
        .band_below(ui, [0., height], width, &request.commands, actions);
    height + crate::application::quick_access_toolbar::BELOW_ROW_HEIGHT
}
fn request_context_menu_when_clicked(ui: &Ui, ribbon: &mut DocumentRibbon) {
    let hovered = ui.is_window_hovered_with_flags(imgui::WindowHoveredFlags::CHILD_WINDOWS);
    if hovered && ui.is_mouse_clicked(imgui::MouseButton::Right) {
        ribbon.display.request_context_menu();
    }
}
/// The workspace narrows by a docked ribbon; a floating one lies over it.
fn reserve_ribbon_margins(owners: &mut RibbonDrawOwners<'_>, _width: f32) {
    let (left, right) = match owners.ribbon.display.placement {
        RibbonPlacement::Left => (SIDE_RIBBON_WIDTH, 0.),
        RibbonPlacement::Right => (0., SIDE_RIBBON_WIDTH),
        RibbonPlacement::Top | RibbonPlacement::Floating => (0., 0.),
    };
    owners.workspace.reserve_for_ribbon(left, right);
}
const SIDE_RIBBON_WIDTH: f32 = 212.;
const SIDE_TABS_WIDTH: f32 = 84.;
const FLOATING_RIBBON_POSITION: [f32; 2] = [420., 240.];
/// Taller than wide, yet short enough that the fourth group shows only partly: it scrolls to it.
const FLOATING_RIBBON_HEIGHT: f32 = 230.;
/// The ribbon's tabs and group buttons as a column at a side, or in a floating panel.
fn side_ribbon(
    ui: &Ui,
    request: &GuiFrameRequest,
    owners: &mut RibbonDrawOwners<'_>,
    frame: &RibbonFrame,
    area: [f32; 3],
    actions: &mut Vec<GuiShellAction>,
) {
    let [width, height, band] = area;
    let (title, position, size, panel_x) = match owners.ribbon.display.placement {
        RibbonPlacement::Top => return,
        RibbonPlacement::Left => (
            "##ribbon-column",
            [0., band],
            [SIDE_RIBBON_WIDTH, height - band],
            (SIDE_RIBBON_WIDTH + 2., false),
        ),
        RibbonPlacement::Right => (
            "##ribbon-column",
            [width - SIDE_RIBBON_WIDTH, band],
            [SIDE_RIBBON_WIDTH, height - band],
            (width - SIDE_RIBBON_WIDTH - 2., true),
        ),
        RibbonPlacement::Floating => (
            "Ribbon",
            FLOATING_RIBBON_POSITION,
            [SIDE_RIBBON_WIDTH, FLOATING_RIBBON_HEIGHT],
            (FLOATING_RIBBON_POSITION[0] + SIDE_RIBBON_WIDTH + 2., false),
        ),
    };
    crate::widget_abi::window_geometry(ui, position, size, Condition::Always);
    let floating = owners.ribbon.display.placement == RibbonPlacement::Floating;
    let Some(_column) = ui
        .window(title)
        .flags(imgui::WindowFlags::NO_DOCKING | imgui::WindowFlags::NO_SCROLLBAR)
        .title_bar(floating)
        .movable(false)
        .resizable(false)
        .collapsible(false)
        .begin()
    else {
        return;
    };
    if floating {
        owners.ribbon.display.floating_title_menu(ui);
    }
    let title_bottom = ui.window_pos()[1] + ui.frame_height();
    if !floating || ui.io().mouse_pos[1] >= title_bottom {
        request_context_menu_when_clicked(ui, owners.ribbon);
    }
    side_tabs(ui, owners.ribbon, frame);
    ui.same_line_with_pos(SIDE_TABS_WIDTH);
    side_groups(ui, request, owners, panel_x, actions);
}
fn side_tabs(ui: &Ui, ribbon: &mut DocumentRibbon, frame: &RibbonFrame) {
    ui.group(|| {
        let mut hovered = false;
        for tab in &frame.tabs {
            let chosen = ui
                .selectable_config(tab.label())
                .selected(ribbon.active == *tab)
                .size([SIDE_TABS_WIDTH - 16., 0.])
                .build();
            hovered |= ui.is_item_hovered();
            if chosen {
                ribbon.active = *tab;
            }
        }
        if hovered {
            ribbon.scroll_tabs(&frame.tabs, ui.io().mouse_wheel);
        }
    });
}
fn side_groups(
    ui: &Ui,
    request: &GuiFrameRequest,
    owners: &mut RibbonDrawOwners<'_>,
    panel_x: (f32, bool),
    actions: &mut Vec<GuiShellAction>,
) {
    let ribbon = &mut *owners.ribbon;
    ui.child_window("##ribbon-groups").build(|| {
        crate::ribbon_panels::draw_side_groups(
            ui,
            ribbon.active,
            &request.commands,
            crate::ribbon_panels::RibbonPanelOwners {
                home_active: owners.home_active,
                icons: owners.icons,
                parameters: owners.parameters,
                preferences: owners.preferences,
                editor: owners.theme_editor,
                workspace: owners.workspace,
                workspaces: owners.workspaces,
                quick_access: owners.quick_access,
                precise_input: owners.precise_input,
                plane_selection_active: owners.plane_selection_active,
            },
            &mut ribbon.panels,
            &mut ribbon.display,
            panel_x,
            actions,
        );
    });
}
fn begin_ribbon_band(ui: &Ui, width: f32, height: f32) -> Option<imgui::WindowToken<'_>> {
    // #3798: chrome owns a fixed band; wheel input must never move its commands.
    crate::widget_abi::window_geometry(ui, [0., 0.], [width, height], Condition::Always);
    ui.window("##application-ribbon")
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .collapsible(false)
        .scroll_bar(false)
        .scrollable(false)
        .menu_bar(true)
        .begin()
}

fn ribbon_contents(
    ui: &Ui,
    request: &GuiFrameRequest,
    owners: &mut RibbonDrawOwners<'_>,
    frame: &RibbonFrame,
    actions: &mut Vec<GuiShellAction>,
) {
    ui.menu_bar(|| {
        owners.files.menu(ui, &request.commands, actions);
        owners
            .preferences
            .menu(ui, &request.commands, owners.home_active);
    });
    crate::widget_abi::cursor_position(ui, [16., 35.]);
    let row_above = !owners.quick_access.below_ribbon;
    if row_above {
        owners.quick_access.row(ui, &request.commands, actions);
    }
    if owners.ribbon.display.placement == RibbonPlacement::Top {
        if row_above {
            ui.same_line();
        }
        ribbon_tabs(ui, owners.ribbon, frame);
        let control = [
            ui.window_size()[0] - 160. - crate::application::ribbon_display::PANEL_SELECTOR_WIDTH,
            35.,
        ];
        owners.ribbon.display.control(ui, control);
        // #3856: adopt selector changes before the row takes its visibility snapshot.
        owners
            .ribbon
            .display
            .panel_selector(ui, owners.ribbon.active, &request.commands);
        ribbon_panel_row(ui, request, owners, actions);
    }
    crate::theme_editor::panel(
        ui,
        request.commands.theme.as_ref(),
        owners.theme_editor,
        actions,
    );
}
fn ribbon_panel_row(
    ui: &Ui,
    request: &GuiFrameRequest,
    owners: &mut RibbonDrawOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    crate::widget_abi::cursor_position(ui, [16., 59.]);
    let ribbon = &mut *owners.ribbon;
    crate::ribbon_panels::draw_panels(
        ui,
        ribbon.active,
        &request.commands,
        crate::ribbon_panels::RibbonPanelOwners {
            home_active: owners.home_active,
            icons: owners.icons,
            parameters: owners.parameters,
            preferences: owners.preferences,
            editor: owners.theme_editor,
            workspace: owners.workspace,
            workspaces: owners.workspaces,
            quick_access: owners.quick_access,
            precise_input: owners.precise_input,
            plane_selection_active: owners.plane_selection_active,
        },
        &mut ribbon.panels,
        &mut ribbon.display,
        actions,
    );
}
fn ribbon_tabs(ui: &Ui, ribbon: &mut DocumentRibbon, frame: &RibbonFrame) {
    let Some(_tabs) = ui.tab_bar("##document-ribbon-tabs") else {
        return;
    };
    let mut hovered = false;
    for tab in &frame.tabs {
        let flags = if frame.select == Some(*tab) {
            TabItemFlags::SET_SELECTED
        } else {
            TabItemFlags::empty()
        };
        let item = ui.tab_item_with_flags(tab.label(), None, flags);
        hovered |= ui.is_item_hovered();
        // On release: the press itself is a click outside any panel still open, and closes it.
        let released = ui.is_item_hovered() && ui.is_mouse_released(imgui::MouseButton::Left);
        let doubled = ui.is_item_hovered()
            && ui.is_mouse_double_clicked(imgui::MouseButton::Left)
            && ribbon.active == *tab;
        if item.is_some() {
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
