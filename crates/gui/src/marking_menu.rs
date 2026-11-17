// SPDX-License-Identifier: GPL-2.0-only
//! Shared radial layout for configurable canvas menus.
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_api::wire::MarkingMenuView;

pub(crate) type ResolveCommand = fn(&str) -> Option<(&'static str, GuiShellAction)>;

pub(crate) fn draw(
    ui: &Ui,
    menu: &MarkingMenuView,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    resolve: ResolveCommand,
) {
    let slots = resolved_slots(ui, menu, resolve);
    if !slots.is_empty() {
        draw_ring(ui, slots, state, actions);
    }
    ui.separator();
    draw_overflow(ui, menu, state, actions, resolve);
}
fn draw_ring(
    ui: &Ui,
    slots: Vec<Slot>,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    let ring = slot_layout(&slots);
    let origin = ui.cursor_pos();
    ui.dummy([ring.size, ring.size]);
    for slot in slots {
        draw_slot(ui, slot, &ring, origin, state, actions);
    }
    ui.set_cursor_pos([
        origin[0],
        origin[1] + ring.size + ui.clone_style().item_spacing[1],
    ]);
}
pub(crate) fn ring_size(ui: &Ui, menu: &MarkingMenuView, resolve: ResolveCommand) -> Option<f32> {
    let slots = resolved_slots(ui, menu, resolve);
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
        if let Some((label, command)) = resolve(&item.command_id)
            && ui
                .menu_item_config(label)
                .enabled(command.enabled(state))
                .build()
        {
            actions.push(command);
        }
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
        let Some((label, command)) = resolve(id) else {
            continue;
        };
        if ui
            .menu_item_config(label)
            .enabled(command.enabled(state))
            .build()
        {
            actions.push(command);
        }
    }
}
struct Slot {
    quadrant: u8,
    label: &'static str,
    action: GuiShellAction,
    size: [f32; 2],
}
fn resolved_slots(ui: &Ui, menu: &MarkingMenuView, resolve: ResolveCommand) -> Vec<Slot> {
    let padding = ui.clone_style().frame_padding;
    menu.quadrants
        .iter()
        .flatten()
        .filter_map(|item| {
            let (label, action) = resolve(&item.command_id)?;
            let text = ui.calc_text_size(label);
            Some(Slot {
                quadrant: item.quadrant.0,
                label,
                action,
                size: [text[0] + 2. * padding[0], ui.frame_height()],
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
) {
    let Some(position) = ring.position(slot.quadrant, slot.size) else {
        return;
    };
    ui.set_cursor_pos([origin[0] + position[0], origin[1] + position[1]]);
    let disabled = ui.begin_disabled(!slot.action.enabled(state));
    let pressed = ui.button_with_size(
        format!("{}##marking-{}", slot.label, slot.quadrant),
        slot.size,
    );
    drop(disabled);
    if pressed {
        actions.push(slot.action);
        ui.close_current_popup();
    }
}
