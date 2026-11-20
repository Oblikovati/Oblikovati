// SPDX-License-Identifier: GPL-2.0-only
//! Enter the existing text editor from an unobstructed canvas activation.
use super::*;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn activate_existing_text_at(
        &mut self,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        if self.sketch_tool.is_some()
            || self.block_selection.is_some()
            || self.active_sketch_edit().is_none()
        {
            return Ok(false);
        }
        let (edit, generation) = self.capture_sketch_tool_context()?;
        let Some(entity) = self.text_activation_target(edit, pixel, window)? else {
            return Ok(false);
        };
        self.open_text_draft(edit, generation, entity)
    }

    fn text_activation_target(
        &self,
        edit: HeadSketchEdit,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<Option<SketchEntityId>, String> {
        let snapshot = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let sketch = snapshot
            .sketches
            .get(edit.sketch)
            .map_err(|cause| cause.to_string())?;
        if !sketch.visible()
            || sketch.geometry().texts().is_empty()
            || self.text_activation_blocked(edit, pixel, window)?
        {
            return Ok(None);
        }
        let scale = snapshot.parameters.units().working_scale();
        pick_existing_text(sketch.geometry(), edit.plane, pixel, scale, window)
    }

    fn text_activation_blocked(
        &self,
        edit: HeadSketchEdit,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        // Match the context menu's priority when a glyph or block covers the text.
        if window
            .hovered_constraint_glyphs()
            .is_some_and(|(sketch, _)| sketch == edit.sketch)
        {
            return Ok(true);
        }
        Ok(matches!(
            self.resolve_sketch_target(edit, pixel, window)?,
            Some(oblikovati_session::selection::SketchPickTarget::BlockInstance(_))
        ))
    }
}
