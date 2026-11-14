// SPDX-License-Identifier: GPL-2.0-only
use super::{DrawingError, DrawingSheetRecipe};
use crate::property::{DESIGN_TRACKING, DOCUMENT_SUMMARY, SUMMARY_INFORMATION};
/// A title-block field's reference to the model's properties. Resolved values are never recipe state.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct DrawingTitleProperty {
    pub label: &'static str,
    pub property_set: &'static str,
    pub property_name: &'static str,
}
// Go model/drawing/titleblock.go: the standard definition binds these properties in this order.
const DEFAULT_TITLE_FIELDS: &[DrawingTitleProperty] = &[
    DrawingTitleProperty {
        label: "Title",
        property_set: SUMMARY_INFORMATION,
        property_name: "Title",
    },
    DrawingTitleProperty {
        label: "Part Number",
        property_set: DESIGN_TRACKING,
        property_name: "Part Number",
    },
    DrawingTitleProperty {
        label: "Material",
        property_set: DESIGN_TRACKING,
        property_name: "Material",
    },
    DrawingTitleProperty {
        label: "Drawn By",
        property_set: SUMMARY_INFORMATION,
        property_name: "Author",
    },
    DrawingTitleProperty {
        label: "Revision",
        property_set: DESIGN_TRACKING,
        property_name: "Revision Number",
    },
    DrawingTitleProperty {
        label: "Company",
        property_set: DOCUMENT_SUMMARY,
        property_name: "Company",
    },
];
impl DrawingSheetRecipe {
    /// Read the standard border's left, right, top and bottom margins in millimeters.
    /// Example: `sheet.border_margins_mm() == Some([20.,10.,10.,10.])`.
    pub fn border_margins_mm(&self) -> Option<[f64; 4]> {
        // Go model/drawing/border.go: DefaultBorderDefinition owns these drawing-format dimensions.
        self.border.then_some([20., 10., 10., 10.])
    }
    /// Read the standard title block's property bindings, without copying stale model values.
    /// Example: `sheet.title_block_fields()?.unwrap()[0].label == "Title"`.
    pub fn title_block_fields(
        &self,
    ) -> Result<Option<&'static [DrawingTitleProperty]>, DrawingError> {
        match self.title_block.as_str() {
            "" => Ok(None),
            "Default" => Ok(Some(DEFAULT_TITLE_FIELDS)),
            name => Err(DrawingError(format!(
                "title block {name:?}: expected the Default definition or no title block"
            ))),
        }
    }
}
