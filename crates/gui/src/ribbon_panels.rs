// SPDX-License-Identifier: GPL-2.0-only
//! Per-owner panel measurements keep the Go ribbon's width-dependent collapse behavior.
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
        }
    }
    pub(crate) fn command(self) -> Option<(&'static str, GuiShellAction)> {
        Some(match self {
            Self::Launch => ("New Part", GuiShellAction::NewPart),
            Self::Sketch => ("Sketch XY", GuiShellAction::StartSketchXY),
            Self::Exit => ("Finish Sketch", GuiShellAction::FinishSketch),
            Self::Modify => ("Delete", GuiShellAction::DeleteSketchSelection),
            Self::Navigate => ("Home", GuiShellAction::Home),
            Self::Create | Self::Constrain | Self::Format | Self::Parameters | Self::Appearance => {
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
struct PanelDrawContext<'a> {
    ui: &'a Ui,
    state: &'a GuiCommandState,
    parameters: &'a mut crate::parameters::GuiParameterEditor,
    editor: &'a mut crate::theme_editor::GuiThemeEditor,
    actions: &'a mut Vec<GuiShellAction>,
}
impl PanelDrawContext<'_> {
    fn content(&mut self, panel: RibbonPanel, origin: [f32; 2]) -> bool {
        crate::ribbon_panel_contents::draw_panel_content(
            self.ui,
            panel,
            origin,
            self.state,
            self.parameters,
            self.editor,
            self.actions,
        )
    }
}
/// Draw one measured row; example: `draw_panels(ui, tab, state, parameters, editor, layout, actions)`.
pub(crate) fn draw_panels(
    ui: &Ui,
    tab: RibbonTab,
    state: &GuiCommandState,
    parameters: &mut crate::parameters::GuiParameterEditor,
    editor: &mut crate::theme_editor::GuiThemeEditor,
    layout: &mut RibbonPanelLayout,
    actions: &mut Vec<GuiShellAction>,
) {
    let available = ui.content_region_avail()[0].max(1.);
    let height = 80. + layout.scrollbar_height(ui);
    layout.pending_layout = false;
    let region = crate::widget_abi::scroll_region(ui, "##ribbon-panels", [available, height]);
    if region.visible {
        let mut context = PanelDrawContext {
            ui,
            state,
            parameters,
            editor,
            actions,
        };
        draw_row(&mut context, tab, layout, available);
        layout.observe_scrollbar(ui.scroll_max_x() > 0.);
    }
}
fn panels_for(tab: RibbonTab, state: &GuiCommandState) -> Vec<RibbonPanel> {
    use RibbonPanel::*;
    match tab {
        RibbonTab::GetStarted => vec![Launch],
        RibbonTab::CreateModify => vec![Sketch],
        RibbonTab::Sketch => vec![Create, Exit, Constrain, Modify, Format],
        RibbonTab::Manage => vec![Parameters],
        RibbonTab::View if state.theme.is_some() || state.color_scheme.is_some() => {
            vec![Navigate, Appearance]
        }
        RibbonTab::View => vec![Navigate],
    }
}
fn draw_row(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    layout: &mut RibbonPanelLayout,
    available: f32,
) {
    let panels = panels_for(tab, context.state);
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
