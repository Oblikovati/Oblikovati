// SPDX-License-Identifier: GPL-2.0-only
//! Owned drawing recipes; geometry projection and referenced-document resolution are injected by the host.
mod annotation;
mod border;
pub use border::border_zone_label;
mod document;
mod format;
pub use format::DrawingTitleProperty;
mod error;
mod note;
mod parts_list;
mod recipe;
mod sheet;
mod sheet_format;
pub use sheet_format::DrawingSheetFormat;
mod spelling;
mod standard;
pub use standard::{DRAWING_STANDARD_PRESETS, DrawingStandardPreset};
mod validation;
mod view;
pub use annotation::DrawingAnnotationDefinition;
pub use document::DrawingDocument;
pub use error::DrawingError;
pub use note::DrawingNotePresentation;
pub use parts_list::{
    DrawingPartsListPresentation, DrawingPartsListRow, DrawingPartsListSource, DrawingTableLabel,
    DrawingTableLine,
};
pub use recipe::{DrawingRecipe, DrawingSheetRecipe};
pub use view::{DrawingBaseViewDefinition, DrawingViewLabelEdit};
