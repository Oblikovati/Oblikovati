// SPDX-License-Identifier: GPL-2.0-only
//! Keep one tab order while the visible part of the strip changes.
use std::collections::BTreeSet;

#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub(super) enum TabKey {
    Home,
    Window(u64),
}

#[derive(Default)]
pub(super) struct TabOrder {
    order: Vec<TabKey>,
    visible: BTreeSet<TabKey>,
    settling: u8,
    pressed: bool,
    dragging: bool,
    positions: Vec<(TabKey, f32)>,
}
impl TabOrder {
    pub(super) fn synchronize_state(&mut self, state: &crate::GuiCommandState) {
        let keys: Vec<_> = std::iter::once(TabKey::Home)
            .chain(
                crate::application::document_tabs::windows::all_tab_rows(state)
                    .into_iter()
                    .map(|tab| TabKey::Window(tab.window)),
            )
            .collect();
        self.synchronize(&keys);
    }
    /// Retain the order of open tabs; example: `order.synchronize(&keys)`.
    pub(super) fn synchronize(&mut self, keys: &[TabKey]) {
        let open: BTreeSet<_> = keys.iter().copied().collect();
        self.order.retain(|key| open.contains(key));
        for key in keys {
            if !self.order.contains(key) {
                self.order.push(*key);
            }
        }
    }
    /// The retained position of a tab; example: `order.rank(TabKey::Home)`.
    pub(super) fn rank(&self, key: TabKey) -> Option<usize> {
        self.order.iter().position(|known| *known == key)
    }
    /// Reset ImGui's order when the visible set changes; example: `order.subset(&keys)`.
    pub(super) fn subset(&mut self, keys: &[TabKey]) -> bool {
        let visible = keys.iter().copied().collect();
        let changed = self.visible != visible;
        if changed {
            self.visible = visible;
            self.settling = 2;
        }
        self.positions.clear();
        changed
    }
    /// Let ImGui reorder only after returned tabs settle; example: `order.reorderable()`.
    pub(super) fn reorderable(&mut self) -> bool {
        if self.settling == 0 {
            return true;
        }
        self.settling -= 1;
        false
    }
    /// A new press cannot inherit a previous drag; example: `order.begin_input(true)`.
    pub(super) fn begin_input(&mut self, clicked: bool) {
        if clicked {
            self.pressed = false;
            self.dragging = false;
        }
    }
    /// A hidden strip cannot keep an unfinished drag; example: `order.cancel_input()`.
    pub(super) fn cancel_input(&mut self) {
        self.pressed = false;
        self.dragging = false;
    }
    /// Record the drawn position; example: `order.note(TabKey::Home, 20.)`.
    pub(super) fn note(&mut self, key: TabKey, x: f32, pressed: bool) {
        self.positions.push((key, x));
        self.pressed |= pressed;
    }
    /// Adopt physical positions only after an actual tab drag; example: `order.observe_drag(true, false)`.
    pub(super) fn observe_drag(&mut self, dragging: bool, released: bool) -> bool {
        self.dragging |= self.pressed && dragging;
        let changed = self.dragging && apply_positions(&mut self.order, &self.positions);
        if released {
            self.dragging = false;
            self.pressed = false;
        }
        changed
    }
}

fn apply_positions(order: &mut [TabKey], positions: &[(TabKey, f32)]) -> bool {
    let mut physical = positions.to_vec();
    physical.sort_by(|a, b| a.1.total_cmp(&b.1));
    let visible: BTreeSet<_> = physical.iter().map(|(key, _)| *key).collect();
    if visible.len() != physical.len() {
        return false;
    }
    let slots: Vec<usize> = order
        .iter()
        .enumerate()
        .filter_map(|(i, key)| visible.contains(key).then_some(i))
        .collect();
    if slots.len() != physical.len() {
        return false;
    }
    let changed = slots
        .iter()
        .zip(&physical)
        .any(|(slot, (key, _))| order[*slot] != *key);
    for (slot, (key, _)) in slots.into_iter().zip(physical) {
        order[slot] = key;
    }
    changed
}

#[cfg(test)]
mod tests {
    use super::{TabKey, TabOrder, apply_positions};
    #[test]
    fn dragged_subset_preserves_other_slots_and_home_identity() {
        let mut order = [
            TabKey::Home,
            TabKey::Window(0),
            TabKey::Window(2),
            TabKey::Window(3),
        ];
        assert!(apply_positions(
            &mut order,
            &[
                (TabKey::Window(2), 1.),
                (TabKey::Home, 2.),
                (TabKey::Window(0), 3.)
            ]
        ));
        assert_eq!(
            order,
            [
                TabKey::Window(2),
                TabKey::Home,
                TabKey::Window(0),
                TabKey::Window(3)
            ]
        );
    }
    #[test]
    fn sync_removes_closed_tabs_and_appends_new_without_resetting_order() {
        let mut order = TabOrder::default();
        order.synchronize(&[
            TabKey::Home,
            TabKey::Window(0),
            TabKey::Window(1),
            TabKey::Window(2),
        ]);
        apply_positions(
            &mut order.order,
            &[
                (TabKey::Window(2), 0.),
                (TabKey::Home, 1.),
                (TabKey::Window(0), 2.),
                (TabKey::Window(1), 3.),
            ],
        );
        order.synchronize(&[
            TabKey::Home,
            TabKey::Window(0),
            TabKey::Window(2),
            TabKey::Window(3),
        ]);
        assert_eq!(
            order.order,
            [
                TabKey::Window(2),
                TabKey::Home,
                TabKey::Window(0),
                TabKey::Window(3)
            ]
        );
    }
    #[test]
    fn hidden_open_document_keeps_its_dragged_order() {
        let mut state = crate::GuiCommandState::default();
        state.documents = (1..=3)
            .map(|id| oblikovati_api::wire::DocumentInfo {
                id,
                visible: true,
                ..Default::default()
            })
            .collect();
        let mut order = TabOrder::default();
        order.synchronize_state(&state);
        apply_positions(
            &mut order.order,
            &[(TabKey::Window(3), 1.), (TabKey::Window(1), 2.)],
        );
        state.documents[2].visible = false;
        order.synchronize_state(&state);
        state.documents[2].visible = true;
        order.synchronize_state(&state);
        assert!(order.rank(TabKey::Window(3)) < order.rank(TabKey::Window(1)));
    }
    #[test]
    fn pointer_drag_without_tab_press_does_not_change_order() {
        let mut order = TabOrder::default();
        order.synchronize(&[TabKey::Home, TabKey::Window(1)]);
        order.begin_input(true);
        order.note(TabKey::Window(1), 1., false);
        order.note(TabKey::Home, 2., false);
        assert!(!order.observe_drag(true, true));
        assert_eq!(order.order, [TabKey::Home, TabKey::Window(1)]);
    }
    #[test]
    fn invalid_or_duplicate_observations_do_not_change_order() {
        let mut order = [TabKey::Home, TabKey::Window(1)];
        assert!(!apply_positions(
            &mut order,
            &[(TabKey::Home, 1.), (TabKey::Home, 2.)]
        ));
        assert!(!apply_positions(&mut order, &[(TabKey::Window(9), 1.)]));
        assert_eq!(order, [TabKey::Home, TabKey::Window(1)]);
    }
}
