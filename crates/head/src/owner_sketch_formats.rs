// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchCurveFormat;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn toggle_gui_sketch_format(
        &mut self,
        format: SketchCurveFormat,
    ) -> Result<bool, String> {
        self.synchronize_sketch_edit();
        let edit = self
            .active_sketch_edit()
            .ok_or_else(|| format!("{format:?}: expected an active sketch edit"))?;
        self.require_format_context(edit)?;
        if self.placement_state().is_none() && self.convert_current_format_picks(edit, format)? {
            return Ok(true);
        }
        self.flip_creation_format(format)?;
        self.refresh_formatted_placement();
        Ok(true)
    }
    fn convert_current_format_picks(
        &mut self,
        edit: crate::HeadSketchEdit,
        format: SketchCurveFormat,
    ) -> Result<bool, String> {
        // #3845: history invalidates handles; an empty live selection arms a mode, never rebinds them.
        if !self.has_live_format_picks(edit)? {
            return Ok(false);
        }
        let report = self
            .files
            .commands(edit.document)
            .map_err(|cause| cause.to_string())?
            .toggle_editor_curve_format(edit.sketch, format)
            .map_err(|cause| cause.to_string())?;
        if report.applicable.is_empty() {
            return Ok(false);
        }
        self.placement_notice(format!("{} applied.", format.label()));
        Ok(true)
    }
    fn has_live_format_picks(&self, edit: crate::HeadSketchEdit) -> Result<bool, String> {
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?;
        Ok(!session.selected_sketch_picks(edit.sketch).is_empty())
    }
    pub(super) fn gui_sketch_format_available(&self) -> bool {
        if self.active_sketch_edit().is_none() {
            return false;
        }
        match self.placement_state() {
            Some(tool) => self.placement_context_valid(tool),
            None => self.sketch_tool.is_none(),
        }
    }
    fn require_format_context(&self, edit: crate::HeadSketchEdit) -> Result<(), String> {
        if !self.gui_sketch_format_available() {
            return Err(format!(
                "Format sketch {}: expected an idle editor or current placement tool",
                edit.sketch.0
            ));
        }
        if self
            .placement_state()
            .is_some_and(|tool| self.line_scope_context_valid(tool))
        {
            return Ok(());
        }
        self.require_edit_document(edit.document)
    }
    pub(super) fn require_format_tool(
        &self,
        tool: crate::SketchPlacementTool,
    ) -> Result<(), String> {
        if self.sketch_formats.construction && tool == crate::SketchPlacementTool::Point {
            return Err("Point with construction mode true: expected curve placement; construction points are unsupported".into());
        }
        Ok(())
    }
    fn flip_creation_format(&mut self, format: SketchCurveFormat) -> Result<(), String> {
        if format == SketchCurveFormat::Construction
            && !self.sketch_formats.construction
            && self
                .placement_state()
                .is_some_and(|tool| tool.tool == crate::SketchPlacementTool::Point)
        {
            return Err("Construction with active Point: expected a curve placement tool; construction points are unsupported".into());
        }
        match format {
            SketchCurveFormat::Construction => {
                self.sketch_formats.construction = !self.sketch_formats.construction
            }
            SketchCurveFormat::Centerline => {
                self.sketch_formats.centerline = !self.sketch_formats.centerline
            }
        }
        self.placement_notice(format!("{} creation mode changed.", format.label()));
        Ok(())
    }
    fn refresh_formatted_placement(&mut self) {
        let state = self.sketch_tool.take();
        match state {
            Some(crate::sketch_tool::SketchToolState::Placement(mut tool)) => {
                tool.formats = self.sketch_formats.clone();
                match tool.refresh_creation_formats(&self.sketch_construction) {
                    Ok(()) => self.refresh_placement_fields(&mut tool),
                    Err(cause) => {
                        self.command_notice = Some(DesktopCommandNotice::PlacementFailed { cause })
                    }
                }
                self.sketch_tool = Some(crate::sketch_tool::SketchToolState::Placement(tool));
            }
            other => self.sketch_tool = other,
        }
    }
}
