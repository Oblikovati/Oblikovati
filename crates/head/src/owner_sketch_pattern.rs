// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::SketchRectangularPatternRequest;
use oblikovati_viewport::ViewportSketchPreview;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn preview_gui_pattern(
        &mut self,
        request: Option<SketchRectangularPatternRequest>,
    ) -> Result<bool, String> {
        let cleared = self.pattern_preview.take().is_some();
        let Some(request) = request else {
            return Ok(cleared);
        };
        let (edit, generation) = self.pattern_context()?;
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?;
        let preview = session
            .preview_pattern_from_expressions(edit.sketch, request)
            .map_err(|cause| cause.to_string())?;
        self.pattern_preview = Some((
            edit,
            generation,
            ViewportSketchPreview::Resolved {
                plane: edit.plane,
                geometry: Box::new(preview.geometry),
                members: preview.members,
            },
        ));
        Ok(true)
    }
    pub(super) fn create_gui_pattern(
        &mut self,
        request: SketchRectangularPatternRequest,
    ) -> Result<bool, String> {
        self.pattern_preview = None;
        let outcome = self.commit_gui_pattern(request);
        let serial = self
            .pattern_result
            .map_or(1, |(serial, _)| serial.saturating_add(1));
        self.pattern_result = Some((serial, outcome.is_ok()));
        outcome
    }
    fn commit_gui_pattern(
        &mut self,
        request: SketchRectangularPatternRequest,
    ) -> Result<bool, String> {
        let (edit, _) = self.pattern_context()?;
        self.files
            .commands(edit.document)
            .map_err(|cause| cause.to_string())?
            .run_projected(
                "Create rectangular sketch pattern",
                |snapshot, ids| snapshot.create_pattern_from_expressions(edit.sketch, request, ids),
                |_, pattern| Ok(pattern.id),
            )
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    fn pattern_context(&self) -> Result<(crate::HeadSketchEdit, u64), String> {
        let (edit, generation) = self.capture_sketch_tool_context()?;
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?;
        if self.sketch_tool.is_some() || session.in_transaction() {
            return Err(format!(
                "pattern in sketch {:?}: expected a ready sketch without an open transaction",
                edit.sketch
            ));
        }
        Ok((edit, generation))
    }
}
