// SPDX-License-Identifier: GPL-2.0-only
//! Auxiliary command enums and validation types re-exported from `commands`.

/// The current selection used to define an offset plane.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiOffsetPlaneBase {
    Origin(crate::GuiOriginPlaneKind),
    WorkPlane(oblikovati_model::feature::FeatureId),
    SelectedFace,
}

/// The owner-validated offset for one document generation.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiOffsetPlaneValidation {
    pub generation: u64,
    pub document: u64,
    pub offset: String,
    pub error: Option<String>,
}

/// The selection categories supported by the viewport's current pickers.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum GuiSelectionMode {
    #[default]
    All,
    Faces,
    Components,
    Custom,
}
