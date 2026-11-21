// SPDX-License-Identifier: GPL-2.0-only
//! Shared radial layout for configurable canvas menus.
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_api::wire::MarkingMenuView;

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
fn resolved_slots(ui: &Ui, menu: &MarkingMenuView, resolve: ResolveCommand) -> Vec<Slot> {
    let padding = ui.clone_style().frame_padding;
    menu.quadrants
        .iter()
        .flatten()
        .filter_map(|item| {
            let entry = resolve(&item.command_id)?;
            let text = ui.calc_text_size(entry.label);
            Some(Slot {
                quadrant: item.quadrant.0,
                entry,
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
    let disabled = ui.begin_disabled(!slot.entry.dispatchable(state));
    let pressed = ui.button_with_size(
        format!("{}##marking-{}", slot.entry.label, slot.quadrant),
        slot.size,
    );
    drop(disabled);
    if pressed && let Some(action) = slot.entry.action {
        actions.push(action);
        ui.close_current_popup();
    }
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
