// SPDX-License-Identifier: GPL-2.0-only
//! Per-owner panel measurements keep the Go ribbon's width-dependent collapse behavior.
use crate::application::ribbon_display::{RibbonDisplay, RibbonDisplayState, TemporaryPanel};
use crate::ribbon::RibbonTab;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use std::collections::BTreeMap;

#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub(crate) enum RibbonPanel {
    Launch,
    Sketch,
    Create,
    Exit,
    Constrain,
    Modify,
    Format,
    Parameters,
    Navigate,
    Appearance,
    Visibility,
    Windows,
    Component,
    SheetMetalTools,
    DrawingViews,
    DrawingModify,
    DrawingSketch,
    DrawingSheets,
    AnnotateDimension,
    AnnotateText,
    AnnotateSymbols,
    PartCreate,
    PartModify,
    PartWorkFeatures,
    PartPattern,
    PartNotes,
    PresentationModel,
}
impl RibbonPanel {
    pub(crate) fn label(self) -> &'static str {
        match self {
            Self::Launch => "Launch",
            Self::Sketch => "Sketch",
            Self::Create => "Create",
            Self::Exit => "Exit",
            Self::Constrain => "Constrain",
            Self::Modify => "Modify",
            Self::Format => "Format",
            Self::Parameters => "Parameters",
            Self::Navigate => "Navigate",
            Self::Appearance => "Appearance",
            Self::Visibility => "Visibility",
            Self::Windows => "Windows",
            Self::Component => "Component",
            Self::SheetMetalTools => "Sheet Metal",
            Self::DrawingViews | Self::PartCreate => "Create",
            Self::DrawingModify | Self::PartModify => "Modify",
            Self::PartWorkFeatures => "Work Features",
            Self::PartNotes => "Notes",
            Self::PartPattern => "Pattern",
            Self::DrawingSketch => "Sketch",
            Self::DrawingSheets => "Sheets",
            Self::AnnotateDimension => "Dimension",
            Self::AnnotateText => "Text",
            Self::AnnotateSymbols => "Symbols",
            Self::PresentationModel => "Workshop",
        }
    }
    pub(crate) fn command(self) -> Option<(&'static str, GuiShellAction)> {
        Some(match self {
            Self::Launch => ("New Part", GuiShellAction::NewPart),
            Self::Sketch => ("Sketch XY", GuiShellAction::StartSketchXY),
            Self::Exit => ("Finish Sketch", GuiShellAction::FinishSketch),
            Self::Modify => ("Delete", GuiShellAction::DeleteSketchSelection),
            Self::Navigate => ("Home", GuiShellAction::Home),
            Self::Create
            | Self::Constrain
            | Self::Format
            | Self::Parameters
            | Self::Appearance
            | Self::Visibility
            | Self::Windows
            | Self::Component
            | Self::SheetMetalTools
            | Self::DrawingViews
            | Self::DrawingModify
            | Self::DrawingSketch
            | Self::DrawingSheets
            | Self::AnnotateDimension
            | Self::AnnotateText
            | Self::AnnotateSymbols
            | Self::PartCreate
            | Self::PartModify
            | Self::PartWorkFeatures
            | Self::PartPattern
            | Self::PartNotes
            | Self::PresentationModel => {
                return None;
            }
        })
    }
}
#[derive(Default)]
pub(crate) struct RibbonPanelLayout {
    widths: BTreeMap<(RibbonTab, RibbonPanel), f32>,
    scrollbar_shown: bool,
    pub pending_layout: bool,
}
impl RibbonPanelLayout {
    pub(crate) fn scrollbar_height(&self, ui: &Ui) -> f32 {
        if self.scrollbar_shown {
            ui.clone_style().scrollbar_size
        } else {
            0.
        }
    }
    fn record_width(&mut self, tab: RibbonTab, panel: RibbonPanel, width: f32) {
        let old = self.widths.insert((tab, panel), width);
        self.pending_layout |= old != Some(width);
    }
    fn expanded_count(
        &self,
        tab: RibbonTab,
        panels: &[RibbonPanel],
        collapsed: f32,
        available: f32,
    ) -> usize {
        let widths: Vec<f32> = panels
            .iter()
            .map(|panel| self.widths.get(&(tab, *panel)).copied().unwrap_or(0.))
            .collect();
        crate::ribbon_fit::expanded_panel_count(&widths, 8., collapsed, available)
    }
    fn observe_scrollbar(&mut self, shown: bool) {
        self.pending_layout |= self.scrollbar_shown != shown;
        self.scrollbar_shown = shown;
    }
}
/// Editors and layout a ribbon panel's controls can change.
pub(crate) struct RibbonPanelOwners<'a> {
    pub parameters: &'a mut crate::parameters::GuiParameterEditor,
    pub editor: &'a mut crate::theme_editor::GuiThemeEditor,
    pub workspace: &'a mut crate::application::workspace_layout::WorkspaceLayout,
    pub workspaces: &'a mut crate::application::document_creation::workspaces::DocumentWorkspaces,
}
struct PanelDrawContext<'a> {
    ui: &'a Ui,
    state: &'a GuiCommandState,
    hidden: std::collections::BTreeSet<(RibbonTab, RibbonPanel)>,
    owners: RibbonPanelOwners<'a>,
    actions: &'a mut Vec<GuiShellAction>,
}
impl PanelDrawContext<'_> {
    fn content(&mut self, panel: RibbonPanel, origin: [f32; 2]) -> bool {
        crate::ribbon_panel_contents::draw_panel_content(
            self.ui,
            panel,
            origin,
            self.state,
            &mut self.owners,
            self.actions,
        )
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
    let mut context = PanelDrawContext {
        ui,
        state,
        hidden: display.hidden_panels.clone(),
        owners,
        actions,
    };
    if display.mode == RibbonDisplay::Full {
        full_row(&mut context, tab, layout);
    } else {
        compact_row(&mut context, tab, display);
        temporary_panel(&mut context, tab, layout, display);
    }
}
fn full_row(context: &mut PanelDrawContext<'_>, tab: RibbonTab, layout: &mut RibbonPanelLayout) {
    let ui = context.ui;
    let available = ui.content_region_avail()[0].max(1.);
    let height = 80. + layout.scrollbar_height(ui);
    let region = crate::widget_abi::scroll_region(ui, "##ribbon-panels", [available, height]);
    if region.visible {
        draw_row(context, tab, layout, available);
        layout.observe_scrollbar(ui.scroll_max_x() > 0.);
    }
}
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
    let mut context = PanelDrawContext {
        ui,
        state,
        hidden: display.hidden_panels.clone(),
        owners,
        actions,
    };
    for panel in shown_panels(tab, state, &display.hidden_panels) {
        if crate::widget_abi::button(ui, panel.label(), SIDE_GROUP_BUTTON) {
            display.temporary = Some(TemporaryPanel::Group(panel));
            display.side_anchor = Some(crate::application::ribbon_display::SideAnchor {
                x: panel_x.0,
                y: ui.item_rect_min()[1],
                ends_at_x: panel_x.1,
            });
        }
    }
    temporary_panel(&mut context, tab, layout, display);
}
const SIDE_GROUP_BUTTON: [f32; 2] = [112., 48.];
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
    for panel in shown_panels(tab, context.state, &context.hidden) {
        if crate::widget_abi::button(ui, panel.label(), size) {
            display.temporary = Some(TemporaryPanel::Group(panel));
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
    let (position, size) = temporary_geometry(ui, tab, layout, display, shown, top);
    crate::widget_abi::window_geometry(ui, position, size, imgui::Condition::Always);
    let before = context.actions.len();
    let mut keep = true;
    ui.window("##ribbon-temporary-panel")
        // `flags` replaces the flags set so far, so it comes first.
        .flags(imgui::WindowFlags::NO_DOCKING)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .collapsible(false)
        .build(|| {
            let local = temporary_contents(context, tab, layout, shown);
            keep = !local
                && (ui.is_window_hovered() || !ui.is_mouse_clicked(imgui::MouseButton::Left));
        });
    if !keep || context.actions.len() != before {
        display.temporary = None;
    }
}
/// A group's panel opens under its button at its measured width; a whole tab spans the window.
fn temporary_geometry(
    ui: &Ui,
    tab: RibbonTab,
    layout: &RibbonPanelLayout,
    display: &RibbonDisplayState,
    shown: TemporaryPanel,
    top: f32,
) -> ([f32; 2], [f32; 2]) {
    let viewport = ui.io().display_size[0];
    let TemporaryPanel::Group(panel) = shown else {
        return ([8., top], [viewport - 16., 104.]);
    };
    let measured = layout.widths.get(&(tab, panel)).copied().unwrap_or(108.);
    let width = (measured + 32.).max(140.).min(viewport - 16.);
    if let Some(anchor) = display.side_anchor {
        let x = if anchor.ends_at_x {
            anchor.x - width
        } else {
            anchor.x
        };
        return ([x, anchor.y], [width, 104.]);
    }
    let x = display.temporary_anchor.min(viewport - width - 8.).max(8.);
    ([x, top], [width, 104.])
}
fn temporary_contents(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    layout: &mut RibbonPanelLayout,
    shown: TemporaryPanel,
) -> bool {
    let ui = context.ui;
    let available = ui.content_region_avail()[0].max(1.);
    let region = crate::widget_abi::scroll_region(ui, "##temporary-contents", [available, 86.]);
    if !region.visible {
        return false;
    }
    match shown {
        TemporaryPanel::Group(panel) => {
            let (width, local) = draw_expanded(context, panel, [0.; 2]);
            layout.record_width(tab, panel, width);
            local
        }
        TemporaryPanel::WholeTab => {
            draw_row(context, tab, layout, available);
            false
        }
    }
}
/// The panels a user has not hidden, in the tab's order.
fn shown_panels(
    tab: RibbonTab,
    state: &GuiCommandState,
    hidden: &std::collections::BTreeSet<(RibbonTab, RibbonPanel)>,
) -> Vec<RibbonPanel> {
    let mut panels = panels_for(tab, state);
    panels.retain(|panel| !hidden.contains(&(tab, *panel)));
    panels
}
/// Every panel a tab offers, hidden ones included.
pub(crate) fn panels_for(tab: RibbonTab, state: &GuiCommandState) -> Vec<RibbonPanel> {
    use RibbonPanel::*;
    match tab {
        RibbonTab::GetStarted => vec![Launch],
        RibbonTab::Model => vec![
            Sketch,
            PartCreate,
            PartModify,
            PartWorkFeatures,
            PartPattern,
            Parameters,
        ],
        RibbonTab::SheetMetal => vec![SheetMetalTools, Sketch],
        RibbonTab::Assemble => vec![Component],
        RibbonTab::PlaceViews => vec![DrawingViews, DrawingModify, DrawingSketch, DrawingSheets],
        RibbonTab::Annotate
            if state.document_type == Some(oblikovati_api::types::DOCUMENT_PART) =>
        {
            vec![PartNotes]
        }
        RibbonTab::Annotate => vec![AnnotateDimension, AnnotateText, AnnotateSymbols],
        RibbonTab::Presentation => vec![PresentationModel],
        RibbonTab::Sketch => vec![Create, Exit, Constrain, Modify, Format],
        RibbonTab::Manage => vec![Parameters],
        RibbonTab::View if state.theme.is_some() || state.color_scheme.is_some() => {
            vec![Visibility, Appearance, Windows, Navigate]
        }
        RibbonTab::View => vec![Visibility, Windows, Navigate],
    }
}
fn draw_row(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    layout: &mut RibbonPanelLayout,
    available: f32,
) {
    let panels = shown_panels(tab, context.state, &context.hidden);
    let collapsed = panels
        .iter()
        .map(|panel| (context.ui.calc_text_size(panel.label())[0] + 16.).max(108.))
        .fold(0., f32::max);
    let expanded = layout.expanded_count(tab, &panels, collapsed, available);
    let mut x = 0.;
    for (index, panel) in panels.into_iter().enumerate() {
        let width = if index < expanded {
            let (width, _) = draw_expanded(context, panel, [x, 0.]);
            layout.record_width(tab, panel, width);
            width
        } else {
            draw_collapsed(context, tab, panel, [x, 0.], collapsed, layout);
            collapsed
        };
        x += width + 8.;
    }
}
fn draw_expanded(
    context: &mut PanelDrawContext<'_>,
    panel: RibbonPanel,
    origin: [f32; 2],
) -> (f32, bool) {
    let ui = context.ui;
    crate::widget_abi::cursor_position(ui, origin);
    let mut local_action = false;
    ui.group(|| {
        local_action = context.content(panel, origin);
        crate::widget_abi::cursor_position(ui, [origin[0], origin[1] + 66.]);
        ui.text(panel.label());
        if panel == RibbonPanel::Create {
            // Preserve the existing Create grid's trailing space and wide-window command positions.
            crate::widget_abi::cursor_position(ui, origin);
            crate::widget_abi::dummy(ui, [364., 0.]);
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
    if crate::widget_abi::button(ui, &format!("v{identity}"), [width, 58.]) {
        ui.open_popup(&identity);
    }
    let anchor = ui.item_rect_min()[0];
    if ui.is_item_hovered() {
        ui.tooltip_text(format!("Show {} commands", panel.label()));
    }
    crate::widget_abi::cursor_position(ui, [origin[0], origin[1] + 66.]);
    ui.text(panel.label());
    draw_panel_popup(context, tab, panel, &identity, anchor, layout);
}
fn draw_panel_popup(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    panel: RibbonPanel,
    identity: &str,
    anchor: f32,
    layout: &mut RibbonPanelLayout,
) {
    let ui = context.ui;
    let width = layout.widths.get(&(tab, panel)).copied().unwrap_or(108.);
    let size = popup_geometry(ui, anchor, width, 158. + layout.scrollbar_height(ui));
    let Some(_popup) = ui.begin_popup(identity) else {
        return;
    };
    let region = crate::widget_abi::scroll_region(ui, "##panel-contents", size);
    if !region.visible {
        return;
    }
    let before = context.actions.len();
    let (measured, local_action) = draw_expanded(context, panel, [0.; 2]);
    layout.record_width(tab, panel, measured);
    if context.actions.len() != before || local_action {
        ui.close_current_popup();
    }
}
fn popup_geometry(ui: &Ui, anchor: f32, width: f32, band_height: f32) -> [f32; 2] {
    let viewport = ui.io().display_size[0];
    let content_width = width.min((viewport - 32.).max(1.));
    let padding = ui.clone_style().window_padding;
    let outer_width = content_width + 2. * padding[0];
    let height = 80.
        + if content_width < width {
            ui.clone_style().scrollbar_size
        } else {
            0.
        };
    let x = crate::ribbon_fit::clamp_panel_flyout_x(anchor, outer_width, viewport);
    crate::widget_abi::window_geometry(
        ui,
        [x, band_height],
        [outer_width, height + 2. * padding[1]],
        imgui::Condition::Always,
    );
    [content_width, height]
}
