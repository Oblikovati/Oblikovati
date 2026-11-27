// SPDX-License-Identifier: GPL-2.0-only
//! Additional ribbon commands stay anchored to their panel and can be pinned.
use super::*;
#[path = "ribbon_panel_additional/bend_notice.rs"]
mod bend_notice;
#[path = "ribbon_panel_additional/commands.rs"]
mod commands;
pub(super) use commands::tool_options;
#[path = "ribbon_panel_additional/hidden_strip.rs"]
mod hidden_strip;
#[path = "ribbon_panel_additional/temporary_tab.rs"]
mod temporary_tab;
const ADDITIONAL_WIDTH: f32 = 230.;
const ADDITIONAL_HEIGHT: f32 = 130.;
const VIEWPORT_MARGIN: f32 = 8.;

#[derive(Default)]
pub(super) struct PanelExpansions {
    open: BTreeMap<(RibbonTab, RibbonPanel), Expansion>,
    document: Option<u64>,
    mode: Option<RibbonDisplay>,
    titles_visible: Option<bool>,
    last_ribbon_bottom: f32,
    temporary_tab_armed: bool,
    preserve_tab_press: bool,
}
#[derive(Clone, Copy)]
struct Expansion {
    anchor: [f32; 2],
    trigger: [[f32; 2]; 2],
    pinned: bool,
    focus: bool,
    bounds: [[f32; 2]; 2],
    bend_notice: bend_notice::BendNotice,
}
impl PanelExpansions {
    pub(super) fn clear(&mut self) {
        self.open.clear();
        self.temporary_tab_armed = false;
        self.preserve_tab_press = false;
    }
    pub(super) fn use_document(&mut self, document: Option<u64>) {
        if self.document != document {
            self.clear();
            self.document = document;
        }
    }
    pub(super) fn use_mode(&mut self, mode: RibbonDisplay) {
        if self.mode != Some(mode) {
            self.clear();
            self.mode = Some(mode);
        }
    }
    pub(super) fn use_title_visibility(&mut self, visible: bool) {
        if self
            .titles_visible
            .is_some_and(|previous| previous != visible)
        {
            self.open.retain(|_, expansion| expansion.pinned);
        }
        self.titles_visible = Some(visible);
    }
    pub(super) fn close_one(&mut self, tab: RibbonTab, panel: RibbonPanel) {
        self.open.remove(&(tab, panel));
    }
    pub(super) fn close_unpinned(&mut self) {
        self.open.retain(|_, item| item.pinned);
    }
    pub(super) fn contains(&self, point: [f32; 2]) -> bool {
        self.open.values().any(|entry| inside(point, entry.bounds))
    }
    pub(super) fn has_open(&self) -> bool {
        !self.open.is_empty()
    }
    pub(super) fn last_ribbon_bottom(&self) -> f32 {
        self.last_ribbon_bottom
    }
}
pub(super) fn available(tab: RibbonTab, panel: RibbonPanel) -> bool {
    matches!(
        (tab, panel),
        (
            RibbonTab::Model,
            RibbonPanel::PartModify | RibbonPanel::PartSimplification
        ) | (RibbonTab::Tools, RibbonPanel::ToolOptions)
            | (
                RibbonTab::Sketch,
                RibbonPanel::Constrain | RibbonPanel::Create
            )
    )
}
pub(super) fn trigger(
    ui: &Ui,
    tab: RibbonTab,
    panel: RibbonPanel,
    anchor: [f32; 2],
    title_clicked: bool,
    layout: &mut RibbonPanelLayout,
) {
    let title_bounds = [ui.item_rect_min(), ui.item_rect_max()];
    ui.same_line_with_spacing(0., 4.);
    let _panel_id = ui.push_id(format!("{panel:?}"));
    let _bg = ui.push_style_color(imgui::StyleColor::Button, [0.; 4]);
    let _border = ui.push_style_var(imgui::StyleVar::FrameBorderSize(0.));
    let _ink = ui.push_style_color(imgui::StyleColor::Text, crate::theme::text_muted(ui));
    let clicked = ui.small_button("▾##additional-commands");
    let arrow_bounds = [ui.item_rect_min(), ui.item_rect_max()];
    let trigger = [
        [
            title_bounds[0][0],
            title_bounds[0][1].min(arrow_bounds[0][1]),
        ],
        [
            arrow_bounds[1][0],
            title_bounds[1][1].max(arrow_bounds[1][1]),
        ],
    ];
    let key = (tab, panel);
    if (clicked || title_clicked) && layout.additional.open.remove(&key).is_none() {
        layout.additional.open.retain(|_, item| item.pinned);
        layout.additional.temporary_tab_armed = false;
        layout.additional.open.insert(
            key,
            Expansion {
                anchor,
                trigger,
                pinned: false,
                focus: true,
                bounds: [[0.; 2]; 2],
                bend_notice: Default::default(),
            },
        );
    }
    if let Some(expansion) = layout.additional.open.get_mut(&key) {
        expansion.anchor = anchor;
        expansion.trigger = trigger;
    }
    if ui.is_item_hovered() && !layout.additional.open.contains_key(&key) {
        ui.tooltip_text("Additional commands");
    }
}
pub(super) fn draw(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    layout: &mut RibbonPanelLayout,
    display: &mut RibbonDisplayState,
) -> bool {
    let mut closed = false;
    layout.additional.last_ribbon_bottom = context.ribbon_bottom;
    layout.additional.open.retain(|(owner, panel), expansion| {
        *owner == tab
            && context.additional_allowed
            && (!display.hidden_panels.contains(&(tab, *panel))
                || hidden_strip::retains(*panel, expansion, display))
    });
    let preserve_tab_press =
        std::mem::take(&mut layout.additional.preserve_tab_press) || layout.pending_model_pick;
    let keys: Vec<_> = layout.additional.open.keys().copied().collect();
    for key in keys {
        let Some(expansion) = layout.additional.open.get_mut(&key) else {
            continue;
        };
        if display.hidden_panels.contains(&key) {
            hidden_strip::draw(context, key.1, expansion);
            continue;
        }
        let area = context
            .owners
            .workspace
            .document_area(context.ui.io().display_size, context.ribbon_bottom);
        if draw_window(
            context,
            key.0,
            key.1,
            area,
            expansion,
            display,
            preserve_tab_press,
        ) {
            layout.additional.open.remove(&key);
            closed = true;
        }
    }
    closed
}
fn draw_window(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    panel: RibbonPanel,
    area: crate::application::document_creation::workspaces::WorkspaceArea,
    expansion: &mut Expansion,
    display: &mut RibbonDisplayState,
    preserve_tab_press: bool,
) -> bool {
    let ui = context.ui;
    let scale = context.owners.icons.layout_factor();
    let [viewport_width, viewport_height] = ui.io().display_size;
    // Additional commands may cover the browser as well as the viewport (#3856).
    let left = 0.;
    let right = viewport_width.max(1.);
    let appearance = context.owners.icons.appearance;
    let command_rows = appearance == crate::ribbon_command_layout::RibbonAppearance::Normal
        || panel == RibbonPanel::ToolOptions;
    let preferred_width = if command_rows {
        ADDITIONAL_WIDTH
    } else {
        let count = if panel == RibbonPanel::UserCommands {
            context.owners.customization.expanded_command_count(tab)
        } else {
            3
        };
        let (position, size) = appearance.cell(count.saturating_sub(1));
        (position[0] + size[0] + 28.).max(88.)
    };
    let width = (preferred_width * scale).min((right - left - VIEWPORT_MARGIN).max(1.));
    let anchor_top =
        if display.placement == crate::application::ribbon_display::RibbonPlacement::Top {
            area.top.max(context.ribbon_bottom)
        } else {
            area.top
        };
    let top = anchor_top.min((viewport_height - 1.).max(0.));
    let preferred_height = if command_rows {
        commands::normal_expansion_height(ui, panel, context.owners.customization)
            .unwrap_or(ADDITIONAL_HEIGHT)
    } else {
        appearance.command_height() + 50.
    };
    let height = (preferred_height * scale).min((viewport_height - top).max(1.));
    let x = expansion.anchor[0].clamp(left, (right - width).max(left));
    expansion.bounds = [[x, top], [x + width, top + height]];
    crate::widget_abi::window_geometry(ui, [x, top], [width, height], imgui::Condition::Always);
    let mut close = false;
    let focus = std::mem::take(&mut expansion.focus);
    expansion_window(ui, panel)
        .flags(
            imgui::WindowFlags::NO_DOCKING
                | imgui::WindowFlags::NO_SAVED_SETTINGS
                | imgui::WindowFlags::HORIZONTAL_SCROLLBAR,
        )
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .collapsible(false)
        .focused(focus)
        .build(|| {
            close |= expansion.bend_notice.draw(ui);
            if panel == RibbonPanel::UserCommands {
                let right = context.owners.customization.draw_panel(
                    ui,
                    tab,
                    Some(true),
                    context.state,
                    context.owners.icons,
                    context.actions,
                );
                context.owners.quick_access.command_took_right_click |= right;
                close |= !context.owners.customization.has_expanded(tab);
            } else if panel == RibbonPanel::ToolOptions {
                close |= commands::options_flyout(ui, display, context.owners.icons);
            } else {
                let before = context.actions.len();
                commands::draw(context, tab, panel, display, &mut expansion.bend_notice);
                if !expansion.pinned
                    && context.actions[before..].contains(&GuiShellAction::OpenMoveBody)
                {
                    close = true;
                }
                if builtin::supports(panel) {
                    close |=
                        !builtin::is_expanded(panel, context.state, context.owners.customization);
                }
            }
            ui.separator();
            if shows_pin(display) {
                let active = expansion.pinned.then(|| {
                    ui.push_style_color(
                        imgui::StyleColor::Button,
                        ui.style_color(imgui::StyleColor::ButtonActive),
                    )
                });
                let clicked = ui.small_button("   Pin##additional-pin");
                draw_pin(ui);
                if clicked {
                    expansion.pinned = !expansion.pinned;
                }
                drop(active);
                ui.same_line();
            }
            let title_clicked = ui.small_button(panel.label());
            if ui.is_item_hovered() && ui.is_mouse_clicked(imgui::MouseButton::Right) {
                display.request_expansion_context(tab, panel);
            }
            if title_clicked && !expansion.pinned {
                close = true;
            }
            if !expansion.pinned
                && !expansion.bend_notice.active()
                && !preserve_tab_press
                && ui.is_mouse_clicked(imgui::MouseButton::Left)
                && !inside_window(ui)
                && !inside(ui.io().mouse_pos, expansion.trigger)
                && !(panel == RibbonPanel::ToolOptions && context.options_menu_hovered)
                && !display.context_contains(ui.io().mouse_pos)
                && !display.selector_open
                && !display
                    .selector_trigger
                    .is_some_and(|bounds| inside(ui.io().mouse_pos, bounds))
            {
                close = true;
            }
            if !expansion.pinned && display.expansion_context_escape(ui, tab, panel) {
                display.temporary_escape_consumed = true;
                close = true;
            }
            if !expansion.pinned
                && context.owners.workspace.part_context_open
                && ui.is_key_pressed_no_repeat(imgui::Key::Escape)
            {
                display.temporary_escape_consumed = true;
                close = true;
            }
            if !expansion.pinned && display.selector_escape {
                close = true;
            }
        });
    close
}
fn expansion_window(ui: &Ui, panel: RibbonPanel) -> imgui::Window<'_, '_, String> {
    // The hidden title strip is another presentation of the same transient ribbon window.
    ui.window(format!("##ribbon-additional-{panel:?}"))
}
fn shows_pin(display: &RibbonDisplayState) -> bool {
    !(display.mode == crate::application::ribbon_display::RibbonDisplay::Tabs
        && display.temporary == Some(crate::application::ribbon_display::TemporaryPanel::WholeTab))
}
fn draw_pin(ui: &Ui) {
    let [x, y] = ui.item_rect_min();
    let scale = ui.current_font_size() / 13.;
    let point = |dx: f32, dy: f32| [x + dx * scale, y + dy * scale];
    let ink = ui.style_color(imgui::StyleColor::Text);
    let draw = ui.get_window_draw_list();
    for (from, to) in [
        ([3., 3.], [10., 3.]),
        ([4., 4.], [4., 8.]),
        ([9., 4.], [9., 8.]),
        ([2., 9.], [11., 9.]),
        ([6.5, 9.], [6.5, 13.]),
    ] {
        draw.add_line(point(from[0], from[1]), point(to[0], to[1]), ink)
            .build();
    }
}
fn inside_window(ui: &Ui) -> bool {
    let position = ui.window_pos();
    let size = ui.window_size();
    inside(
        ui.io().mouse_pos,
        [position, [position[0] + size[0], position[1] + size[1]]],
    )
}
fn inside(point: [f32; 2], bounds: [[f32; 2]; 2]) -> bool {
    (0..2).all(|axis| point[axis] >= bounds[0][axis] && point[axis] <= bounds[1][axis])
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn temporary_tab_footer_hides_pin_only_in_tabs_mode() {
        use crate::application::ribbon_display::{RibbonDisplay, TemporaryPanel};
        let mut display = RibbonDisplayState::default();
        assert!(shows_pin(&display));
        display.mode = RibbonDisplay::Tabs;
        display.temporary = Some(TemporaryPanel::WholeTab);
        assert!(!shows_pin(&display));
        display.mode = RibbonDisplay::PanelTitles;
        display.temporary = Some(TemporaryPanel::Group(RibbonPanel::PartModify));
        assert!(shows_pin(&display));
    }
    #[test]
    fn switching_documents_drops_pinned_expansions() {
        let mut expansions = PanelExpansions::default();
        expansions.use_document(Some(1));
        expansions.open.insert(
            (RibbonTab::Model, RibbonPanel::PartModify),
            Expansion {
                anchor: [0.; 2],
                trigger: [[0.; 2]; 2],
                pinned: true,
                focus: false,
                bounds: [[0.; 2]; 2],
                bend_notice: Default::default(),
            },
        );
        expansions.use_document(Some(1));
        assert_eq!(expansions.open.len(), 1);
        expansions.use_document(Some(2));
        assert!(expansions.open.is_empty());
    }
    #[test]
    fn hit_test_includes_edges_and_excludes_outside_points() {
        let bounds = [[1., 2.], [5., 7.]];
        assert!(inside([1., 7.], bounds));
        assert!(!inside([5.1, 4.], bounds));
    }
    #[test]
    fn changing_display_mode_discards_pinned_expansions() {
        let mut expansions = PanelExpansions::default();
        expansions.use_mode(RibbonDisplay::Full);
        expansions.open.insert(
            (RibbonTab::Model, RibbonPanel::PartModify),
            Expansion {
                anchor: [0.; 2],
                trigger: [[0.; 2]; 2],
                pinned: true,
                focus: false,
                bounds: [[0.; 2]; 2],
                bend_notice: Default::default(),
            },
        );
        expansions.use_mode(RibbonDisplay::Tabs);
        assert!(expansions.open.is_empty());
    }
}
