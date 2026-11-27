// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::{
    DIMENSION_INCH, DIMENSION_MILLIMETER, DRAFTING_ANSI, DRAFTING_ISO, DimensionUnit,
    DraftingStandard,
};

/// Dimensions and line weight for one built-in drafting standard.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct DrawingStandardPreset {
    pub standard: DraftingStandard,
    pub wire_name: &'static str,
    pub text_height_mm: f64,
    pub arrow_size_mm: f64,
    pub decimal_places: i64,
    pub dimension_unit: DimensionUnit,
    pub dimension_line_mm: f64,
    pub visible_line_mm: f64,
}

/// Built-in presets in the drawing API's display order.
pub const DRAWING_STANDARD_PRESETS: [DrawingStandardPreset; 2] = [
    DrawingStandardPreset {
        standard: DRAFTING_ISO,
        wire_name: "iso",
        text_height_mm: 3.5,
        arrow_size_mm: 3.5,
        decimal_places: 2,
        dimension_unit: DIMENSION_MILLIMETER,
        dimension_line_mm: 0.25,
        visible_line_mm: 0.5,
    },
    DrawingStandardPreset {
        standard: DRAFTING_ANSI,
        wire_name: "ansi",
        text_height_mm: 3.,
        arrow_size_mm: 3.,
        decimal_places: 3,
        dimension_unit: DIMENSION_INCH,
        dimension_line_mm: 0.3,
        visible_line_mm: 0.6,
    },
];
