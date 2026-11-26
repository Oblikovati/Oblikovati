// SPDX-License-Identifier: GPL-2.0-only
//! Independent document and library browsing for both Part asset windows.
use crate::{GuiAssetChoice, GuiAssetLibrary};
use imgui::Ui;

#[derive(Default)]
pub(crate) struct AssetBrowserView {
    query: String,
    document: DisplayOptions,
    library: DisplayOptions,
    source: Option<GuiAssetLibrary>,
    category: Option<(GuiAssetLibrary, &'static str)>,
}

#[derive(Default)]
struct DisplayOptions {
    usage: usize,
    presentation: usize,
    sort: usize,
    thumbnail: usize,
}

impl AssetBrowserView {
    pub(crate) fn draw(
        &mut self,
        ui: &Ui,
        choices: &[GuiAssetChoice],
        selected: &mut Option<String>,
    ) {
        self.search(ui);
        let query = self.query.trim().to_lowercase();
        let matches = |entry: &&GuiAssetChoice| {
            query.is_empty()
                || entry.name.to_lowercase().contains(&query)
                || entry.id.to_lowercase().contains(&query)
        };
        let height = (ui.content_region_avail()[1] * 0.3).clamp(42., 130.);
        self.document_region(
            ui,
            choices.iter().filter(|row| row.document).filter(matches),
            selected,
            height,
            &query,
        );
        let library: Vec<_> = choices
            .iter()
            .filter(|row| row.library.is_some())
            .filter(matches)
            .collect();
        self.library_region(ui, &library, selected, !query.is_empty());
    }

    fn search(&mut self, ui: &Ui) {
        ui.set_next_item_width((ui.content_region_avail()[0] - 55.).max(40.));
        if ui
            .input_text("##asset-search", &mut self.query)
            .hint("Search assets")
            .build()
        {
            self.category = None;
        }
        ui.same_line();
        if ui.button("Clear##asset-search") {
            self.query.clear();
            self.category = None;
        }
    }

    fn document_region<'a>(
        &mut self,
        ui: &Ui,
        rows: impl Iterator<Item = &'a GuiAssetChoice>,
        selected: &mut Option<String>,
        height: f32,
        query: &str,
    ) {
        ui.text("Document assets");
        ui.same_line_with_spacing(0., 14.);
        self.document.menu(ui, "Document view", true);
        let rows: Vec<_> = rows.filter(|row| self.document.includes(row)).collect();
        ui.child_window("##document-assets")
            .size([0., height])
            .build(|| {
                if rows.is_empty() {
                    ui.text(if query.is_empty() {
                        "No document assets"
                    } else {
                        "No matching document assets"
                    });
                } else {
                    self.document.rows(ui, rows, selected, false);
                }
            });
    }

    fn library_region(
        &mut self,
        ui: &Ui,
        rows: &[&GuiAssetChoice],
        selected: &mut Option<String>,
        searching: bool,
    ) {
        self.library_selector(ui);
        ui.same_line_with_spacing(0., 14.);
        self.library.menu(ui, "Library view", false);
        let rows: Vec<_> = rows
            .iter()
            .copied()
            .filter(|row| self.source.is_none() || row.library == self.source)
            .collect();
        let height = (ui.content_region_avail()[1] - ui.frame_height_with_spacing() * 2.).max(64.);
        let tree_width = (ui.content_region_avail()[0] * 0.32).clamp(80., 155.);
        ui.child_window("##asset-categories")
            .size([tree_width, height])
            .build(|| self.category_tree(ui, &rows));
        ui.same_line();
        ui.child_window("##library-assets")
            .size([0., height])
            .build(|| {
                let rows: Vec<_> = rows
                    .into_iter()
                    .filter(|row| {
                        self.category.is_none_or(|(library, category)| {
                            row.library == Some(library) && asset_category(row) == category
                        })
                    })
                    .collect();
                if rows.is_empty() {
                    ui.text("No matching library assets");
                } else {
                    self.library.rows(ui, rows, selected, searching);
                }
            });
    }

    fn library_selector(&mut self, ui: &Ui) {
        ui.set_next_item_width(126.);
        let preview = self.source.map_or("All libraries", library_name);
        if let Some(_combo) = ui.begin_combo("##asset-library", preview) {
            if ui.selectable("All libraries") {
                self.source = None;
                self.category = None;
            }
            for source in [GuiAssetLibrary::Shipped, GuiAssetLibrary::Project] {
                if ui.selectable(library_name(source)) {
                    self.source = Some(source);
                    self.category = None;
                }
            }
        }
    }

