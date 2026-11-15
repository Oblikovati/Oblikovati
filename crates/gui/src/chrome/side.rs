// SPDX-License-Identifier: GPL-2.0-only
use super::{RibbonDrawOwners, request_context_menu_when_clicked};
use crate::application::ribbon_display::{RibbonDisplayState, RibbonPlacement};
use crate::ribbon::{DocumentRibbon, RibbonFrame};
use crate::{GuiFrameRequest, GuiShellAction};
use imgui::{Condition, Ui};

/// The workspace narrows by a docked ribbon; a floating one lies over it.
pub(super) fn reserve_ribbon_margins(owners: &mut RibbonDrawOwners<'_>, _width: f32) {
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
pub(super) fn side_ribbon(
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
    let cursor = ui.cursor_pos();
    let selector_was_open = owners.ribbon.display.selector_open;
    let panels = crate::ribbon_panels::available_panels(
        owners.ribbon.active,
        &request.commands,
        &owners.ribbon.customization,
    );
    owners
        .ribbon
        .display
        .side_panel_selector(ui, owners.ribbon.active, &panels);
    crate::widget_abi::cursor_position(ui, cursor);
    side_tabs(ui, owners.ribbon, frame, selector_was_open);
    ui.same_line_with_pos(SIDE_TABS_WIDTH);
    side_groups(ui, request, owners, panel_x, actions);
}
fn side_tabs(ui: &Ui, ribbon: &mut DocumentRibbon, frame: &RibbonFrame, selector_was_open: bool) {
    ui.group(|| {
        let mut hovered = false;
        for tab in &frame.tabs {
            let chosen = ui
                .selectable_config(tab.label())
                .selected(ribbon.active == *tab)
                .size([SIDE_TABS_WIDTH - 16., 0.])
                .build();
            let context_press = ribbon.display.context_tab_press(ui);
            let selector_press =
                RibbonDisplayState::selector_pointer_press_when(ui, selector_was_open);
            let tip = crate::ribbon_key_tips::tab_tip(*tab).to_string();
            if ribbon.tips.tabs() && ribbon.tips.matching(&tip) {
                crate::ribbon_key_tips::badge(ui, &tip);
            }
            hovered |= ui.is_item_hovered();
            if chosen {
                ribbon.active = *tab;
            } else if context_press || selector_press {
                ribbon.select_tab(*tab);
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
    let mut icons = owners.icons.clone();
    icons.appearance = ribbon.display.appearance;
    ui.child_window("##ribbon-groups")
        .size([0., -32.])
        .build(|| {
            crate::ribbon_panels::draw_side_groups(
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
                    active_tab: ribbon.active,
                    plane_selection_active: owners.plane_selection_active,
                    hole_tip: ribbon.tips.model_commands() && ribbon.tips.matching("HO"),
                    modify_group_tip: ribbon.tips.model_commands() && ribbon.tips.matching("Y"),
                    sketch_group_tip: ribbon.tips.model_commands() && ribbon.tips.matching("K"),
                    create_group_tip: ribbon.tips.model_commands() && ribbon.tips.matching("C"),
                    key_tips_active: ribbon.tips.active(),
                    sketch_tip: ribbon.tips.model_commands() && ribbon.tips.matching("SK"),
                    extrude_tip: ribbon.tips.model_commands() && ribbon.tips.matching("EX"),
                },
                &mut ribbon.panels,
                &mut ribbon.display,
                panel_x,
                actions,
            );
        });
}
