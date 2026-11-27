// SPDX-License-Identifier: GPL-2.0-only
//! Public state and action types for Import (file picker, STL options, STEP dockable panel).
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_model::feature::{
    FeatureId,
    imported_model::{ImportMode, ImportedFileKind, StepImportOptions, StlImportOptions},
};

/// Inspected metadata for a selected STL or STEP source file.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiImportSourceSummary {
    pub source_path: String,
    pub source_name: String,
    pub file_kind: ImportedFileKind,
    pub solid_names: Vec<String>,
    pub bounding_box: [f64; 3],
    pub triangle_count: usize,
}

/// Authored Import options shown when reopening an existing imported reference or feature.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiImportEditSeed {
    pub source_path: String,
    pub source_name: String,
    pub file_kind: ImportedFileKind,
    pub import_mode: ImportMode,
    pub body_names: Vec<String>,
    pub link_suppressed: bool,
    pub link_broken: bool,
    pub stl_options: StlImportOptions,
    pub step_options: StepImportOptions,
}

/// Exact draft request sent from GUI to the desktop geometry owner.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiImportRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub source_path: String,
    pub source_name: String,
    pub file_kind: ImportedFileKind,
    pub import_mode: ImportMode,
    pub body_names: Vec<String>,
    pub stl_options: StlImportOptions,
    pub step_options: StepImportOptions,
    pub show_viewport_preview: bool,
}

/// Geometry owner's certification result for an Import draft.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiImportPreview {
    Ready,
    Failed(String),
}

/// Last certified or declined Import draft state.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiImportPreviewState {
    pub request: GuiImportRequest,
    pub result: GuiImportPreview,
}

/// Actions emitted by the Import ribbon button, dialogs, and browser context menu.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiImportAction {
    Open,
    OpenEdit(FeatureId),
    InspectSource {
        document: u64,
        path: String,
    },
    Preview(Box<GuiImportRequest>),
    Clear,
    Commit(Box<GuiImportRequest>),
    SetLinkSuppressed {
        document: u64,
        feature: FeatureId,
        suppressed: bool,
    },
    BreakLink {
        document: u64,
        feature: FeatureId,
    },
}

impl GuiImportAction {
    pub(crate) fn enabled(&self, state: &GuiCommandState) -> bool {
        let can_start = state.document.is_some()
            && state.certified_part_feature_available
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        let current = |request: &GuiImportRequest| {
            can_start
                && state.document == Some(request.document)
                && state.document_generation == Some(request.generation)
        };
        match self {
            Self::Open => can_start,
            Self::OpenEdit(feature) => {
                can_start
                    && state.browser.as_ref().is_some_and(|browser| {
                        browser
                            .import_edits
                            .get(feature)
                            .is_some_and(|seed| !seed.link_suppressed && !seed.link_broken)
                    })
            }
            Self::InspectSource { document, path } => {
                can_start && state.document == Some(*document) && !path.trim().is_empty()
            }
            Self::Preview(request) => current(request),
            Self::Clear => true,
            Self::Commit(request) => {
                current(request)
                    && !request.source_path.trim().is_empty()
                    && (request.file_kind == ImportedFileKind::Stl
                        || state.import_preview.as_ref().is_some_and(|preview| {
                            preview.request == **request
                                && preview.result == GuiImportPreview::Ready
                        }))
            }
            Self::SetLinkSuppressed {
                document, feature, ..
            }
            | Self::BreakLink { document, feature } => {
                can_start
                    && state.document == Some(*document)
                    && state.browser.as_ref().is_some_and(|browser| {
                        browser.import_edits.get(feature).is_some_and(|seed| {
                            seed.import_mode == ImportMode::ReferenceModel && !seed.link_broken
                        })
                    })
            }
        }
    }
}
