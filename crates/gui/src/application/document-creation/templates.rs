// SPDX-License-Identifier: GPL-2.0-only
//! Document template descriptors shared by the chooser and desktop owner.

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiTemplateKind {
    #[default]
    Part,
    SheetMetal,
    Assembly,
    Drawing,
    Presentation,
}
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiTemplateUnits {
    #[default]
    Metric,
    Inch,
}
impl GuiTemplateUnits {
    /// The template folder that holds these units; example: `Metric`.
    pub(crate) fn folder(self) -> &'static str {
        match self {
            Self::Metric => "Metric",
            Self::Inch => "English",
        }
    }
    fn word(self) -> &'static str {
        match self {
            Self::Metric => "metric",
            Self::Inch => "inch",
        }
    }
}
/// A folder of the project template catalog, independent of a file's stored units.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiTemplateFolder {
    Root,
    Units(GuiTemplateUnits),
}
impl Default for GuiTemplateFolder {
    fn default() -> Self {
        Self::Units(GuiTemplateUnits::default())
    }
}
impl GuiTemplateFolder {
    pub(super) const ALL: [Self; 3] = [
        Self::Root,
        Self::Units(GuiTemplateUnits::Metric),
        Self::Units(GuiTemplateUnits::Inch),
    ];
    pub(super) fn label(self) -> &'static str {
        match self {
            Self::Root => "Templates",
            Self::Units(units) => units.folder(),
        }
    }
}
/// One chooser tile; example: `GuiDocumentTemplate { kind: GuiTemplateKind::Part, units: GuiTemplateUnits::Metric }`.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct GuiDocumentTemplate {
    pub kind: GuiTemplateKind,
    pub units: GuiTemplateUnits,
}

/// Metadata for one file offered by the active project's chooser catalog.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiProjectTemplate {
    pub path: std::path::PathBuf,
    pub title: String,
    pub folder: GuiTemplateFolder,
    pub kind: GuiTemplateKind,
    pub length_unit: &'static str,
    pub size: u64,
    pub material: Option<GuiTemplateMaterial>,
}
/// Physical material described by a template recipe.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiTemplateMaterial {
    /// The recipe does not assign a material.
    Unassigned,
    /// The assigned identity resolved to this display name.
    Assigned { id: String, name: String },
    /// The recipe assigns an identity that cannot be resolved.
    Unresolved { id: String },
    /// The source needed to inspect the assigned material failed.
    Unavailable { cause: String },
}
/// One template and its owner-resolved material.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiTemplateMaterialEntry {
    pub template: GuiDocumentTemplate,
    pub material: GuiTemplateMaterial,
}
impl GuiDocumentTemplate {
    /// The canonical length unit; example: a metric template answers `mm`.
    pub fn length_unit(self) -> &'static str {
        match self.units {
            GuiTemplateUnits::Metric => "mm",
            GuiTemplateUnits::Inch => "in",
        }
    }
    /// The wire document type; example: a sheet-metal template creates a `part`.
    pub fn type_name(self) -> &'static str {
        match self.kind {
            GuiTemplateKind::Part | GuiTemplateKind::SheetMetal => "part",
            GuiTemplateKind::Assembly => "assembly",
            GuiTemplateKind::Drawing => "drawing",
            GuiTemplateKind::Presentation => "presentation",
        }
    }
    /// The registered flavor, empty for a plain document.
    pub fn sub_type(self) -> &'static str {
        match self.kind {
            GuiTemplateKind::SheetMetal => SHEET_METAL_SUB_TYPE,
            _ => "",
        }
    }
    /// The word that begins a new document's name; example: `Part` becomes `Part1`.
    pub fn name_prefix(self) -> &'static str {
        match self.kind {
            GuiTemplateKind::Part | GuiTemplateKind::SheetMetal => "Part",
            GuiTemplateKind::Assembly => "Assembly",
            GuiTemplateKind::Drawing => "Drawing",
            GuiTemplateKind::Presentation => "Presentation",
        }
    }
    pub(crate) fn title(self) -> String {
        let units = self.units.word();
        match self.kind {
            GuiTemplateKind::Part => format!("Standard {units} part"),
            GuiTemplateKind::SheetMetal => format!("{} sheet-metal part", capitalized(units)),
            GuiTemplateKind::Assembly => format!("Standard {units} assembly"),
            GuiTemplateKind::Drawing => format!("{} drawing with title block", capitalized(units)),
            GuiTemplateKind::Presentation => format!("Standard {units} presentation"),
        }
    }
    pub(crate) fn type_label(self) -> &'static str {
        self.name_prefix()
    }
    pub(crate) fn unit_label(self) -> &'static str {
        match self.units {
            GuiTemplateUnits::Metric => "Length: millimeters (mm)",
            GuiTemplateUnits::Inch => "Length: inches (in)",
        }
    }
}
/// The API constant is a `Cow`, so a `&'static str` needs its own spelling; a test proves they agree.
const SHEET_METAL_SUB_TYPE: &str = "org.oblikovati.part.sheetMetal";
fn capitalized(word: &str) -> String {
    let mut letters = word.chars();
    letters.next().map_or_else(String::new, |first| {
        first.to_uppercase().chain(letters).collect()
    })
}
/// The gallery's groups in display order, each with its tiles.
pub(crate) const TEMPLATE_GROUPS: [(&str, &[GuiTemplateKind]); 4] = [
    (
        "Part",
        &[GuiTemplateKind::Part, GuiTemplateKind::SheetMetal],
    ),
    ("Assembly", &[GuiTemplateKind::Assembly]),
    ("Drawing", &[GuiTemplateKind::Drawing]),
    ("Presentation", &[GuiTemplateKind::Presentation]),
];
#[cfg(test)]
#[path = "../../../tests/application/document-creation/template_kinds.rs"]
mod tests;
