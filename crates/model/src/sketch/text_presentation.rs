// SPDX-License-Identifier: GPL-2.0-only
//! Keep evaluated Sketch Text out of document recipes.
use super::{Sketch2D, SketchEntityId, SketchError, SketchText2D, resolve_document_sketch_text};
use crate::parameter::ParameterCollection;
use oblikovati_api::types::SketchTextStyle;
use std::borrow::Cow;

/// A linked Text whose value could not be resolved for display.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchTextDisplayDecline {
    pub entity: SketchEntityId,
    pub cause: String,
}

impl SketchText2D {
    /// Read a transient display style. For example, resolve a file-name field with `"Part.opd"`.
    pub fn display_text_style(
        &self,
        parameters: &ParameterCollection<u64>,
        file_name: &str,
    ) -> Result<Cow<'_, SketchTextStyle>, SketchError> {
        resolve_document_sketch_text(&self.style, &self.parameter_bindings, parameters, file_name)
    }
}

impl Sketch2D {
    /// Copy the sketch for presentation and report failed fields by entity.
    /// For example, pass the resulting geometry to the viewport after a parameter edit.
    pub fn text_display_geometry(
        &self,
        parameters: &ParameterCollection<u64>,
        file_name: &str,
    ) -> (Self, Vec<SketchTextDisplayDecline>) {
        let mut presentation = self.clone();
        let mut declines = Vec::new();
        for text in &mut presentation.texts {
            if let Some(cause) = project_text(text, parameters, file_name) {
                declines.push(SketchTextDisplayDecline {
                    entity: text.id,
                    cause,
                });
            }
        }
        (presentation, declines)
    }
}

fn project_text(
    text: &mut SketchText2D,
    parameters: &ParameterCollection<u64>,
    file_name: &str,
) -> Option<String> {
    let failure = match text.display_text_style(parameters, file_name) {
        Ok(Cow::Borrowed(_)) => return None,
        Ok(Cow::Owned(display)) => {
            text.style = display;
            None
        }
        Err(error) => {
            // Keep the entity and its anchor in the packet, but submit no glyphs for the failed field.
            text.style.content.clear();
            text.style.linked_fields = None;
            text.style.formatting_runs = None;
            text.style.stacked_fractions = None;
            Some(error.to_string())
        }
    };
    text.parameter_bindings.clear();
    failure
}
