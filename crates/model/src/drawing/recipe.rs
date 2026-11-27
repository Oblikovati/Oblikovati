// SPDX-License-Identifier: GPL-2.0-only
use super::{DrawingAnnotationDefinition, DrawingBaseViewDefinition, DrawingSheetFormat, spelling};
use oblikovati_api::types::{
    BorderLabelMode, DraftingStandard, SheetOrientation, SheetSize, TitleBlockLocation,
};
use serde::{Deserialize, Serialize};
/// Drawing content in the Go native recipe shape; no workspace pointers or kernel handles are stored.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct DrawingRecipe {
    #[serde(default)]
    pub model_reference: String,
    #[serde(default, with = "spelling::drafting_standard")]
    pub standard: DraftingStandard,
    #[serde(default)]
    pub active_sheet: usize,
    #[serde(default = "default_sheets")]
    pub sheets: Vec<DrawingSheetRecipe>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub sheet_formats: Vec<DrawingSheetFormat>,
}
/// One owned sheet definition; view definitions do not claim an evaluated projection.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct DrawingSheetRecipe {
    pub name: String,
    #[serde(with = "spelling::sheet_size")]
    pub size: SheetSize,
    #[serde(default, with = "spelling::sheet_orientation")]
    pub orientation: SheetOrientation,
    #[serde(default)]
    pub width_mm: f64,
    #[serde(default)]
    pub height_mm: f64,
    #[serde(default)]
    pub border: bool,
    #[serde(default, rename = "borderHZones", skip_serializing_if = "zero_zones")]
    pub border_h_zones: i64,
    #[serde(default, rename = "borderVZones", skip_serializing_if = "zero_zones")]
    pub border_v_zones: i64,
    #[serde(
        default,
        rename = "borderHLabel",
        with = "spelling::border_label",
        skip_serializing_if = "default_border_label"
    )]
    pub border_h_label: BorderLabelMode,
    #[serde(
        default,
        rename = "borderVLabel",
        with = "spelling::border_label",
        skip_serializing_if = "default_border_label"
    )]
    pub border_v_label: BorderLabelMode,
    #[serde(default)]
    pub title_block: String,
    #[serde(default, with = "spelling::title_block_location")]
    pub title_block_at: TitleBlockLocation,
    #[serde(default)]
    pub revision: String,
    #[serde(default)]
    pub views: Vec<DrawingBaseViewDefinition>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub annotations: Vec<DrawingAnnotationDefinition>,
}

fn zero_zones(count: &i64) -> bool {
    *count == 0
}

fn default_border_label(mode: &BorderLabelMode) -> bool {
    *mode == oblikovati_api::types::ALPHABETICAL_BORDER_LABEL
}

fn default_sheets() -> Vec<DrawingSheetRecipe> {
    vec![DrawingSheetRecipe::a3_landscape("Sheet:1")]
}
