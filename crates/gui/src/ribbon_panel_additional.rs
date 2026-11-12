// SPDX-License-Identifier: GPL-2.0-only
//! Additional ribbon commands stay anchored to their panel and can be pinned.
use super::*;
const ADDITIONAL_WIDTH: f32 = 230.;
const ADDITIONAL_HEIGHT: f32 = 130.;
const VIEWPORT_MARGIN: f32 = 8.;

#[derive(Default)]
pub(super) struct PanelExpansions {
    open: BTreeMap<(RibbonTab, RibbonPanel), Expansion>,
    document: Option<u64>,
    mode: Option<RibbonDisplay>,
}
#[derive(Clone, Copy)]
struct Expansion {
    anchor: [f32; 2],
    trigger: [[f32; 2]; 2],
    pinned: bool,
    focus: bool,
    bounds: [[f32; 2]; 2],
}
impl PanelExpansions {
    pub(super) fn clear(&mut self) {
        self.open.clear();
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
    pub(super) fn contains(&self, point: [f32; 2]) -> bool {
        self.open.values().any(|entry| inside(point, entry.bounds))
    }
}
pub(super) fn available(tab: RibbonTab, panel: RibbonPanel) -> bool {
    matches!(
        (tab, panel),
        (
            RibbonTab::Model,
            RibbonPanel::PartModify | RibbonPanel::PartSimplification
        ) | (
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
    let clicked = ui.small_button("v##additional-commands");
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
        layout.additional.open.insert(
            key,
            Expansion {
                anchor,
                trigger,
                pinned: false,
                focus: true,
                bounds: [[0.; 2]; 2],
            },
        );
    }
    if let Some(expansion) = layout.additional.open.get_mut(&key) {
        expansion.anchor = anchor;
        expansion.trigger = trigger;
    }
    if ui.is_item_hovered() {
        ui.tooltip_text("Additional commands");
    }
}
pub(super) fn draw(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    layout: &mut RibbonPanelLayout,
    display: &RibbonDisplayState,
) -> bool {
    let mut closed = false;
    layout.additional.open.retain(|(owner, panel), _| {
        *owner == tab
            && !display.hidden_panels.contains(&(tab, *panel))
            && context.additional_allowed
    });
    let keys: Vec<_> = layout.additional.open.keys().copied().collect();
    for key in keys {
        let Some(expansion) = layout.additional.open.get_mut(&key) else {
            continue;
        };
        let area = context
            .owners
            .workspace
            .document_area(context.ui.io().display_size, context.ribbon_bottom);
        if draw_window(context, key.1, area, expansion, display) {
            layout.additional.open.remove(&key);
            closed = true;
        }
    }
    closed
}
fn draw_window(
    context: &mut PanelDrawContext<'_>,
    panel: RibbonPanel,
    area: crate::application::document_creation::workspaces::WorkspaceArea,
    expansion: &mut Expansion,
    display: &RibbonDisplayState,
) -> bool {
    let ui = context.ui;
    let scale = context.owners.icons.layout_factor();
    let [viewport_width, viewport_height] = ui.io().display_size;
    let left = area.left.clamp(0., (viewport_width - 1.).max(0.));
    let right = area.right.clamp(left + 1., viewport_width.max(left + 1.));
    let width = (ADDITIONAL_WIDTH * scale).min((right - left - VIEWPORT_MARGIN).max(1.));
    let top = area.top.min((viewport_height - 1.).max(0.));
    let height = (ADDITIONAL_HEIGHT * scale).min((viewport_height - top).max(1.));
    let x = expansion.anchor[0].clamp(left, (right - width).max(left));
    expansion.bounds = [[x, top], [x + width, top + height]];
    crate::widget_abi::window_geometry(ui, [x, top], [width, height], imgui::Condition::Always);
    let mut close = false;
    let focus = std::mem::take(&mut expansion.focus);
    ui.window(format!("##ribbon-additional-{panel:?}"))
        .flags(imgui::WindowFlags::NO_DOCKING | imgui::WindowFlags::NO_SAVED_SETTINGS)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .collapsible(false)
        .focused(focus)
        .build(|| {
            commands(
                ui,
                panel,
                context.state,
                context.owners.precise_input,
                context.owners.create_block,
                context.actions,
            );
            ui.separator();
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
            if ui.small_button(panel.label()) && !expansion.pinned {
                close = true;
            }
            if !expansion.pinned
                && ui.is_mouse_clicked(imgui::MouseButton::Left)
                && !inside_window(ui)
                && !inside(ui.io().mouse_pos, expansion.trigger)
                && !display.selector_open
                && !display
                    .selector_trigger
                    .is_some_and(|bounds| inside(ui.io().mouse_pos, bounds))
            {
                close = true;
            }
            if !expansion.pinned && display.selector_escape {
                close = true;
            }
        });
    close
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
fn commands(
    ui: &Ui,
    panel: RibbonPanel,
    state: &GuiCommandState,
    precise_input: &mut crate::sketch::precise_input::PreciseInputBar,
    create_block: &mut crate::sketch::create_block::CreateBlockDialog,
    actions: &mut Vec<GuiShellAction>,
) {
    if panel == RibbonPanel::Create {
        create_commands(ui, state, precise_input, create_block, actions);
        return;
    }
    let disabled = ui.begin_disabled(true);
    let labels = match panel {
        RibbonPanel::PartModify => &["Move Bodies", "Bend Part", "Copy Object"][..],
        RibbonPanel::PartSimplification => {
            &["Remove Details", "Fill Voids", "Define Envelopes"][..]
        }
        RibbonPanel::Constrain => &["Edit Coordinate System", "Constraint Inference Scope"][..],
        _ => return,
    };
    for &label in labels {
        ui.button(label);
        if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
            ui.tooltip_text("This command is not available in this build.");
        }
    }
    drop(disabled);
}
fn create_commands(
    ui: &Ui,
    state: &GuiCommandState,
    precise_input: &mut crate::sketch::precise_input::PreciseInputBar,
    create_block: &mut crate::sketch::create_block::CreateBlockDialog,
    actions: &mut Vec<GuiShellAction>,
) {
    let unavailable = ui.begin_disabled(!state.sketch_at_ready || state.transaction_open);
    if ui.button("Create Block") {
        create_block.open(state);
        actions.push(GuiShellAction::SelectBlockGeometry(true));
    }
    drop(unavailable);
    let disabled = ui.begin_disabled(state.precise_target.is_none() || state.transaction_open);
    precise_input.toggle(ui);
    drop(disabled);
}
fn inside(point: [f32; 2], bounds: [[f32; 2]; 2]) -> bool {
    (0..2).all(|axis| point[axis] >= bounds[0][axis] && point[axis] <= bounds[1][axis])
}

#[cfg(test)]
mod tests {
    use super::*;
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
            },
        );
        expansions.use_mode(RibbonDisplay::Tabs);
        assert!(expansions.open.is_empty());
    }
}
