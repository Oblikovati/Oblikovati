// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard movement follows the current gallery order and stays within its group.
use super::{
    GuiCommandState, GuiTemplateKind, TEMPLATE_GROUPS, TemplateChooser, TemplateFolder,
    ordered_kinds,
};
use imgui::{Key, MouseButton, Ui};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum GalleryMove {
    Previous,
    Next,
    First,
    Last,
}
impl TemplateChooser {
    pub(super) fn clear_gallery_navigation(&mut self) {
        self.gallery_keyboard = false;
        self.gallery_scroll_target = None;
        self.gallery_scroll_frames = 0;
        self.gallery_scroll_y = None;
    }
    pub(super) fn navigate_gallery(&mut self, ui: &Ui, state: &GuiCommandState) {
        self.gallery_scroll_frames = self.gallery_scroll_frames.saturating_sub(1);
        if ui.is_mouse_clicked(MouseButton::Left) || ui.is_mouse_clicked(MouseButton::Right) {
            self.gallery_keyboard = false;
        }
        if !self.gallery_keyboard
            || !ui.is_window_focused()
            || ui.io().want_text_input
            || super::super::super::widget_abi::nested_popup_open(ui)
        {
            return;
        }
        let Some(movement) = pressed_move(ui) else {
            return;
        };
        self.move_selection(state, movement);
        if self.view.is_icon_view() && self.selected.is_some() {
            if let Some(y) = self.gallery_scroll_y {
                ui.set_scroll_y(y);
                self.gallery_scroll_frames = 2;
            }
            self.gallery_scroll_target = Some(match movement {
                GalleryMove::Previous | GalleryMove::First => 0.,
                GalleryMove::Next | GalleryMove::Last => 1.,
            });
        }
    }
    pub(super) fn remember_selected_tile_y(
        &mut self,
        ui: &Ui,
        template: super::GuiDocumentTemplate,
    ) {
        if self.view.is_icon_view() && self.selected == Some(template) && ui.is_item_visible() {
            self.gallery_scroll_y = Some(ui.scroll_y());
        }
    }
    pub(super) fn scroll_selected_tile(&mut self, ui: &Ui, template: super::GuiDocumentTemplate) {
        if self.selected != Some(template) {
            return;
        }
        if let Some(ratio) = self.gallery_scroll_target.take() {
            ui.set_scroll_here_x_with_ratio(ratio);
            self.gallery_scroll_frames = 1;
        }
    }
    fn move_selection(&mut self, state: &GuiCommandState, movement: GalleryMove) {
        let (Some(selected), TemplateFolder::Units(units)) = (self.selected, self.folder) else {
            return;
        };
        if selected.units != units {
            return;
        }
        let kinds = self.navigation_group(selected.kind);
        let ordered = ordered_kinds(&kinds, self.sort, state, units);
        let Some(index) = ordered.iter().position(|kind| *kind == selected.kind) else {
            return;
        };
        let next = move_index(index, ordered.len(), movement);
        if next != index {
            self.selected = Some(super::GuiDocumentTemplate {
                kind: ordered[next],
                units,
            });
            self.drawing_layout = None;
        }
    }
    fn navigation_group(&self, selected: GuiTemplateKind) -> Vec<GuiTemplateKind> {
        TEMPLATE_GROUPS
            .iter()
            .filter(|(_, kinds)| self.ungrouped || kinds.contains(&selected))
            .flat_map(|(_, kinds)| kinds.iter().copied())
            .collect()
    }
}
fn pressed_move(ui: &Ui) -> Option<GalleryMove> {
    [
        (Key::LeftArrow, GalleryMove::Previous),
        (Key::RightArrow, GalleryMove::Next),
        (Key::Home, GalleryMove::First),
        (Key::End, GalleryMove::Last),
    ]
    .into_iter()
    .find_map(|(key, movement)| ui.is_key_pressed_no_repeat(key).then_some(movement))
}
fn move_index(index: usize, count: usize, movement: GalleryMove) -> usize {
    match movement {
        GalleryMove::Previous => index.saturating_sub(1),
        GalleryMove::Next => (index + 1).min(count - 1),
        GalleryMove::First => 0,
        GalleryMove::Last => count - 1,
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn movement_uses_the_visible_sort_and_does_not_cross_groups() {
        use super::super::{GuiDocumentTemplate, GuiTemplateUnits};
        let mut chooser = TemplateChooser::default();
        chooser.selected = Some(GuiDocumentTemplate {
            kind: GuiTemplateKind::SheetMetal,
            units: GuiTemplateUnits::Metric,
        });
        let state = GuiCommandState::default();
        chooser.move_selection(&state, GalleryMove::Next);
        assert_eq!(chooser.selected.unwrap().kind, GuiTemplateKind::Part);
        chooser.move_selection(&state, GalleryMove::Next);
        assert_eq!(chooser.selected.unwrap().kind, GuiTemplateKind::Part);
        chooser.sort = chooser.sort.toggle_name();
        chooser.move_selection(&state, GalleryMove::Next);
        assert_eq!(chooser.selected.unwrap().kind, GuiTemplateKind::SheetMetal);
        chooser.folder = TemplateFolder::Units(GuiTemplateUnits::Inch);
        chooser.move_selection(&state, GalleryMove::First);
        assert_eq!(chooser.selected.unwrap().kind, GuiTemplateKind::SheetMetal);
    }
    #[test]
    fn ungrouped_endpoints_follow_the_full_sorted_catalog() {
        use super::super::{GuiDocumentTemplate, GuiTemplateUnits};
        let mut chooser = TemplateChooser::default();
        chooser.ungrouped = true;
        chooser.selected = Some(GuiDocumentTemplate {
            kind: GuiTemplateKind::Presentation,
            units: GuiTemplateUnits::Metric,
        });
        chooser.move_selection(&GuiCommandState::default(), GalleryMove::First);
        assert_eq!(chooser.selected.unwrap().kind, GuiTemplateKind::Drawing);
    }
    #[test]
    fn closing_the_gallery_clears_pending_navigation_and_redraw() {
        let mut chooser = TemplateChooser::default();
        chooser.gallery_keyboard = true;
        chooser.gallery_scroll_target = Some(1.);
        chooser.gallery_scroll_frames = 2;
        chooser.gallery_scroll_y = Some(42.);
        chooser.clear_gallery_navigation();
        assert!(!chooser.gallery_keyboard);
        assert_eq!(chooser.gallery_scroll_target, None);
        assert_eq!(chooser.gallery_scroll_frames, 0);
        assert_eq!(chooser.gallery_scroll_y, None);
    }
    #[test]
    fn movement_stays_inside_group_endpoints() {
        assert_eq!(move_index(0, 3, GalleryMove::Previous), 0);
        assert_eq!(move_index(2, 3, GalleryMove::Next), 2);
        assert_eq!(move_index(1, 3, GalleryMove::First), 0);
        assert_eq!(move_index(1, 3, GalleryMove::Last), 2);
    }
}
