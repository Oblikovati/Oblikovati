// SPDX-License-Identifier: GPL-2.0-only
use crate::application::workspace_layout::browser_view::BrowserView;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{MouseButton, StyleColor, Ui};
use oblikovati_model::{document_sketch::DocumentSketchId, feature::FeatureId};

/// The browser tree a document kind shows; each kind names its own branches.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiBrowserKind {
    #[default]
    Part,
    SheetMetal,
    Assembly,
    Drawing,
    Presentation,
}
/// Immutable document structure; only the desktop owner can activate an edit.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiModelBrowser {
    pub name: String,
    pub kind: GuiBrowserKind,
    pub has_origin: bool,
    pub sketches: Vec<GuiBrowserSketch>,
    pub features: Vec<GuiBrowserFeature>,
    pub selected_sketches: Vec<DocumentSketchId>,
    /// The feature row a press chose.
    pub selected_feature: Option<FeatureId>,
    /// The part's solids, each made by a feature of the program.
    pub bodies: Vec<GuiBrowserBody>,
}
/// Stable sketch identity paired with its current canonical API inspection or local decline.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBrowserSketch {
    pub id: DocumentSketchId,
    pub inspection: Result<oblikovati_api::wire::SketchInfo, String>,
    pub name: String,
    pub visible: bool,
    pub consumed_by: Option<FeatureId>,
}
/// One solid in the part, named as the document names it; example: `Solid1` from `Extrude1`.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBrowserBody {
    pub name: String,
    pub feature: FeatureId,
}
/// One authored feature in model program order, with its actual suppression state.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBrowserFeature {
    pub id: FeatureId,
    pub name: String,
    pub suppressed: bool,
    /// What Show Extended Names adds to the row: its operation and depth.
    pub extended: Option<String>,
}
impl GuiModelBrowser {
    pub(crate) fn validate(&self) -> Result<(), String> {
        self.validate_labels()?;
        self.validate_identities()?;
        for id in &self.selected_sketches {
            if !self.sketches.iter().any(|row| row.id == *id) {
                return Err(format!(
                    "browser selected sketch {id:?}: expected a current sketch row"
                ));
            }
        }
        for row in &self.sketches {
            row.validate(&self.features)?;
        }
        Ok(())
    }
    fn validate_labels(&self) -> Result<(), String> {
        for text in std::iter::once(&self.name)
            .chain(self.sketches.iter().map(|row| &row.name))
            .chain(self.features.iter().map(|row| &row.name))
        {
            if text.contains('\0') {
                return Err(format!("browser label {text:?}: expected text without NUL"));
            }
        }
        Ok(())
    }
    fn validate_identities(&self) -> Result<(), String> {
        let sketches: std::collections::BTreeSet<_> =
            self.sketches.iter().map(|row| row.id.0).collect();
        let features: std::collections::BTreeSet<_> =
            self.features.iter().map(|row| row.id.0).collect();
        if sketches.len() != self.sketches.len()
            || features.len() != self.features.len()
            || sketches.contains(&0)
            || features.contains(&0)
        {
            return Err(format!(
                "browser identities sketches={sketches:?}, features={features:?}: expected unique nonzero IDs within each namespace"
            ));
        }
        Ok(())
    }
}
impl GuiBrowserSketch {
    fn validate(&self, features: &[GuiBrowserFeature]) -> Result<(), String> {
        self.validate_consumer(features)?;
        let tooltip = match &self.inspection {
            Ok(summary) => &summary.plane,
            Err(cause) => cause,
        };
        if tooltip.contains('\0') {
            return Err(format!(
                "browser sketch {:?} inspection {tooltip:?}: expected text without NUL",
                self.id
            ));
        }
        Ok(())
    }
    fn validate_consumer(&self, features: &[GuiBrowserFeature]) -> Result<(), String> {
        if self
            .consumed_by
            .is_some_and(|owner| !features.iter().any(|feature| feature.id == owner))
        {
            return Err(format!(
                "browser sketch {:?}, owner {:?}: expected an owner in the current feature program",
                self.id, self.consumed_by
            ));
        }
        Ok(())
    }
}
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
    view.header(ui, browser.kind == crate::GuiBrowserKind::Drawing);
    browser_origin::clear_on_empty_click(ui, state, browser, view, actions);
    let _document = ui.push_id(format!("document-{:?}", state.document));
    ui.text(&browser.name);
    if let Some(query) = view.query() {
        crate::application::workspace_layout::browser_view::search_results(ui, browser, query);
        return;
    }
    let tree = BrowserTree {
        ui,
        state,
        browser,
        expansion: view.take_expansion(),
    };
    tree.draw(view, actions);
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
        self.bodies(view, actions);
        self.origin(view, actions);
        self.parameters();
        self.sketches(actions);
        self.features(view, actions);
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
    fn drawing(&self, view: &mut BrowserView) {
        for (branch, (label, rows)) in DRAWING_TREE.iter().enumerate() {
            let Some(_node) = self.branch(label, true) else {
                continue;
            };
            for (row, name) in rows.iter().enumerate() {
                let selected = view.selected_row == Some((branch, row));
                let chosen = if *name == SHEET_TITLE_BLOCK {
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
        let node = self.ui.tree_node_config("Origin").default_open(true);
        let node = match (closed, opened, self.expansion) {
            (true, ..) => node.opened(false, imgui::Condition::Always),
            (false, true, _) => node.opened(true, imgui::Condition::Always),
            (false, false, Some(open)) => node.opened(open, imgui::Condition::Always),
            (false, false, None) => node,
        };
        let token = node.push();
        browser_origin::origin_branch_menu(self.ui, view);
        if let Some(_origin) = token {
            for (row, label) in ORIGIN_ROWS.into_iter().enumerate() {
                browser_origin::origin_row(self.ui, view, row, label, actions);
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
    fn sketches(&self, actions: &mut Vec<GuiShellAction>) {
        if let Some(_sketches) = self.branch("Sketches", true) {
            for row in self
                .browser
                .sketches
                .iter()
                .filter(|row| row.consumed_by.is_none())
            {
                sketch_row(self.ui, self.state, row, actions);
            }
        }
    }
    fn features(&self, view: &mut BrowserView, actions: &mut Vec<GuiShellAction>) {
        if let Some(_features) = self.branch("Features", true) {
            for feature in &self.browser.features {
                self.feature(feature, self.expansion, view, actions);
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
        browser_feature::bodies_menu(self.ui, view);
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
        let node = self.ui.tree_node_config(&body.name);
        let node = match expansion {
            Some(open) => node.opened(open, imgui::Condition::Always),
            None => node,
        };
        let Some(_row) = node.push() else {
            return;
        };
        let made_by = self
            .browser
            .features
            .iter()
            .find(|feature| feature.id == body.feature);
        if let Some(feature) = made_by {
            self.feature(feature, expansion, view, actions);
        }
    }
    fn feature(
        &self,
        feature: &GuiBrowserFeature,
        expansion: Option<bool>,
        view: &mut BrowserView,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let _feature = self.ui.push_id(format!("feature-{}", feature.id.0));
        let token = browser_feature::feature_row(
            self.ui,
            self.state,
            self.browser,
            feature,
            expansion,
            view,
            actions,
        );
        if let Some(_children) = token {
            for row in self
                .browser
                .sketches
                .iter()
                .filter(|row| row.consumed_by == Some(feature.id))
            {
                sketch_row(self.ui, self.state, row, actions);
            }
        }
    }
}
const SHEET_TITLE_BLOCK: &str = "Title Block";
/// What a title block holds, one level under its row on the sheet.
const TITLE_BLOCK_FIELD: &str = "Field Text";
/// A drawing's browser tree: the built-in template's one sheet and the resources it draws from.
pub(crate) const DRAWING_TREE: [(&str, &[&str]); 2] = [
    (
        "Drawing Resources",
        &["Sheet Formats", "Default Border", "Default Title Block"],
    ),
    ("Sheet:1", &["Default Border", "Title Block"]),
];
pub(crate) const ORIGIN_ROWS: [&str; 7] = [
    "Center Point",
    "X Axis",
    "Y Axis",
    "Z Axis",
    "XY Plane",
    "XZ Plane",
    "YZ Plane",
];
pub(crate) fn model_browser_width(viewport_width: f32) -> f32 {
    // Keep the native view centre available for picks even when content scale narrows the layout (#3850).
    300_f32.min((viewport_width * 0.5 - 16.).max(1.))
}
