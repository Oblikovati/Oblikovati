// SPDX-License-Identifier: GPL-2.0-only
//! Recent rows in the File menu, with a dedicated pin lane.
use crate::application::home_recent_documents::recent_rows::fit_recent_cell;
use crate::icons::{GuiRibbonIcons, IconGlyph};
use crate::{GuiCommandState, GuiShellAction};
use imgui::{StyleVar, Ui};

/// How the File menu orders and shows its recent files.
#[derive(Default)]
pub(super) struct RecentMenu {
    by_date: bool,
    with_locations: bool,
}
impl RecentMenu {
    pub(super) fn rows(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        icons: &GuiRibbonIcons,
    ) {
        self.controls(ui);
        if state.recent_documents.is_empty() {
            ui.text_disabled("No recent documents");
        }
        let mut rows: Vec<&crate::GuiRecentDocument> = state.recent_documents.iter().collect();
        if self.by_date {
            rows.sort_by_key(|row| std::cmp::Reverse(row.modified));
        }
        for row in rows.into_iter().take(RECENT_MENU_ROWS) {
            self.row(ui, row, actions, icons);
        }
    }
    fn controls(&mut self, ui: &Ui) {
        let ordering = if self.by_date {
            "By Access Date"
        } else {
            "By Ordered List"
        };
        if super::menu_art::current_value_button(ui, &format!("{ordering}###recent-ordering")) {
            self.by_date = !self.by_date;
        }
        ui.same_line();
        let display = if self.with_locations {
            "Name and Location"
        } else {
            "Name"
        };
        if super::menu_art::current_value_button(ui, &format!("{display}###recent-display")) {
            self.with_locations = !self.with_locations;
        }
    }
    fn row(
        &self,
        ui: &Ui,
        row: &crate::GuiRecentDocument,
        actions: &mut Vec<GuiShellAction>,
        icons: &GuiRibbonIcons,
    ) {
        let _row = ui.push_id(&row.path);
        let width = (ui.content_region_avail()[0] - PIN_GUTTER - PIN_WIDTH).max(44.);
        let measure = |text: &str| ui.calc_text_size(text)[0];
        let name = fit_recent_cell(&row.name, width - 54., measure);
        if document_button(ui, row, &name, width, icons) {
            actions.push(GuiShellAction::OpenDocument(
                oblikovati_api::wire::OpenDocumentArgs {
                    full_document_name: row.path.clone(),
                    ..Default::default()
                },
            ));
        }
        ui.same_line_with_spacing(0., PIN_GUTTER);
        let (label, glyph) = if row.pinned {
            ("Unpin", IconGlyph::RecentPinned)
        } else {
            ("Pin", IconGlyph::RecentPin)
        };
        if icons.navigation_button(ui, label, glyph, [PIN_WIDTH, ROW_HEIGHT]) {
            actions.push(GuiShellAction::Recent(crate::GuiRecentAction::SetPinned {
                path: row.path.clone(),
                pinned: !row.pinned,
            }));
        }
        if self.with_locations {
            ui.text_disabled(fit_recent_cell(&row.location, width - 12., measure));
        }
    }
}

const RECENT_MENU_ROWS: usize = 8;
const ROW_HEIGHT: f32 = 36.;
const PIN_WIDTH: f32 = 26.;
const PIN_GUTTER: f32 = 12.;

fn document_button(
    ui: &Ui,
    row: &crate::GuiRecentDocument,
    name: &str,
    width: f32,
    icons: &GuiRibbonIcons,
) -> bool {
    let _align = ui.push_style_var(StyleVar::ButtonTextAlign([0., 0.5]));
    let _padding = ui.push_style_var(StyleVar::FramePadding([28., 4.]));
    let clicked =
        crate::widget_abi::button(ui, &format!("{name}###recent-open"), [width, ROW_HEIGHT]);
    super::menu_art::paint_glyph(
        ui,
        icons,
        crate::icons::document_glyph(row.kind),
        ui.item_rect_min()[0] + 6.,
        1.,
    );
    if name != row.name && ui.is_item_hovered() {
        ui.tooltip_text(&row.name);
    }
    clicked
}
