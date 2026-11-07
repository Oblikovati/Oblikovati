// SPDX-License-Identifier: GPL-2.0-only
//! Owned drawing recipes; geometry projection and referenced-document resolution are injected by the host.
mod annotation;
mod document;
mod format;
pub use format::DrawingTitleProperty;
mod error;
mod recipe;
mod sheet;
mod spelling;
mod validation;
mod view;
pub use annotation::DrawingAnnotationDefinition;
pub use document::DrawingDocument;
pub use error::DrawingError;
pub use recipe::{DrawingRecipe, DrawingSheetRecipe};
pub use view::DrawingBaseViewDefinition;
