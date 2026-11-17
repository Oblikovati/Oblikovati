// SPDX-License-Identifier: GPL-2.0-only
#[path = "browser_blocks.rs"]
mod browser_blocks;
#[path = "browser_model_state.rs"]
pub(crate) mod browser_model_state;
#[path = "browser_drawing_tree.rs"]
pub(crate) mod drawing_tree;
use crate::application::workspace_layout::browser_view::BrowserView;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{MouseButton, StyleColor, Ui};
use oblikovati_model::document_sketch::DocumentSketchId;

#[path = "browser_types.rs"]
mod browser_types;
pub use browser_types::*;
#[path = "browser_sketch_row.rs"]
mod sketch_rows;
use crate::application::workspace_layout::{
    browser_feature, browser_origin, browser_view::BrowserBranch,
};
use sketch_rows::sketch_row;
/// Draw the browser tree, or the rows a search leaves; example: inside the Model window.
pub(crate) fn browser_contents(
    ui: &Ui,
    state: &GuiCommandState,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(browser) = &state.browser else {
        ui.text("(no document)");
        return;
    };
    view.feature_rows.clear();
    view.feature_open.clear();
    view.header(ui, browser.kind == crate::GuiBrowserKind::Drawing);
    browser_origin::follow_keys(ui, view, actions, state);
    view.follow_relationships();
    let _document = ui.push_id(format!("document-{:?}", state.document));
    crate::application::model_browser::browser_root::root_row(ui, state, browser, view, actions);
    if let Some(query) = view.query() {
        crate::application::workspace_layout::browser_view::search_results(
            ui, state, browser, query,
        );
        view.feature_expansion = None;
        return;
    }
    let tree = BrowserTree {
        ui,
        state,
        browser,
        expansion: view.take_expansion(),
    };
    tree.draw(view, actions);
    // A removed or hidden branch cannot receive a pending keyboard expansion.
    view.feature_expansion = None;
    crate::application::model_browser::feature_keys::follow(ui, state, browser, view, actions);
}
/// One frame's tree; `expansion` is set for the single frame after Expand All or Collapse All.
struct BrowserTree<'a> {
    ui: &'a Ui,
    state: &'a GuiCommandState,
    browser: &'a GuiModelBrowser,
    expansion: Option<bool>,
}
impl BrowserTree<'_> {
    fn draw(&self, view: &mut BrowserView, actions: &mut Vec<GuiShellAction>) {
        match self.browser.kind {
            GuiBrowserKind::Part => self.modeling("End of Part", view, actions),
            GuiBrowserKind::SheetMetal => self.folded_model(view, actions),
            GuiBrowserKind::Assembly => self.assembly(view, actions),
            GuiBrowserKind::Drawing => self.drawing(view),
            GuiBrowserKind::Presentation => self.ui.text("Scene1"),
        }
    }
    fn branch(&self, label: &str, default_open: bool) -> Option<imgui::TreeNodeToken<'_>> {
        let node = self.ui.tree_node_config(label).default_open(default_open);
        match self.expansion {
            Some(open) => node.opened(open, imgui::Condition::Always).push(),
            None => node.push(),
        }
    }
    fn modeling(
        &self,
        end_marker: &str,
        view: &mut BrowserView,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.model_states(view, actions);
        self.bodies(view, actions);
        self.origin(view, actions);
        self.parameters();
        self.blocks(actions);
        self.sketches(view, actions);
        self.features(view, actions);
        if self.browser.kind == GuiBrowserKind::Assembly {
            self.occurrences(&self.browser.occurrences, actions);
        }
        // The marker closes a modelled document's history; a document without an origin has none.
        if self.browser.has_origin {
            self.ui.text(end_marker);
        }
    }
    fn folded_model(&self, view: &mut BrowserView, actions: &mut Vec<GuiShellAction>) {
        if let Some(_folded) = self.branch("Folded Model", true) {
            self.modeling("End of Folded", view, actions);
        }
    }
    fn assembly(&self, view: &mut BrowserView, actions: &mut Vec<GuiShellAction>) {
        self.ui.text("Representations");
        self.ui.text("Relationships");
        self.modeling("End of Assembly", view, actions);
    }
    fn occurrences(&self, rows: &[GuiBrowserOccurrence], actions: &mut Vec<GuiShellAction>) {
        for (index, row) in rows.iter().enumerate() {
            let _id = self.ui.push_id(format!("occurrence-{index}"));
            let _muted = row.suppressed.then(|| {
                self.ui.push_style_color(
                    StyleColor::Text,
                    self.ui.style_color(StyleColor::TextDisabled),
                )
            });
            if row.children.is_empty() {
                self.occurrence_label(row, actions);
            } else {
                self.occurrence_branch(row, actions);
            }
        }
    }
    fn occurrence_label(&self, row: &GuiBrowserOccurrence, actions: &mut Vec<GuiShellAction>) {
        let selected = self.browser.selected_occurrence.as_ref() == Some(&row.path);
        if self
            .ui
            .selectable_config(&row.name)
            .selected(selected)
            .build()
        {
            self.select_occurrence(row, actions);
        }
        self.occurrence_menu(row, actions);
        if row.deferred {
            self.ui.same_line();
            self.ui.text_disabled("[Express]");
        }
        if let Some(error) = &row.error {
            self.ui.same_line();
            self.ui.text_colored([1., 0.35, 0.25, 1.], "[unresolved]");
            if self.ui.is_item_hovered() {
                self.ui.tooltip_text(error);
            }
        }
    }
    fn occurrence_branch(&self, row: &GuiBrowserOccurrence, actions: &mut Vec<GuiShellAction>) {
        let selected = self.browser.selected_occurrence.as_ref() == Some(&row.path);
        let token = self
            .ui
            .tree_node_config(&row.name)
            .open_on_arrow(true)
            .default_open(true)
            .selected(selected)
            .push();
        if self.ui.is_item_clicked() && !self.ui.is_item_toggled_open() {
            self.select_occurrence(row, actions);
        }
        self.occurrence_menu(row, actions);
        if let Some(_node) = token {
            self.occurrences(&row.children, actions);
        }
    }
    fn select_occurrence(&self, row: &GuiBrowserOccurrence, actions: &mut Vec<GuiShellAction>) {
        actions.push(GuiShellAction::BrowserChoice {
            document: self.state.document.unwrap_or(0),
            choice: crate::GuiBrowserChoice::Occurrence(row.path.clone()),
        });
    }
    fn occurrence_menu(&self, row: &GuiBrowserOccurrence, actions: &mut Vec<GuiShellAction>) {
        let Some(_menu) = self.ui.begin_popup_context_item() else {
            return;
        };
        let label = if row.suppressed {
            "Unsuppress"
        } else {
            "Suppress"
        };
        if self
            .ui
            .menu_item_config(label)
            .enabled(row.path.len() == 1 && !self.state.transaction_open)
            .build()
        {
            actions.push(GuiShellAction::BrowserChoice {
                document: self.state.document.unwrap_or(0),
                choice: crate::GuiBrowserChoice::SetOccurrenceSuppressed {
                    path: row.path.clone(),
                    suppressed: !row.suppressed,
                },
            });
        }
    }
    fn drawing(&self, view: &mut BrowserView) {
        let drawing = self
            .state
            .document
            .and_then(|id| self.state.drawing_documents.get(&id));
        for (branch, (label, rows)) in drawing_tree::drawing_tree(drawing).iter().enumerate() {
            let Some(_node) = self.branch(label, true) else {
                continue;
            };
            for (row, name) in rows.iter().enumerate() {
                let selected = view.selected_row == Some((branch, row));
                let chosen = if name == SHEET_TITLE_BLOCK {
                    self.title_block_row(selected)
                } else {
                    self.ui.selectable_config(name).selected(selected).build()
                };
                if chosen {
                    view.selected_row = Some((branch, row));
                }
            }
        }
    }
    /// A selectable row that is also a branch: its arrow opens it, its text selects it.
    fn title_block_row(&self, selected: bool) -> bool {
        let node = self
            .ui
            .tree_node_config(SHEET_TITLE_BLOCK)
            .open_on_arrow(true)
            .default_open(true)
            .selected(selected);
        let token = match self.expansion {
            Some(open) => node.opened(open, imgui::Condition::Always).push(),
            None => node.push(),
        };
        let chosen = self.ui.is_item_clicked() && !self.ui.is_item_toggled_open();
        if token.is_some() {
            self.ui.text(TITLE_BLOCK_FIELD);
        }
        chosen
    }
    fn origin(&self, view: &mut BrowserView, actions: &mut Vec<GuiShellAction>) {
        if !self.browser.has_origin || view.hide_work_features {
            return;
        }
        // Brought back by Display Preferences, the branch starts closed.
        let closed = std::mem::take(&mut view.origin_closes_on_return);
        let opened = view.take_opened_branch(BrowserBranch::Origin);
        let keyed = view.take_origin_open();
        let node = self
            .ui
            .tree_node_config("Origin")
            .default_open(true)
            .selected(view.origin_folder_chosen);
        let node = match (keyed, closed, opened, self.expansion) {
            (Some(open), ..) => node.opened(open, imgui::Condition::Always),
            (None, true, ..) => node.opened(false, imgui::Condition::Always),
            (None, false, true, _) => node.opened(true, imgui::Condition::Always),
            (None, false, false, Some(open)) => node.opened(open, imgui::Condition::Always),
            (None, false, false, None) => node,
        };
        let token = node.push();
        view.origin_open = token.is_some();
        browser_origin::origin_branch_menu(self.ui, view);
        if let Some(_origin) = token {
            for (row, label) in ORIGIN_ROWS.into_iter().enumerate() {
                browser_origin::origin_row(self.ui, self.state, view, row, label, actions);
            }
        }
    }
    fn parameters(&self) {
        if let Some(_parameters) = self.branch("Parameters", false) {
            for row in &self.state.parameters {
                self.ui.bullet_text(format!("{} = {}", row.name, row.value));
            }
        }
    }
    fn sketches(&self, view: &mut BrowserView, actions: &mut Vec<GuiShellAction>) {
        if let Some(_sketches) = self.branch("Sketches", true) {
            for row in self
                .browser
                .sketches
                .iter()
                .filter(|row| row.consumed_by.is_none())
            {
                sketch_row(self.ui, self.state, row, view, actions);
            }
        }
    }
    fn features(&self, view: &mut BrowserView, actions: &mut Vec<GuiShellAction>) {
        if let Some(_features) = self.branch("Features", true) {
            for feature in &self.browser.features {
                self.feature(feature, self.expansion, view, actions, true);
            }
        }
    }
    /// The Solid Bodies folder: each solid, and under it the feature that made it.
    fn bodies(&self, view: &mut BrowserView, actions: &mut Vec<GuiShellAction>) {
        if self.browser.bodies.is_empty() {
            return;
        }
        let children = view.take_bodies_expansion();
        let node = self.ui.tree_node_config("Solid Bodies");
        // Collapse All Children folds what is under the folder and leaves the folder open.
        let node = match (children == Some(true), self.expansion) {
            (true, _) => node.opened(true, imgui::Condition::Always),
            (false, Some(open)) => node.opened(open, imgui::Condition::Always),
            (false, None) => node,
        };
        let token = node.push();
        browser_feature::bodies_menu(self.ui, self.state, self.browser, view, actions);
        let Some(_folder) = token else {
            return;
        };
        for body in &self.browser.bodies {
            self.body(body, children.or(self.expansion), view, actions);
        }
    }
    fn body(
        &self,
        body: &GuiBrowserBody,
        expansion: Option<bool>,
        view: &mut BrowserView,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let _body = self.ui.push_id(format!("body-{}", body.feature.0));
        let Some(_row) = browser_feature::body_row(
            self.ui,
            self.state,
            self.browser,
            body,
            expansion,
            view,
            actions,
        ) else {
            return;
        };
        let made_by = self
            .browser
            .features
            .iter()
            .find(|feature| feature.id == body.feature);
        if let Some(feature) = made_by {
            self.feature(feature, expansion, view, actions, false);
        }
    }
    fn feature(
        &self,
        feature: &GuiBrowserFeature,
        expansion: Option<bool>,
        view: &mut BrowserView,
        actions: &mut Vec<GuiShellAction>,
        record_rows: bool,
    ) {
        use crate::application::model_browser::feature_keys::FeatureRow;
        if record_rows {
            view.feature_rows.push(FeatureRow::Feature(feature.id));
        }
        let _feature = self.ui.push_id(format!("feature-{}", feature.id.0));
        let expansion = if record_rows {
            view.take_feature_expansion(feature.id).or(expansion)
        } else {
            expansion
        };
        let token = browser_feature::feature_row(
            self.ui,
            self.state,
            self.browser,
            feature,
            expansion,
            view,
            actions,
        );
        if record_rows {
            view.feature_open.insert(feature.id, token.is_some());
        }
        if let Some(_children) = token {
            for row in self
                .browser
                .sketches
                .iter()
                .filter(|row| row.consumed_by == Some(feature.id))
            {
                if record_rows {
                    view.feature_rows.push(FeatureRow::Sketch(row.id));
                }
                sketch_row(self.ui, self.state, row, view, actions);
            }
        }
    }
}
const SHEET_TITLE_BLOCK: &str = "Title Block";
/// What a title block holds, one level under its row on the sheet.
const TITLE_BLOCK_FIELD: &str = "Field Text";
pub(crate) const ORIGIN_ROWS: [&str; 7] = [
    "YZ Plane",
    "XZ Plane",
    "XY Plane",
    "X Axis",
    "Y Axis",
    "Z Axis",
    "Center Point",
];
