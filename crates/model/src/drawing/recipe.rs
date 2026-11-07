// SPDX-License-Identifier: GPL-2.0-only
use super::{DrawingAnnotationDefinition, DrawingBaseViewDefinition, spelling};
use oblikovati_api::types::{DraftingStandard, SheetOrientation, SheetSize, TitleBlockLocation};
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

fn default_sheets() -> Vec<DrawingSheetRecipe> {
    vec![DrawingSheetRecipe::a3_landscape("Sheet:1")]
}
