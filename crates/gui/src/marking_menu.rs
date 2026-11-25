// SPDX-License-Identifier: GPL-2.0-only
//! Shared radial layout for configurable canvas menus.
use crate::icons::{GuiRibbonIcons, IconGlyph, IconSize};
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_api::wire::MarkingMenuView;
use oblikovati_model::marking_menu::{MarkingMenuOverflow, MarkingMenuStyle};

pub(crate) type ResolveCommand = fn(&str) -> Option<MarkingEntry>;

/// A rendered entry with an action only when its capability is available.
/// For example, `MarkingEntry::disabled("Select Tangencies")` shows an inactive slot.
#[derive(Debug, PartialEq)]
pub(crate) struct MarkingEntry {
    label: &'static str,
    action: Option<GuiShellAction>,
}

impl MarkingEntry {
    /// Bind a caption to its host action. For example, `MarkingEntry::command(("Save", GuiShellAction::Save))`.
    pub(crate) fn command((label, action): (&'static str, GuiShellAction)) -> Self {
        Self {
            label,
            action: Some(action),
        }
    }

    /// Show a caption with no host action. For example, `MarkingEntry::disabled("Select Other...")`.
    pub(crate) fn disabled(label: &'static str) -> Self {
        Self {
            label,
            action: None,
        }
    }

    fn dispatchable(&self, state: &GuiCommandState) -> bool {
        self.action
            .as_ref()
            .is_some_and(|action| action.enabled(state))
    }
}

/// Use the linear context menu when marking is disabled or the user chose classic.
/// Example: `use_classic(false, false)` returns `true`.
pub(crate) fn use_classic(configured_classic: bool, marking_enabled: bool) -> bool {
    configured_classic || !marking_enabled
}

pub(crate) fn draw(
    ui: &Ui,
    menu: &MarkingMenuView,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    resolve: ResolveCommand,
    icons: &GuiRibbonIcons,
) {
    let style = state.marking_menu_display.style;
    let slots = resolved_slots(ui, menu, resolve, style, icons);
    if !slots.is_empty() {
        draw_ring(ui, slots, state, actions, icons);
    }
    if state.marking_menu_display.overflow != MarkingMenuOverflow::RadialOnly {
        ui.separator();
        draw_overflow(ui, menu, state, actions, resolve);
    }
}
fn draw_ring(
    ui: &Ui,
    slots: Vec<Slot>,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    icons: &GuiRibbonIcons,
) {
    let ring = slot_layout(&slots);
    let origin = ui.cursor_pos();
    ui.dummy([ring.size, ring.size]);
    for slot in slots {
        draw_slot(ui, slot, &ring, origin, state, actions, icons);
    }
    ui.set_cursor_pos([
        origin[0],
        origin[1] + ring.size + ui.clone_style().item_spacing[1],
    ]);
}
pub(crate) fn ring_size(
    ui: &Ui,
    menu: &MarkingMenuView,
    resolve: ResolveCommand,
    style: MarkingMenuStyle,
    icons: &GuiRibbonIcons,
) -> Option<f32> {
    let slots = resolved_slots(ui, menu, resolve, style, icons);
    (!slots.is_empty()).then(|| slot_layout(&slots).size)
}
fn slot_layout(slots: &[Slot]) -> crate::marking_menu_layout::RingLayout {
    let maximum = slots.iter().fold([0.0_f32; 2], |maximum, slot| {
        [maximum[0].max(slot.size[0]), maximum[1].max(slot.size[1])]
    });
    crate::marking_menu_layout::RingLayout::new(maximum)
}
pub(crate) fn draw_classic(
    ui: &Ui,
    menu: &MarkingMenuView,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    resolve: ResolveCommand,
) {
    for item in menu.quadrants.iter().flatten() {
        draw_row(ui, state, actions, resolve(&item.command_id));
    }
    draw_overflow(ui, menu, state, actions, resolve);
    ui.separator();
}
pub(crate) fn overflow_count(menu: &MarkingMenuView, resolve: ResolveCommand) -> usize {
    menu.overflow
        .iter()
        .flatten()
        .filter(|id| resolve(id).is_some())
        .count()
}
fn draw_overflow(
    ui: &Ui,
    menu: &MarkingMenuView,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    resolve: ResolveCommand,
) {
    for id in menu.overflow.iter().flatten() {
        draw_row(ui, state, actions, resolve(id));
    }
}
fn draw_row(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    entry: Option<MarkingEntry>,
) {
    let Some(entry) = entry else { return };
    if ui
        .menu_item_config(entry.label)
        .enabled(entry.dispatchable(state))
        .build()
        && let Some(action) = entry.action
    {
        actions.push(action);
    }
}
struct Slot {
    quadrant: u8,
    entry: MarkingEntry,
    size: [f32; 2],
}
fn resolved_slots(
    ui: &Ui,
    menu: &MarkingMenuView,
    resolve: ResolveCommand,
    style: MarkingMenuStyle,
    icons: &GuiRibbonIcons,
) -> Vec<Slot> {
    let padding = ui.clone_style().frame_padding;
    menu.quadrants
        .iter()
        .flatten()
        .filter_map(|item| {
            let entry = resolve(&item.command_id)?;
            let text = ui.calc_text_size(entry.label);
            let [icon_width, icon_height] = icons.logical[IconSize::Small.index()];
            let size = match style {
                MarkingMenuStyle::IconAndText => [
                    text[0] + icon_width + (2. * padding[0]).max(7.),
                    ui.frame_height().max(icon_height + 4.),
                ],
                MarkingMenuStyle::IconOnly => [icon_width + 8., icon_height + 4.],
                MarkingMenuStyle::TextOnly => [text[0] + 2. * padding[0], ui.frame_height()],
            };
            Some(Slot {
                quadrant: item.quadrant.0,
                entry,
                size,
            })
        })
        .collect()
}
fn draw_slot(
    ui: &Ui,
    slot: Slot,
    ring: &crate::marking_menu_layout::RingLayout,
    origin: [f32; 2],
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    icons: &GuiRibbonIcons,
) {
    let Some(position) = ring.position(slot.quadrant, slot.size) else {
        return;
    };
    ui.set_cursor_pos([origin[0] + position[0], origin[1] + position[1]]);
    let disabled = ui.begin_disabled(!slot.entry.dispatchable(state));
    let pressed = slot_button(ui, &slot, state.marking_menu_display.style, icons);
    drop(disabled);
    if pressed && let Some(action) = slot.entry.action {
        actions.push(action);
        ui.close_current_popup();
    }
}

fn slot_button(ui: &Ui, slot: &Slot, style: MarkingMenuStyle, icons: &GuiRibbonIcons) -> bool {
    let label = format!("{}##marking-{}", slot.entry.label, slot.quadrant);
    if style == MarkingMenuStyle::TextOnly {
        return ui.button_with_size(label, slot.size);
    }
    let glyph = match slot.entry.action.as_ref() {
        Some(GuiShellAction::OpenExtrude) => IconGlyph::Extrude,
        Some(GuiShellAction::OpenFillet) => IconGlyph::Fillet,
        Some(GuiShellAction::OpenChamfer) => IconGlyph::Chamfer,
        Some(GuiShellAction::OpenOffsetPlane) => IconGlyph::WorkPlane,
        Some(action) => crate::icons::action_glyph(action).unwrap_or(IconGlyph::Customize),
        None => IconGlyph::Select,
    };
    let pressed = crate::widget_abi::icon_button(
        ui,
        &label,
        crate::widget_abi::IconButtonPresentation {
            texture: icons.key(glyph, IconSize::Small),
            size: slot.size,
            icon: icons.logical[IconSize::Small.index()],
            large: false,
            show_caption: style == MarkingMenuStyle::IconAndText,
        },
    );
    if style == MarkingMenuStyle::IconOnly
        && ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED)
    {
        ui.tooltip_text(slot.entry.label);
    }
    pressed
}

#[cfg(test)]
mod visibility_tests {
    use super::use_classic;
    #[test]
    fn disabled_marking_preserves_classic_choice() {
        assert!(!use_classic(false, true));
        assert!(use_classic(false, false));
        assert!(use_classic(true, true));
        assert!(use_classic(true, false));
    }
}
