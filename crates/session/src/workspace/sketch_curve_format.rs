// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchCurveFormat, SketchCurveFormatChange, SketchError},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Convert this owner's applicable picks without changing focus or keeping another selection.
    /// Example: `commands.toggle_editor_curve_format(sketch, SketchCurveFormat::Construction)?;`.
    pub fn toggle_editor_curve_format(
        &mut self,
        sketch: DocumentSketchId,
        format: SketchCurveFormat,
    ) -> Result<SketchCurveFormatChange, CommandError<SketchError>> {
        self.session.toggle_editor_curve_format(sketch, format)
    }
}
