// SPDX-License-Identifier: GPL-2.0-only
//! Independent document and library browsing for both Part asset windows.
use crate::{GuiAssetChoice, GuiAssetLibrary};
use imgui::Ui;

mod thumbnail;
pub(crate) use thumbnail::{
    PreviewLight, paint_cube_preview, paint_preview as paint_appearance_preview,
};
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
        let doc_rows: Vec<_> = choices
            .iter()
            .filter(|row| row.document)
            .filter(matches)
            .filter(|row| self.document.includes(row))
            .collect();
        let height = if doc_rows.is_empty() {
            34.
        } else {
            (ui.content_region_avail()[1] * 0.28).clamp(56., 130.)
        };
        self.document_region(ui, doc_rows, &mut selection, height, &query);
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
        use imgui::{StyleColor, StyleVar};
        let width = ui.content_region_avail()[0].max(60.);
        let height = ui.frame_height();
        let clear_width = if self.query.is_empty() { 0. } else { 22. };
        let [left, top] = ui.cursor_screen_pos();
        let end = [left + width, top + height];
        ui.get_window_draw_list()
            .add_rect([left, top], end, ui.style_color(StyleColor::FrameBg))
            .rounding(4.)
            .filled(true)
            .build();
        let _bw = ui.push_style_var(StyleVar::FrameBorderSize(0.));
        let field_bg = ui.push_style_color(StyleColor::FrameBg, [0.; 4]);
        ui.set_next_item_width((width - clear_width).max(32.));
        if ui
            .input_text("##asset-search", &mut self.query)
            .hint("Search assets")
            .build()
        {
            self.category = None;
        }
        drop(field_bg);
        let focused = ui.is_item_active();
        if !self.query.is_empty() {
            let _bg = ui.push_style_color(StyleColor::Button, [0.; 4]);
            let _ink = ui.push_style_color(StyleColor::Text, crate::theme::text_muted(ui));
            ui.same_line_with_spacing(0., 0.);
            if ui.button_with_size("×##asset-search", [clear_width, height]) {
                self.query.clear();
                self.category = None;
            }
            if ui.is_item_hovered() {
                ui.tooltip_text("Clear search");
            }
        }
        let outline = if focused {
            StyleColor::NavHighlight
        } else {
            StyleColor::Border
        };
        ui.get_window_draw_list()
            .add_rect([left, top], end, ui.style_color(outline))
            .rounding(4.)
            .build();
    }

    fn document_region(
        &mut self,
        ui: &Ui,
        rows: Vec<&GuiAssetChoice>,
        selection: &mut AssetRows<'_>,
        height: f32,
        query: &str,
    ) {
        {
            let _semi = crate::ui_scale::ui_semibold_font(ui);
            ui.text("Document assets");
        }
        let view_w = ui.calc_text_size("View")[0] + 14.;
        let right = ui.window_content_region_max()[0];
        ui.same_line_with_pos((right - view_w).max(ui.cursor_pos()[0] + 6.));
        self.document
            .menu(ui, "View##doc-view", "Document view", true);
        ui.child_window("##document-assets")
            .size([0., height])
            .build(|| {
                if rows.is_empty() {
                    ui.text_colored(
                        crate::theme::text_muted(ui),
                        if query.is_empty() {
                            "No document assets"
                        } else {
                            "No matching document assets"
                        },
                    );
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
        let avail = ui.content_region_avail()[0];
        let view_w = ui.calc_text_size("View")[0] + 14.;
        let selector_w = (avail - view_w - 6.).clamp(80., 170.);
        self.library_selector(ui, selector_w);
        ui.same_line_with_spacing(0., 6.);
        self.library
            .menu(ui, "View##lib-view", "Library view", false);
        let rows: Vec<_> = rows
            .iter()
            .copied()
            .filter(|row| self.source.is_none() || row.library == self.source)
            .collect();
        let footer_rows = if self.library.appearance { 3. } else { 2. };
        let height =
            (ui.content_region_avail()[1] - ui.frame_height_with_spacing() * footer_rows).max(64.);
        let width = ui.content_region_avail()[0];
        let stack_results = width < 260. || (searching && width < 310.);
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

    fn library_selector(&mut self, ui: &Ui, width: f32) {
        ui.set_next_item_width(width);
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
    fn menu(&mut self, ui: &Ui, label: &str, tooltip: &str, document: bool) {
        let _id = ui.push_id(label);
        if ui.small_button(label) {
            ui.open_popup("##display");
        }
        if ui.is_item_hovered() {
            ui.tooltip_text(tooltip);
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
                (true, 1) => a.appearance_type().cmp(b.appearance_type()),
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
            asset_column(ui, "Type", "Ceramic");
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
                ui.text(asset.appearance_type());
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