    fn category_tree(&mut self, ui: &Ui, rows: &[&GuiAssetChoice]) {
        if ui
            .selectable_config("All categories")
            .selected(self.category.is_none())
            .build()
        {
            self.category = None;
        }
        for source in [GuiAssetLibrary::Shipped, GuiAssetLibrary::Project] {
            let categories: std::collections::BTreeSet<_> = rows
                .iter()
                .filter(|row| row.library == Some(source))
                .map(|row| asset_category(row))
                .collect();
            if categories.is_empty() {
                continue;
            }
            if let Some(_tree) = ui
                .tree_node_config(library_name(source))
                .default_open(true)
                .push()
            {
                for category in categories {
                    if ui
                        .selectable_config(category)
                        .selected(self.category == Some((source, category)))
                        .build()
                    {
                        self.category = Some((source, category));
                    }
                }
            }
        }
    }
}

fn asset_category(row: &GuiAssetChoice) -> &'static str {
    row.category.unwrap_or("Uncategorized")
}

fn library_name(source: GuiAssetLibrary) -> &'static str {
    match source {
        GuiAssetLibrary::Project => "Project library",
        GuiAssetLibrary::Shipped => "Built-in",
    }
}

impl DisplayOptions {
    fn menu(&mut self, ui: &Ui, label: &str, document: bool) {
        let _id = ui.push_id(label);
        if ui.small_button(label) {
            ui.open_popup("##display");
        }
        let Some(_popup) = ui.begin_popup("##display") else {
            return;
        };
        if document {
            ui.set_next_item_width(136.);
            ui.combo_simple_string(
                "Show",
                &mut self.usage,
                &["Show All", "Show Used", "Show Unused"],
            );
        }
        ui.set_next_item_width(136.);
        ui.combo_simple_string(
            "View",
            &mut self.presentation,
            &["List View", "Thumbnail View", "Text View"],
        );
        ui.set_next_item_width(136.);
        ui.combo_simple_string(
            "Sort by",
            &mut self.sort,
            &["Name", "Material Color", "Category"],
        );
        ui.set_next_item_width(136.);
        ui.combo_simple_string(
            "Thumbnail size",
            &mut self.thumbnail,
            &["16 px", "24 px", "32 px", "64 px", "256 px"],
        );
    }

    fn includes(&self, asset: &GuiAssetChoice) -> bool {
        match self.usage {
            1 => asset.used,
            2 => !asset.used,
            _ => true,
        }
    }

    fn rows(
        &self,
        ui: &Ui,
        mut rows: Vec<&GuiAssetChoice>,
        selected: &mut Option<String>,
        show_library: bool,
    ) {
        rows.sort_by(|a, b| {
            match self.sort {
                1 => a.swatch.cmp(&b.swatch),
                2 => a.category.cmp(&b.category),
                _ => std::cmp::Ordering::Equal,
            }
            .then_with(|| a.name.cmp(&b.name))
            .then_with(|| a.id.cmp(&b.id))
        });
        if self.presentation == 0 {
            self.table(ui, &rows, selected, show_library);
        } else {
            for row in imgui::ListClipper::new(rows.len() as i32).begin(ui).iter() {
                self.asset_row(ui, rows[row as usize], selected);
            }
        }
    }

    fn table(
        &self,
        ui: &Ui,
        rows: &[&GuiAssetChoice],
        selected: &mut Option<String>,
        show_library: bool,
    ) {
        let Some(_table) = ui.begin_table("##asset-list", if show_library { 3 } else { 2 }) else {
            return;
        };
        let mut name = imgui::TableColumnSetup::new("Name");
        name.flags = imgui::TableColumnFlags::WIDTH_STRETCH;
        name.init_width_or_weight = 2.5;
        ui.table_setup_column_with(name);
        if show_library {
            ui.table_setup_column("Library");
        }
        ui.table_setup_column("Category");
        ui.table_headers_row();
        for index in imgui::ListClipper::new(rows.len() as i32).begin(ui).iter() {
            let asset = rows[index as usize];
            ui.table_next_row();
            ui.table_next_column();
            self.asset_row(ui, asset, selected);
            if show_library {
                ui.table_next_column();
                ui.text(if asset.library == Some(GuiAssetLibrary::Project) {
                    "Project"
                } else {
                    "Built-in"
                });
            }
            ui.table_next_column();
            ui.text(asset_category(asset));
        }
    }

    fn asset_row(&self, ui: &Ui, asset: &GuiAssetChoice, selected: &mut Option<String>) {
        let _id = ui.push_id(&asset.id);
        if self.presentation == 1
            && let Some([r, g, b]) = asset.swatch
        {
            let size = [16_f32, 24., 32., 64., 256.][self.thumbnail]
                .min(ui.content_region_avail()[0].max(16.));
            if ui
                .color_button_config(
                    "##swatch",
                    [r as f32 / 255., g as f32 / 255., b as f32 / 255., 1.],
                )
                .size([size, size])
                .build()
            {
                *selected = Some(asset.id.clone());
            }
        }
        if ui
            .selectable_config(&asset.name)
            .selected(selected.as_deref() == Some(&asset.id))
            .build()
        {
            *selected = Some(asset.id.clone());
        }
    }
}
