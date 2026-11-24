// SPDX-License-Identifier: GPL-2.0-only
use super::{
    GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits, TemplateChooser, TemplateFolder,
};

impl TemplateChooser {
    /// Open the chooser with a document kind selected.
    pub(crate) fn request_kind(&mut self, kind: GuiTemplateKind) {
        let units = match self.folder {
            TemplateFolder::Units(units) => units,
            TemplateFolder::Root => GuiTemplateUnits::default(),
        };
        self.go_to(TemplateFolder::Units(units));
        self.selected = Some(GuiDocumentTemplate { kind, units }.into());
        self.drawing_layout = None;
        self.request();
    }
}
