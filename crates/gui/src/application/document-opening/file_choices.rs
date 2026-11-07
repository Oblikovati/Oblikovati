// SPDX-License-Identifier: GPL-2.0-only
//! How a selected file should open. A new selection starts these again.
use imgui::Ui;
use std::path::Path;

const LAST_ACTIVE: &str = "Last Active";
const PRIMARY: &str = "Primary";
const SKIP_UNRESOLVED: &str = "Skip all unresolved files";

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum FileKind {
    Part,
    Assembly,
    Drawing,
    Other,
}
impl FileKind {
    pub(super) fn of(path: &str) -> Self {
        match Path::new(path).extension().and_then(|kind| kind.to_str()) {
            Some("opd") => Self::Part,
            Some("oad") => Self::Assembly,
            Some("odd") => Self::Drawing,
            _ => Self::Other,
        }
    }
}
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub(super) struct FileChoices {
    express: bool,
    defer: bool,
    skip_unresolved: bool,
    design_view: usize,
}
impl FileChoices {
    /// This drawing should open with its updates deferred.
    pub(super) fn defers(&self) -> bool {
        self.defer
    }
    pub(super) fn panel(&mut self, ui: &Ui, kind: FileKind, views: &[String]) {
        match kind {
            FileKind::Part => self.part_rows(ui, views),
            FileKind::Assembly => self.assembly_rows(ui, views),
            FileKind::Drawing => self.drawing_rows(ui),
            FileKind::Other => {}
        }
    }
    fn part_rows(&mut self, ui: &Ui, views: &[String]) {
        ui.text(format!("Model State: {PRIMARY}"));
        self.design_view_row(ui, views);
    }
    fn assembly_rows(&mut self, ui: &Ui, views: &[String]) {
        ui.checkbox("Load in Express Mode", &mut self.express);
        ui.text(format!("Model State: {PRIMARY}"));
        let _express = ui.begin_disabled(self.express);
        self.design_view_row(ui, views);
        ui.text(format!("Position View: {PRIMARY}"));
    }
    /// Deferring leaves nothing to resolve, so it takes the choice back as well.
    fn drawing_rows(&mut self, ui: &Ui) {
        if ui.checkbox("Load in Defer Mode", &mut self.defer) && self.defer {
            self.skip_unresolved = false;
        }
        let _deferred = ui.begin_disabled(self.defer);
        ui.checkbox(SKIP_UNRESOLVED, &mut self.skip_unresolved);
    }
    fn design_view_row(&mut self, ui: &Ui, views: &[String]) {
        let mut rows = vec![LAST_ACTIVE, PRIMARY];
        rows.extend(views.iter().map(String::as_str));
        let shown = rows.get(self.design_view).copied().unwrap_or(LAST_ACTIVE);
        let label = format!("Design View: {shown}");
        if let Some(row) = crate::widget_abi::popup_choice(ui, &label, "##file-view", &rows) {
            self.design_view = row;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn a_files_kind_comes_from_its_extension() {
        assert_eq!(FileKind::of("/a/part.opd"), FileKind::Part);
        assert_eq!(FileKind::of("/a/set.oad"), FileKind::Assembly);
        assert_eq!(FileKind::of("/a/sheet.odd"), FileKind::Drawing);
        assert_eq!(FileKind::of("/a/notes.txt"), FileKind::Other);
    }
}
