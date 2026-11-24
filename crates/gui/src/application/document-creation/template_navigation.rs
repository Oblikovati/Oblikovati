// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard movement follows the current gallery order and stays within its group.
use super::{GuiCommandState, GuiTemplateKind, TEMPLATE_GROUPS, TemplateChoice, TemplateChooser};
use imgui::{Key, MouseButton, Ui};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum GalleryMove {
    Previous,
    Next,
    First,
    Last,
}
pub(super) enum GalleryScrollTarget {
    LeftEdge,
    TileStart,
    TileEnd,
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
        if scroll_gallery(ui) {
            self.gallery_scroll_frames = 2;
            return;
        }
        let Some(movement) = pressed_move(ui) else {
            return;
        };
        let Some(index) = self.move_selection(state, movement) else {
            return;
        };
        if self.view.is_icon_view() {
            self.scroll_to_navigation(ui, movement, index);
        }
    }
    fn scroll_to_navigation(&mut self, ui: &Ui, movement: GalleryMove, index: usize) {
        if let Some(y) = self.gallery_scroll_y {
            ui.set_scroll_y(y);
            self.gallery_scroll_frames = 2;
        }
        self.gallery_scroll_target = Some(match movement {
            GalleryMove::Previous | GalleryMove::First if index == 0 => {
                GalleryScrollTarget::LeftEdge
            }
            GalleryMove::Previous | GalleryMove::First => GalleryScrollTarget::TileStart,
            GalleryMove::Next | GalleryMove::Last => GalleryScrollTarget::TileEnd,
        });
    }
    pub(super) fn remember_selected_tile_y(&mut self, ui: &Ui, template: &TemplateChoice) {
        if self.view.is_icon_view()
            && self.selected.as_ref() == Some(template)
            && ui.is_item_visible()
        {
            self.gallery_scroll_y = Some(ui.scroll_y());
        }
    }
    pub(super) fn scroll_selected_tile(&mut self, ui: &Ui, template: &TemplateChoice) {
        if self.selected.as_ref() != Some(template) {
            return;
        }
        if let Some(target) = self.gallery_scroll_target.take() {
            match target {
                // The first tile is indented under its group, so align with the gallery itself.
                GalleryScrollTarget::LeftEdge => ui.set_scroll_x(0.),
                GalleryScrollTarget::TileStart => ui.set_scroll_here_x_with_ratio(0.),
                GalleryScrollTarget::TileEnd => ui.set_scroll_here_x_with_ratio(1.),
            }
            self.gallery_scroll_frames = 1;
        }
    }
    fn move_selection(&mut self, state: &GuiCommandState, movement: GalleryMove) -> Option<usize> {
        let selected = self.selected.as_ref()?;
        let kinds = self.navigation_group(selected.kind());
        let entries = self.catalog(state, &kinds);
        let index = entries.iter().position(|entry| entry == selected)?;
        let next = move_index(index, entries.len(), movement);
        if next != index {
            self.selected = Some(entries[next].clone());
            self.drawing_layout = None;
        }
        Some(next)
    }
    fn navigation_group(&self, selected: GuiTemplateKind) -> Vec<GuiTemplateKind> {
        TEMPLATE_GROUPS
            .iter()
            .filter(|(_, kinds)| self.ungrouped || kinds.contains(&selected))
            .flat_map(|(_, kinds)| kinds.iter().copied())
            .collect()
    }
}
fn scroll_gallery(ui: &Ui) -> bool {
    let direction = if ui.is_key_pressed_no_repeat(Key::DownArrow) {
        1.
    } else if ui.is_key_pressed_no_repeat(Key::UpArrow) {
        -1.
    } else {
        return false;
    };
    let next = (ui.scroll_y() + direction * ui.text_line_height_with_spacing())
        .clamp(0., ui.scroll_max_y());
    ui.set_scroll_y(next);
    true
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
    fn distinct_project_files_with_the_same_kind_are_individually_selectable() {
        use crate::{GuiProjectTemplate, GuiTemplateUnits};

        let make_entry = |title: &str| GuiProjectTemplate {
            path: format!("{title}.opd").into(),
            title: title.into(),
            folder: super::super::TemplateFolder::Units(GuiTemplateUnits::Metric),
            kind: GuiTemplateKind::Part,
            length_unit: "mm",
            size: 42,
            material: None,
            sheet_preview: None,
        };
        let din = make_entry("Standard (DIN)");
        let mm = make_entry("Standard (mm)");
        let state = GuiCommandState {
            project_templates: vec![mm.clone(), din.clone()].into(),
            ..Default::default()
        };
        let mut chooser = TemplateChooser {
            selected: Some(TemplateChoice::Project(din.clone())),
            ..Default::default()
        };
        let _ = chooser.move_selection(&state, GalleryMove::Next);
        assert_eq!(chooser.selected, Some(TemplateChoice::Project(mm)));
        let _ = chooser.move_selection(&state, GalleryMove::Previous);
        assert_eq!(chooser.selected, Some(TemplateChoice::Project(din)));
    }

    #[test]
    fn movement_uses_the_visible_sort_and_does_not_cross_groups() {
        use super::super::{GuiDocumentTemplate, GuiTemplateUnits, TemplateFolder};
        let mut chooser = TemplateChooser {
            selected: Some(
                GuiDocumentTemplate {
                    kind: GuiTemplateKind::SheetMetal,
                    units: GuiTemplateUnits::Metric,
                }
                .into(),
            ),
            ..Default::default()
        };
        let state = GuiCommandState::default();
        let _ = chooser.move_selection(&state, GalleryMove::Next);
        assert_eq!(
            chooser.selected.as_ref().unwrap().kind(),
            GuiTemplateKind::Part
        );
        let _ = chooser.move_selection(&state, GalleryMove::Next);
        assert_eq!(
            chooser.selected.as_ref().unwrap().kind(),
            GuiTemplateKind::Part
        );
        chooser.sort = chooser.sort.toggle_name();
        let _ = chooser.move_selection(&state, GalleryMove::Next);
        assert_eq!(
            chooser.selected.as_ref().unwrap().kind(),
            GuiTemplateKind::SheetMetal
        );
        chooser.folder = TemplateFolder::Units(GuiTemplateUnits::Inch);
        let _ = chooser.move_selection(&state, GalleryMove::First);
        assert_eq!(
            chooser.selected.as_ref().unwrap().kind(),
            GuiTemplateKind::SheetMetal
        );
    }
    #[test]
    fn ungrouped_endpoints_follow_the_full_sorted_catalog() {
        use super::super::{GuiDocumentTemplate, GuiTemplateUnits};
        let mut chooser = TemplateChooser {
            ungrouped: true,
            selected: Some(
                GuiDocumentTemplate {
                    kind: GuiTemplateKind::Presentation,
                    units: GuiTemplateUnits::Metric,
                }
                .into(),
            ),
            ..Default::default()
        };
        let _ = chooser.move_selection(&GuiCommandState::default(), GalleryMove::First);
        assert_eq!(
            chooser.selected.as_ref().unwrap().kind(),
            GuiTemplateKind::Drawing
        );
    }
    #[test]
    fn closing_the_gallery_clears_pending_navigation_and_redraw() {
        let mut chooser = TemplateChooser {
            gallery_keyboard: true,
            gallery_scroll_target: Some(GalleryScrollTarget::TileEnd),
            gallery_scroll_frames: 2,
            gallery_scroll_y: Some(42.),
            ..Default::default()
        };
        chooser.clear_gallery_navigation();
        assert!(!chooser.gallery_keyboard);
        assert!(chooser.gallery_scroll_target.is_none());
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
