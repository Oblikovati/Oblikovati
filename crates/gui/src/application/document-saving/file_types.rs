// SPDX-License-Identifier: GPL-2.0-only
//! What a file dialog's type list offers, and which files each of its rows keeps.
use oblikovati_api::types::{
    DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING, DOCUMENT_PRESENTATION, DocumentType,
};
use std::path::Path;

pub(super) const ALL_FILES: &str = "All Files (*.*)";

/// One native file kind: the list's words and the extension without its dot.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct NativeFileType {
    pub(super) label: &'static str,
    pub extension: &'static str,
}
pub(super) const PART_FILES: NativeFileType = NativeFileType {
    label: "Part Files (*.opd)",
    extension: "opd",
};
pub(super) const ASSEMBLY_FILES: NativeFileType = NativeFileType {
    label: "Assembly Files (*.oad)",
    extension: "oad",
};
pub(super) const DRAWING_FILES: NativeFileType = NativeFileType {
    label: "Drawing Files (*.odd)",
    extension: "odd",
};
pub(super) const PRESENTATION_FILES: NativeFileType = NativeFileType {
    label: "Presentation Files (*.ord)",
    extension: "ord",
};
const NATIVE_FILE_TYPES: [NativeFileType; 4] = [
    PART_FILES,
    ASSEMBLY_FILES,
    DRAWING_FILES,
    PRESENTATION_FILES,
];
/// The file type of a document; an unknown kind falls back to the part file a new document is.
pub(crate) fn native_file_type(kind: Option<DocumentType>) -> NativeFileType {
    match kind {
        Some(DOCUMENT_ASSEMBLY) => ASSEMBLY_FILES,
        Some(DOCUMENT_DRAWING) => DRAWING_FILES,
        Some(DOCUMENT_PRESENTATION) => PRESENTATION_FILES,
        _ => PART_FILES,
    }
}
/// Which files a file type row keeps; folders are always listed.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum FileFilter {
    Everything,
    Extensions(&'static [&'static str]),
    Extension(&'static str),
}
impl FileFilter {
    pub(super) fn keeps(self, name: &str) -> bool {
        match self {
            Self::Everything => true,
            Self::Extension(extension) => has_extension(name, extension),
            Self::Extensions(extensions) => extensions
                .iter()
                .any(|extension| has_extension(name, extension)),
        }
    }
}
const ALL_DOCUMENTS: (&str, FileFilter) = (
    "All Documents (*.opd;*.oad;*.odd;*.ord)",
    FileFilter::Extensions(&["opd", "oad", "odd", "ord"]),
);
/// Exchange formats the list can show; opening one reports that it needs a translator.
const EXCHANGE_FILES: [(&str, FileFilter); 3] = [
    (
        "STEP Files (*.stp;*.step)",
        FileFilter::Extensions(&["stp", "step"]),
    ),
    (
        "IGES Files (*.igs;*.iges)",
        FileFilter::Extensions(&["igs", "iges"]),
    ),
    ("STL Files (*.stl)", FileFilter::Extension("stl")),
];
/// The rows a dialog's file type list offers, each with what it keeps.
pub(crate) struct FileTypeList {
    pub(super) rows: Vec<(&'static str, FileFilter)>,
}
impl FileTypeList {
    /// One document's own type and All Files; example: `FileTypeList::saving(PART_FILES)`.
    pub(crate) fn saving(native: NativeFileType) -> Self {
        Self {
            rows: vec![
                (native.label, FileFilter::Extension(native.extension)),
                (ALL_FILES, FileFilter::Everything),
            ],
        }
    }
    /// All documents, each native type, exchange formats and All Files; example: the Open dialog.
    pub(crate) fn opening() -> Self {
        let native =
            NATIVE_FILE_TYPES.map(|kind| (kind.label, FileFilter::Extension(kind.extension)));
        let mut rows = vec![ALL_DOCUMENTS];
        rows.extend(native);
        rows.extend(EXCHANGE_FILES);
        rows.push((ALL_FILES, FileFilter::Everything));
        Self { rows }
    }
    /// Exchange CAD and mesh files offered by the Import file picker.
    pub(crate) fn importing() -> Self {
        let mut rows = vec![(
            "CAD and Mesh Files (*.stp;*.step;*.stl;*.igs;*.iges)",
            FileFilter::Extensions(&["stp", "step", "stl", "igs", "iges"]),
        )];
        rows.extend(EXCHANGE_FILES);
        rows.push((ALL_FILES, FileFilter::Everything));
        Self { rows }
    }
    /// Only assemblies; example: inserting a model into a presentation.
    pub(crate) fn assemblies() -> Self {
        Self {
            rows: vec![(
                ASSEMBLY_FILES.label,
                FileFilter::Extension(ASSEMBLY_FILES.extension),
            )],
        }
    }
    /// All files, for choices that are not documents; example: a preview image.
    pub(crate) fn any() -> Self {
        Self {
            rows: vec![(ALL_FILES, FileFilter::Everything)],
        }
    }
    /// Image formats accepted by the appearance image chooser.
    pub(crate) fn appearance_images() -> Self {
        Self {
            rows: vec![(
                "Image Files (*.png;*.jpg;*.jpeg;*.bmp)",
                FileFilter::Extensions(&["png", "jpg", "jpeg", "bmp"]),
            )],
        }
    }
    /// What a row keeps; a row that is gone keeps everything.
    pub(crate) fn filter(&self, row: usize) -> FileFilter {
        self.rows
            .get(row)
            .map_or(FileFilter::Everything, |(_, filter)| *filter)
    }
    pub(super) fn labels(&self) -> Vec<&'static str> {
        self.rows.iter().map(|(label, _)| *label).collect()
    }
}
pub(super) fn has_extension(name: &str, extension: &str) -> bool {
    Path::new(name)
        .extension()
        .is_some_and(|found| found.eq_ignore_ascii_case(extension))
}
