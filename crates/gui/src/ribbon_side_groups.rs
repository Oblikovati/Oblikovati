// SPDX-License-Identifier: GPL-2.0-only
//! The side and floating ribbon group column.
use super::*;
use crate::application::ribbon_display::{RibbonPlacement, SideAnchor};

/// A side or floating ribbon's groups: one stacked button each, opening beside the ribbon.
/// Example: `draw_side_groups(ui, tab, state, owners, layout, display, panel_x, actions)`.
// Each is a distinct owner of what a side ribbon draws or changes.
#[allow(clippy::too_many_arguments)]
pub(crate) fn draw_side_groups(
    ui: &Ui,
    tab: RibbonTab,
    state: &GuiCommandState,
    owners: RibbonPanelOwners<'_>,
    layout: &mut RibbonPanelLayout,
    display: &mut RibbonDisplayState,
    panel_x: (f32, bool),
    actions: &mut Vec<GuiShellAction>,
) {
    layout.pending_layout = false;
    layout.use_appearance(display.appearance);
    layout.additional.use_document(state.document);
    layout.additional.use_mode(display.mode);
    layout
        .additional
        .use_title_visibility(display.show_panel_titles);
    if display.take_docking_selection() || owners.home_active {
        layout.additional.clear();
    }
    let mut context = PanelDrawContext {
        ui,
        state,
        hidden: display.hidden_panels.clone(),
        show_panel_titles: display.show_panel_titles,
        additional_allowed: !owners.home_active,
        side_groups: true,
        options_menu_hovered: false,
        ribbon_bottom: ui.window_pos()[1] + ui.window_size()[1],
        owners,
        actions,
    };
    let panels = panels_for_placement(tab, state, context.owners.customization, display.placement);
    for panel in panels {
        if !display.hidden_panels.contains(&(tab, panel)) {
            draw_side_group(&mut context, tab, layout, display, panel_x, panel);
        }
    }
    let temporary_open = display.temporary.is_some();
    temporary_panel(&mut context, tab, layout, display);
    if !temporary_open && layout.additional.has_open() {
        context.ribbon_bottom = layout.additional.last_ribbon_bottom();
        additional::draw(&mut context, tab, layout, display);
    }
}
fn draw_side_group(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    layout: &mut RibbonPanelLayout,
    display: &mut RibbonDisplayState,
    panel_x: (f32, bool),
    panel: RibbonPanel,
) {
    let ui = context.ui;
    let floating = display.placement == RibbonPlacement::Floating;
    if panel == RibbonPanel::Exit {
        // Sketch editing needs a direct way out in the vertical ribbon too (#3856).
        draw_expanded(context, tab, panel, ui.cursor_pos(), layout, None);
        return;
    }
    let pressed = if floating && panel == RibbonPanel::Sketch && tab == RibbonTab::Model {
        context.content(panel, ui.cursor_pos());
        false
    } else {
        side_group_pressed(context, display, panel)
    };
    if pressed || display.take_keyboard_group(panel) {
        display.open_temporary(TemporaryPanel::Group(panel));
        display.side_anchor = Some(SideAnchor {
            x: panel_x.0,
            y: ui.item_rect_min()[1],
            ends_at_x: panel_x.1,
        });
    }
}

fn side_group_pressed(
    context: &PanelDrawContext<'_>,
    display: &RibbonDisplayState,
    panel: RibbonPanel,
) -> bool {
    let ui = context.ui;
    let open = display.temporary == Some(TemporaryPanel::Group(panel));
    let lit = ui.style_color(imgui::StyleColor::NavHighlight);
    let _lit = open.then(|| ui.push_style_color(imgui::StyleColor::Button, lit));
    let _hovered = open.then(|| ui.push_style_color(imgui::StyleColor::ButtonHovered, lit));
    let _disabled = ui.begin_disabled(group_is_unavailable(panel));
    let size = if display.placement == RibbonPlacement::Floating {
        [ui.content_region_avail()[0].max(112.), 42.]
    } else {
        [112., 48.]
    };
    let clicked = crate::widget_abi::button(ui, panel.label(), size);
    group_key_tip(ui, panel, &context.owners);
    clicked
}

fn group_is_unavailable(panel: RibbonPanel) -> bool {
    use RibbonPanel::*;
    matches!(
        panel,
        PartShapeGenerator
            | PartFreeform
            | PartGeometricAnnotation
            | PartGeneralAnnotation
            | PartAnnotationManage
            | PartToleranceAnalysis
    )
}
