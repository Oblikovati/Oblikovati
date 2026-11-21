// SPDX-License-Identifier: GPL-2.0-only
//! The Favorites pane of the browser dock: a saved list of content, as a tree or as thumbnails.
use imgui::Ui;

const VIEW_MENU: &str = "##favorites-view";
const TREE_VIEW: &str = "Tree View";
const THUMBNAIL_VIEW: &str = "Thumbnail View";
const LOCATION_CHOICES: [&str; 2] = ["Favorites", "Library"];
const EMPTY: &str = "No favorites yet";

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
enum FavoritesView {
    #[default]
    Tree,
    Thumbnails,
}
#[derive(Debug, Default)]
pub(crate) struct FavoritesPane {
    view: FavoritesView,
    location: usize,
    search: String,
}
impl FavoritesPane {
    /// Start as a tree again, as a newly added pane does.
    pub(crate) fn reset(&mut self) {
        self.view = FavoritesView::Tree;
    }
    /// Draw the pane's controls; example: when its tab is active.
    pub(crate) fn panel(&mut self, ui: &Ui) {
        let label = match self.view {
            FavoritesView::Tree => TREE_VIEW,
            FavoritesView::Thumbnails => THUMBNAIL_VIEW,
        };
        if crate::widget_abi::button(ui, label, [0.; 2]) {
            ui.open_popup(VIEW_MENU);
        }
        self.view_menu(ui);
        ui.input_text("##favorites-search", &mut self.search)
            .hint("Search")
            .build();
        if self.view == FavoritesView::Thumbnails {
            ui.text("Content Location");
            ui.set_next_item_width(-1.);
            ui.combo_simple_string("##content-location", &mut self.location, &LOCATION_CHOICES);
        }
        ui.text_disabled(EMPTY);
    }
    fn view_menu(&mut self, ui: &Ui) {
        let Some(_menu) = ui.begin_popup(VIEW_MENU) else {
            return;
        };
        for (view, label) in [
            (FavoritesView::Tree, TREE_VIEW),
            (FavoritesView::Thumbnails, THUMBNAIL_VIEW),
        ] {
            if ui
                .menu_item_config(label)
                .selected(self.view == view)
                .build()
            {
                self.view = view;
            }
        }
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn a_reset_pane_is_a_tree_again() {
        let mut pane = FavoritesPane {
            view: FavoritesView::Thumbnails,
            ..Default::default()
        };
        pane.reset();
        assert_eq!(pane.view, FavoritesView::Tree);
    }
}
