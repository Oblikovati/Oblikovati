// SPDX-License-Identifier: GPL-2.0-only
//! Per-owner panel measurements keep the Go ribbon's width-dependent collapse behavior.
use crate::application::ribbon_display::{RibbonDisplay, RibbonDisplayState, TemporaryPanel};
use crate::ribbon::RibbonTab;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use std::collections::BTreeMap;
#[path = "ribbon_panel_additional.rs"]
mod additional;
#[path = "ribbon_builtin.rs"]
pub(crate) mod builtin;

#[path = "ribbon_panel_catalog.rs"]
mod catalog;
pub(crate) use catalog::{RibbonPanel, available_panels, panels_for, panels_for_placement};
#[path = "ribbon_side_groups.rs"]
mod side_groups;
pub(crate) use side_groups::draw_side_groups;

#[derive(Default)]
pub(crate) struct RibbonPanelLayout {
    widths: BTreeMap<(RibbonTab, RibbonPanel), f32>,
    scrollbar_shown: bool,
    measured_appearance: Option<crate::ribbon_command_layout::RibbonAppearance>,
    popup_press: crate::popup_press::PopupPress<(RibbonTab, RibbonPanel)>,
    title_press: crate::popup_press::PopupPress<(RibbonTab, RibbonPanel)>,
    pub pending_layout: bool,
    pub context_title: Option<RibbonPanel>,
    pending_additional: Option<(RibbonTab, RibbonPanel)>,
    pending_model_pick: bool,
    retained_model_face: bool,
    additional: additional::PanelExpansions,
}
#[path = "ribbon_panel_keyboard.rs"]
mod keyboard;
#[path = "ribbon_panel_measurements.rs"]
mod measurements;
use measurements::{temporary_content_height, temporary_geometry};
/// Editors and layout a ribbon panel's controls can change.
pub(crate) struct RibbonPanelOwners<'a> {
    pub home_active: bool,
    pub icons: &'a crate::icons::GuiRibbonIcons,
    pub parameters: &'a mut crate::parameters::GuiParameterEditor,
    pub preferences: &'a mut crate::ui_scale::GuiScalePreferences,
    pub editor: &'a mut crate::theme_editor::GuiThemeEditor,
    pub workspace: &'a mut crate::application::workspace_layout::WorkspaceLayout,
    pub workspaces: &'a mut crate::application::document_creation::workspaces::DocumentWorkspaces,
    pub quick_access: &'a mut crate::application::quick_access_toolbar::QuickAccessToolbar,
    pub precise_input: &'a mut crate::sketch::precise_input::PreciseInputBar,
    pub sketch_pattern: &'a mut crate::sketch::rectangular_pattern::RectangularPatternDialog,
    pub create_block: &'a mut crate::sketch::create_block::CreateBlockDialog,
    pub customization: &'a mut crate::application::ribbon_customization::RibbonCustomization,
    pub team_web: &'a mut crate::ribbon::team_web::TeamWebWindow,
    pub active_tab: RibbonTab,
    pub plane_selection_active: bool,
    pub hole_tip: bool,
    pub modify_group_tip: bool,
    pub sketch_group_tip: bool,
    pub create_group_tip: bool,
    pub key_tips_active: bool,
    pub sketch_tip: bool,
    pub extrude_tip: bool,
    pub tools_application_tip: bool,
    pub tools_document_tip: bool,
}
struct PanelDrawContext<'a> {
    ui: &'a Ui,
    state: &'a GuiCommandState,
    hidden: std::collections::BTreeSet<(RibbonTab, RibbonPanel)>,
    show_panel_titles: bool,
    additional_allowed: bool,
    side_groups: bool,
    ribbon_bottom: f32,
    owners: RibbonPanelOwners<'a>,
    actions: &'a mut Vec<GuiShellAction>,
}
impl PanelDrawContext<'_> {
    fn content(&mut self, panel: RibbonPanel, origin: [f32; 2]) -> bool {
        let acted = crate::ribbon_panel_contents::draw_panel_content(
            self.ui,
            panel,
            origin,
            self.state,
            &mut self.owners,
            self.actions,
            self.additional_allowed,
        );
        if self.side_groups && panel == RibbonPanel::PartModify {
            crate::ribbon_panel_contents::compact_modify_commands(
                self.ui,
                origin,
                self.owners.icons,
            );
        }
        acted
    }
}
/// Draw one measured row; example: `draw_panels(ui, tab, state, owners, layout, actions)`.
pub(crate) fn draw_panels(
    ui: &Ui,
    tab: RibbonTab,
    state: &GuiCommandState,
    owners: RibbonPanelOwners<'_>,
    layout: &mut RibbonPanelLayout,
    display: &mut RibbonDisplayState,
    actions: &mut Vec<GuiShellAction>,
) {
    layout.pending_layout = false;
    layout.context_title = None;
    layout.use_appearance(display.appearance);
    layout.additional.use_document(state.document);
    layout.additional.use_mode(display.mode);
    layout
        .additional
        .use_title_visibility(display.show_panel_titles);
    if display.take_docking_selection() {
        layout.additional.clear();
    }
    if owners.home_active {
        layout.additional.clear();
    }
    let mut context = PanelDrawContext {
        ui,
        state,
        hidden: display.hidden_panels.clone(),
        show_panel_titles: display.show_panel_titles,
        additional_allowed: !owners.home_active
            && display.placement == crate::application::ribbon_display::RibbonPlacement::Top,
        side_groups: false,
        ribbon_bottom: ui.window_pos()[1] + ui.window_size()[1],
        owners,
        actions,
    };
    if display.mode == RibbonDisplay::Full {
        full_row(&mut context, tab, layout);
        layout.pending_additional = None;
        additional::draw(&mut context, tab, layout, display);
    } else {
        compact_row(&mut context, tab, display);
        temporary_panel(&mut context, tab, layout, display);
        if display.temporary.is_none() {
            layout.additional.clear();
        }
    }
}
fn full_row(context: &mut PanelDrawContext<'_>, tab: RibbonTab, layout: &mut RibbonPanelLayout) {
    let ui = context.ui;
    let available = ui.content_region_avail()[0].max(1.);
    let height = (20. + context.owners.icons.appearance.command_height())
        * context.owners.icons.layout_factor()
        + layout.scrollbar_height(ui);
    let region = crate::widget_abi::scroll_region(ui, "##ribbon-panels", [available, height]);
    if region.visible {
        draw_row(context, tab, layout, available);
        layout.observe_scrollbar(ui.scroll_max_x() > 0.);
    }
}
fn group_key_tip(ui: &Ui, panel: RibbonPanel, owners: &RibbonPanelOwners<'_>) {
    let tip = match panel {
        RibbonPanel::PartModify if owners.modify_group_tip => "Y",
        RibbonPanel::Sketch if owners.sketch_group_tip => "K",
        RibbonPanel::PartCreate if owners.create_group_tip => "C",
        RibbonPanel::Create if owners.create_group_tip => "D",
        _ => return,
    };
    crate::ribbon_key_tips::badge(ui, tip);
}
/// One button per group; a group's commands open in the temporary panel.
fn compact_row(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    display: &mut RibbonDisplayState,
) {
    let Some(size) = display.mode.group_button() else {
        return;
    };
    let ui = context.ui;
    for panel in shown_panels(
        tab,
        context.state,
        &context.hidden,
        context.owners.customization,
    ) {
        let open = display.temporary == Some(TemporaryPanel::Group(panel));
        let lit = ui.style_color(imgui::StyleColor::ButtonActive);
        let _lit = open.then(|| ui.push_style_color(imgui::StyleColor::Button, lit));
        let clicked = crate::widget_abi::button(ui, panel.label(), size);
        group_key_tip(ui, panel, &context.owners);
        if clicked || display.take_keyboard_group(panel) {
            display.open_temporary(TemporaryPanel::Group(panel));
            display.temporary_anchor = ui.item_rect_min()[0];
        }
        ui.same_line();
    }
}
/// Commands a compact mode hides, over the workspace until the next click elsewhere.
fn temporary_panel(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    layout: &mut RibbonPanelLayout,
    display: &mut RibbonDisplayState,
) {
    let Some(shown) = display.temporary else {
        return;
    };
    let ui = context.ui;
    let top = ui.window_size()[1];
    let (position, size) = temporary_geometry(
        ui,
        tab,
        layout,
        display,
        shown,
        top,
        context.owners.icons.layout_factor(),
    );
    context.ribbon_bottom = position[1] + size[1];
    crate::widget_abi::window_geometry(ui, position, size, imgui::Condition::Always);
    let before = context.actions.len();
    let mut keep = true;
    let focus = std::mem::take(&mut display.temporary_focus);
    let keep_on_click = std::mem::take(&mut display.keep_temporary_on_click)
        || layout.additional.preserves_tab_press();
    let selector_hovered = display.selector_open && display.selector_pointer_inside;
    ui.window("##ribbon-temporary-panel")
        // `flags` replaces the flags set so far, so it comes first.
        .flags(imgui::WindowFlags::NO_DOCKING)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .collapsible(false)
        .focused(focus)
        .build(|| {
            let local = temporary_contents(context, tab, layout, shown, display);
            keep = !local
                && (ui.is_window_hovered_with_flags(
                    imgui::WindowHoveredFlags::CHILD_WINDOWS
                        | imgui::WindowHoveredFlags::ALLOW_WHEN_BLOCKED_BY_ACTIVE_ITEM,
                ) || !ui.is_mouse_clicked(imgui::MouseButton::Left)
                    || keep_on_click
                    || selector_hovered
                    || layout.additional.contains(ui.io().mouse_pos));
        });
    if context.additional_allowed && additional::draw(context, tab, layout, display) {
        keep = false;
    }
    let escape = display.selector_escape
        || !context.owners.key_tips_active
            && !crate::widget_abi::any_popup_open(ui)
            && ui.is_key_pressed_no_repeat(imgui::Key::Escape);
    if escape {
        display.temporary_escape_consumed = true;
    }
    // Top-docked panels stay open; side-docked groups close on Escape (#3856).
    let close_group =
        escape && display.side_anchor.is_some() && matches!(shown, TemporaryPanel::Group(_));
    if display.selector_escape || close_group || !keep || context.actions.len() != before {
        display.temporary = None;
    }
}
fn temporary_contents(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    layout: &mut RibbonPanelLayout,
    shown: TemporaryPanel,
    display: &mut RibbonDisplayState,
) -> bool {
    let ui = context.ui;
    let available = ui.content_region_avail()[0].max(1.);
    let floating_notes = display.placement
        == crate::application::ribbon_display::RibbonPlacement::Floating
        && shown == TemporaryPanel::Group(RibbonPanel::PartNotes);
    let content_height = temporary_content_height(context.owners.icons.appearance, floating_notes);
    let height = 6. + content_height * context.owners.icons.layout_factor();
    let region = crate::widget_abi::scroll_region(ui, "##temporary-contents", [available, height]);
    if !region.visible {
        return false;
    }
    if floating_notes {
        crate::ribbon_panel_contents::floating_notes(ui, context.owners.icons);
        return false;
    }
    match shown {
        TemporaryPanel::Group(panel) => {
            let combines_options = panel == RibbonPanel::ToolOptions && context.additional_allowed;
            let compact_options = combines_options.then_some(display);
            let (width, local) =
                draw_expanded(context, tab, panel, [0.; 2], layout, compact_options);
            if !combines_options && (!context.side_groups || panel != RibbonPanel::PartModify) {
                layout.record_width(tab, panel, width);
            }
            local
        }
        TemporaryPanel::WholeTab => draw_row(context, tab, layout, available),
    }
}
/// The panels a user has not hidden, in the tab's order.
fn shown_panels(
    tab: RibbonTab,
    state: &GuiCommandState,
    hidden: &std::collections::BTreeSet<(RibbonTab, RibbonPanel)>,
    customization: &crate::application::ribbon_customization::RibbonCustomization,
) -> Vec<RibbonPanel> {
    let mut panels = available_panels(tab, state, customization);
    panels.retain(|panel| !hidden.contains(&(tab, *panel)));
    panels
}
fn draw_row(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    layout: &mut RibbonPanelLayout,
    available: f32,
) -> bool {
    let panels = shown_panels(
        tab,
        context.state,
        &context.hidden,
        context.owners.customization,
    );
    let collapsed = panels
        .iter()
        .map(|panel| (context.ui.calc_text_size(panel.label())[0] + 16.).max(108.))
        .fold(0., f32::max);
    let expanded = layout.expanded_count(tab, &panels, collapsed, available);
    let mut x = 0.;
    let mut activated = false;
    for (index, panel) in panels.into_iter().enumerate() {
        let force_compact = context.owners.icons.appearance
            == crate::ribbon_command_layout::RibbonAppearance::Compact
            && panel == RibbonPanel::PartSimplification;
        // Keep Finish directly available when other groups collapse (#3956).
        let width = if (index < expanded || panel == RibbonPanel::Exit) && !force_compact {
            let (width, local_action) = draw_expanded(context, tab, panel, [x, 0.], layout, None);
            activated |= local_action;
            layout.record_width(tab, panel, width);
            width
        } else {
            draw_collapsed(context, tab, panel, [x, 0.], collapsed, layout);
            collapsed
        };
        x += width + 8.;
    }
    layout.popup_press.finish_frame(context.ui);
    layout.title_press.finish_frame(context.ui);
    activated
}
fn draw_expanded(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    panel: RibbonPanel,
    origin: [f32; 2],
    layout: &mut RibbonPanelLayout,
    compact_options: Option<&mut RibbonDisplayState>,
) -> (f32, bool) {
    let ui = context.ui;
    crate::widget_abi::cursor_position(ui, origin);
    let anchor = ui.cursor_screen_pos();
    let mut local_action = false;
    ui.group(|| {
        local_action = context.content(panel, origin);
        let combines_options = compact_options.is_some();
        if let Some(display) = compact_options {
            context.owners.icons.cursor(ui, origin, [270., 0.]);
            local_action |= additional::tool_options(ui, display);
        }
        let title_y = context.owners.icons.appearance.command_height() + 6.;
        context.owners.icons.cursor(ui, origin, [0., title_y]);
        let expandable = !combines_options
            && context.additional_allowed
            && (additional::available(tab, panel)
                || builtin::is_expanded(panel, context.state, context.owners.customization)
                || (panel == RibbonPanel::UserCommands
                    && context.owners.customization.has_expanded(tab)));
        if context.show_panel_titles {
            ui.text(panel.label());
            if ui.is_item_clicked_with_button(imgui::MouseButton::Right) {
                layout.context_title = Some(panel);
            }
            let title_released = expandable && layout.title_press.released(ui, (tab, panel));
            let title_clicked = ui.is_item_clicked() || title_released;
            local_action |= title_clicked && !expandable;
            if expandable {
                let requested = layout.take_additional(tab, panel);
                additional::trigger(ui, tab, panel, anchor, title_clicked || requested, layout);
                if matches!(panel, RibbonPanel::PartModify | RibbonPanel::Create) {
                    group_key_tip(ui, panel, &context.owners);
                }
            }
        } else if expandable {
            crate::widget_abi::dummy(ui, [0., ui.text_line_height()]);
            additional::trigger(ui, tab, panel, anchor, false, layout);
        }
        if panel == RibbonPanel::Create
            && context.owners.icons.appearance
                == crate::ribbon_command_layout::RibbonAppearance::Normal
        {
            // Preserve the existing Create grid's trailing space and wide-window command positions.
            crate::widget_abi::cursor_position(ui, origin);
            crate::widget_abi::dummy(ui, context.owners.icons.cell([364., 0.]));
        }
    });
    (ui.item_rect_size()[0], local_action)
}
fn draw_collapsed(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    panel: RibbonPanel,
    origin: [f32; 2],
    width: f32,
    layout: &mut RibbonPanelLayout,
) {
    let ui = context.ui;
    let identity = format!("##panel-{tab:?}-{panel:?}");
    crate::widget_abi::cursor_position(ui, origin);
    let clicked = crate::widget_abi::button(
        ui,
        &format!("v{identity}"),
        [
            width,
            (context.owners.icons.appearance.command_height() - 2.).max(18.)
                * context.owners.icons.layout_factor(),
        ],
    );
    let through_popup = layout.popup_press.released(ui, (tab, panel));
    if clicked || through_popup {
        ui.open_popup(&identity);
    }
    let anchor = ui.item_rect_min()[0];
    if ui.is_item_hovered() {
        ui.tooltip_text(format!("Show {} commands", panel.label()));
    }
    let title_y = context.owners.icons.appearance.command_height() + 6.;
    context.owners.icons.cursor(ui, origin, [0., title_y]);
    // Collapsed buttons need their label even when expanded panel captions are hidden.
    ui.text(panel.label());
    draw_panel_popup(context, tab, panel, &identity, anchor, layout);
}

#[path = "ribbon_panel_flyout.rs"]
mod flyout;
use flyout::draw_panel_popup;

#[cfg(test)]
#[path = "../tests/ribbon/panel_appearance.rs"]
mod appearance_tests;
