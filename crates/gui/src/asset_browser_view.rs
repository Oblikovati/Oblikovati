// SPDX-License-Identifier: GPL-2.0-only
//! Independent document and library browsing for both Part asset windows.
use crate::{GuiAssetChoice, GuiAssetLibrary};
use imgui::Ui;

mod thumbnail;
pub(crate) use thumbnail::paint_preview as paint_appearance_preview;
const THUMBNAIL_SIZES: [f32; 5] = [16., 24., 32., 64., 256.];

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
    appearance: bool,
    usage: usize,
    presentation: usize,
    sort: usize,
    thumbnail: usize,
}

struct AssetRows<'a> {
    selected: &'a mut Option<String>,
    library_assignment_enabled: Option<bool>,
    assignment: Option<String>,
}

impl AssetBrowserView {
    pub(crate) fn appearances() -> Self {
        let display = || DisplayOptions {
            appearance: true,
            presentation: 1,
            thumbnail: 3,
            ..Default::default()
        };
        Self {
            document: display(),
            library: display(),
            ..Default::default()
        }
    }

    pub(crate) fn draw(
        &mut self,
        ui: &Ui,
        choices: &[GuiAssetChoice],
        selected: &mut Option<String>,
        library_assignment_enabled: Option<bool>,
    ) -> Option<String> {
        let mut selection = AssetRows {
            selected,
            library_assignment_enabled: None,
            assignment: None,
        };
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
            &mut selection,
            height,
            &query,
        );
        let library: Vec<_> = choices
            .iter()
            .filter(|row| row.library.is_some())
            .filter(matches)
            .collect();
        selection.library_assignment_enabled = library_assignment_enabled;
        self.library_region(ui, &library, &mut selection, !query.is_empty());
        selection.assignment
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
        selection: &mut AssetRows<'_>,
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
                    self.document.rows(ui, rows, selection, false);
                }
            });
    }

    fn library_region(
        &mut self,
        ui: &Ui,
        rows: &[&GuiAssetChoice],
        selection: &mut AssetRows<'_>,
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
        let footer_rows = if self.library.appearance { 3. } else { 2. };
        let height =
            (ui.content_region_avail()[1] - ui.frame_height_with_spacing() * footer_rows).max(64.);
        let width = ui.content_region_avail()[0];
        let stack_results = searching && width < 310.;
        let (tree_width, tree_height) = if stack_results {
            (
                0.,
                (height / 3.).min(6. * ui.text_line_height_with_spacing()),
            )
        } else {
            ((width * 0.32).clamp(80., 155.), height)
        };
        ui.child_window("##asset-categories")
            .size([tree_width, tree_height])
            .build(|| self.category_tree(ui, &rows));
        if !stack_results {
            ui.same_line();
        }
        let list_height = if stack_results {
            height - tree_height - ui.clone_style().item_spacing[1]
        } else {
            height
        };
        ui.child_window("##library-assets")
            .size([0., list_height])
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
                    self.library.rows(ui, rows, selection, searching);
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
            if self.appearance {
                &["Name", "Type", "Material Color", "Category"]
            } else {
                &["Name", "Material Color", "Category"]
            },
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
        selection: &mut AssetRows<'_>,
        show_library: bool,
    ) {
        rows.sort_by(|a, b| {
            match (self.appearance, self.sort) {
                (false, 1) | (true, 2) => a.swatch.cmp(&b.swatch),
                (false, 2) | (true, 3) => a.category.cmp(&b.category),
                (true, 1) => a
                    .appearance_preview
                    .is_some()
                    .cmp(&b.appearance_preview.is_some()),
                _ => std::cmp::Ordering::Equal,
            }
            .then_with(|| a.name.cmp(&b.name))
            .then_with(|| a.id.cmp(&b.id))
        });
        if self.presentation == 0 {
            self.table(ui, &rows, selection, show_library);
        } else if self.appearance && self.presentation == 1 {
            thumbnail::grid(ui, &rows, selection, THUMBNAIL_SIZES[self.thumbnail]);
        } else {
            for row in imgui::ListClipper::new(rows.len() as i32).begin(ui).iter() {
                self.asset_row(ui, rows[row as usize], selection);
            }
        }
    }

    fn table(
        &self,
        ui: &Ui,
        rows: &[&GuiAssetChoice],
        selection: &mut AssetRows<'_>,
        show_library: bool,
    ) {
        let columns = 2 + usize::from(show_library) + usize::from(self.appearance);
        let Some(_table) = ui.begin_table("##asset-list", columns) else {
            return;
        };
        let mut name = imgui::TableColumnSetup::new("Name");
        name.flags = imgui::TableColumnFlags::WIDTH_STRETCH;
        ui.table_setup_column_with(name);
        if show_library {
            asset_column(ui, "Library", "Built-in");
        }
        if self.appearance {
            asset_column(ui, "Type", "Generic");
        }
        asset_column(
            ui,
            "Category",
            if self.appearance {
                "Uncategorized"
            } else {
                "Category"
            },
        );
        ui.table_headers_row();
        for index in imgui::ListClipper::new(rows.len() as i32).begin(ui).iter() {
            let asset = rows[index as usize];
            ui.table_next_row();
            ui.table_next_column();
            self.asset_row(ui, asset, selection);
            if show_library {
                ui.table_next_column();
                ui.text(if asset.library == Some(GuiAssetLibrary::Project) {
                    "Project"
                } else {
                    "Built-in"
                });
            }
            if self.appearance {
                ui.table_next_column();
                ui.text("Generic");
            }
            ui.table_next_column();
            ui.text(asset_category(asset));
        }
    }

    fn asset_row(&self, ui: &Ui, asset: &GuiAssetChoice, selection: &mut AssetRows<'_>) {
        let _id = ui.push_id(&asset.id);
        if self.presentation == 1
            && let Some([r, g, b]) = asset.swatch
        {
            let size = THUMBNAIL_SIZES[self.thumbnail].min(ui.content_region_avail()[0].max(16.));
            if ui
                .color_button_config(
                    "##swatch",
                    [r as f32 / 255., g as f32 / 255., b as f32 / 255., 1.],
                )
                .size([size, size])
                .build()
            {
                *selection.selected = Some(asset.id.clone());
            }
        }
        if ui
            .selectable_config(&asset.name)
            .selected(selection.selected.as_deref() == Some(&asset.id))
            .build()
        {
            *selection.selected = Some(asset.id.clone());
        }
        selection.library_context_menu(ui, asset);
    }
}

fn asset_column(ui: &Ui, heading: &str, sample: &str) {
    let mut column = imgui::TableColumnSetup::new(heading);
    column.flags = imgui::TableColumnFlags::WIDTH_FIXED;
    column.init_width_or_weight =
        ui.calc_text_size(heading)[0].max(ui.calc_text_size(sample)[0]) + 4.;
    ui.table_setup_column_with(column);
}

impl AssetRows<'_> {
    fn library_context_menu(&mut self, ui: &Ui, asset: &GuiAssetChoice) {
        let Some(can_assign) = self.library_assignment_enabled else {
            return;
        };
        let Some(_popup) = ui.begin_popup_context_item() else {
            return;
        };
        if ui
            .menu_item_config("Assign to Selection")
            .enabled(can_assign)
            .build()
        {
            *self.selected = Some(asset.id.clone());
            self.assignment = Some(asset.id.clone());
        }
        for name in ["Rename", "Delete"] {
            ui.menu_item_config(name).enabled(false).build();
        }
        if asset.appearance_preview.is_some() {
            ui.menu_item_config("Where Used").enabled(false).build();
        }
        ui.menu_item_config("Add to").enabled(false).build();
    }
}
