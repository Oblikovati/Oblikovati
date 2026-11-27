// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{sketch_dimension::SketchDimensionState, sketch_tool::SketchToolState};
use oblikovati_gui::GuiDimensionDraft;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn dimension_state(&self) -> Option<&SketchDimensionState> {
        match self.sketch_tool.as_ref()? {
            SketchToolState::Dimension(tool) => Some(tool),
            _ => None,
        }
    }
    pub(super) fn gui_dimension_draft(&self) -> Result<Option<GuiDimensionDraft>, String> {
        if let Some(draft) = self.pending_placement_dimension()? {
            return Ok(Some(draft));
        }
        let Some(tool) = self.dimension_state() else {
            return Ok(None);
        };
        if tool.placed.is_none()
            || !self.sketch_context_valid(tool.edit.document, tool.edit.sketch, tool.generation)
        {
            return Ok(None);
        }
        let request = tool
            .proposal
            .as_ref()
            .ok_or_else(|| "dimension proposal None: expected a placed request".to_string())?;
        Ok(Some(GuiDimensionDraft {
            id: tool.serial,
            kind: request.kind.clone(),
            orientation: request.orientation.clone(),
            expression: request.expression.clone(),
            committed: tool.committed.clone(),
            refusal: tool.refusal.clone(),
            choice_required: tool.choice_required,
        }))
    }
    fn pending_placement_dimension(&self) -> Result<Option<GuiDimensionDraft>, String> {
        use crate::placement_input::PendingDimensionPlacement;
        let Some(tool) = self
            .placement_state()
            .filter(|tool| self.placement_context_valid(tool))
        else {
            return Ok(None);
        };
        let Some(choice) = &tool.pending_dimension else {
            return Ok(None);
        };
        let lock = match &choice.placement {
            PendingDimensionPlacement::Recipe { locks, .. } => locks.get(choice.index),
            PendingDimensionPlacement::Line { input, .. } => input.locks.get(choice.index),
        }
        .and_then(Option::as_ref)
        .ok_or_else(|| {
            format!(
                "dimension choice field {}: expected a locked field",
                choice.index
            )
        })?;
        Ok(Some(GuiDimensionDraft {
            id: choice.id,
            kind: "distance".into(),
            orientation: String::new(),
            expression: lock.expression.clone(),
            committed: None,
            refusal: Some("This dimension would over-constrain the sketch.".into()),
            choice_required: true,
        }))
    }
    pub(super) fn dimension_preview_annotation(
        &self,
    ) -> Result<Option<crate::HeadDimensionAnnotation>, String> {
        let Some(tool) = self.dimension_state() else {
            return Ok(None);
        };
        if tool.committed.is_some()
            || !self.sketch_context_valid(tool.edit.document, tool.edit.sketch, tool.generation)
            || !self.dimension_ready(tool)?
        {
            return Ok(None);
        }
        let Some(position) = tool.placed.or(tool.cursor) else {
            return Ok(None);
        };
        let request = self.dimension_request_at(tool, position)?;
        Ok(Some(crate::HeadDimensionAnnotation {
            plane: tool.edit.plane,
            position,
            line: None,
            witnesses: None,
            unit: None,
            text: format!("{} = {}", request.kind, request.expression),
            edit: None,
            role: oblikovati_gui::GuiDimensionLabelRole::Preview,
        }))
    }
    pub(super) fn dimension_request(
        &self,
        tool: &SketchDimensionState,
    ) -> Result<oblikovati_api::wire::AddDimensionArgs, String> {
        let position = tool.placed.ok_or_else(|| {
            "dimension placement None: expected an explicit label position".to_string()
        })?;
        self.dimension_request_at(tool, position)
    }
    fn dimension_request_at(
        &self,
        tool: &SketchDimensionState,
        position: [f64; 2],
    ) -> Result<oblikovati_api::wire::AddDimensionArgs, String> {
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|cause| cause.to_string())?;
        let snapshot = session.snapshot();
        let geometry = snapshot
            .sketches
            .get(tool.edit.sketch)
            .map_err(|cause| cause.to_string())?
            .geometry();
        let index =
            crate::sketch_geometric::wire_sketch_index(&snapshot.sketches, tool.edit.sketch)?;
        tool.request_at(
            geometry,
            &session.selected_sketch_picks(tool.edit.sketch),
            index,
            snapshot.parameters.units(),
            position,
        )
    }
    pub(super) fn checked_dimension_draft(&self, id: u64) -> Result<&SketchDimensionState, String> {
        let tool = self
            .dimension_state()
            .filter(|tool| tool.serial == id && tool.placed.is_some())
            .ok_or_else(|| {
                format!("dimension draft {id}: expected the current placed owner draft")
            })?;
        if !self.sketch_context_valid(tool.edit.document, tool.edit.sketch, tool.generation) {
            return Err(format!(
                "dimension draft {id} generation {}: expected the current active sketch generation",
                tool.generation
            ));
        }
        Ok(tool)
    }
    pub(super) fn validate_dimension_source(
        &self,
        edit: crate::HeadSketchEdit,
        request: &oblikovati_api::wire::AddDimensionArgs,
    ) -> Result<(), String> {
        let input = crate::sketch_dimension::dimension_measurement_input(
            &request.kind,
            request.entities.as_deref().unwrap_or_default(),
            &request.orientation,
        )?;
        let snapshot = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        snapshot
            .require_converged_sketch(edit.sketch)
            .map_err(|cause| cause.to_string())?;
        snapshot
            .parameters
            .evaluate_scalar_expression(&request.expression, input.dimension())
            .map_err(|cause| {
                format!(
                    "dimension expression {:?}: expected a current valid {:?} value: {cause}",
                    request.expression,
                    input.dimension()
                )
            })?;
        Ok(())
    }
    pub(super) fn dimension_choice_required(
        &self,
        tool: &SketchDimensionState,
        request: &oblikovati_api::wire::AddDimensionArgs,
    ) -> Result<bool, String> {
        use oblikovati_model::history::{
            EditError, SketchDimensionCreation, SketchDimensionCreationError,
        };
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|e| e.to_string())?;
        if session
            .snapshot()
            .effective_sketch_settings()
            .over_constrained_behavior
            != oblikovati_api::types::OVER_CONSTRAINED_PROMPT
        {
            return Ok(false);
        }
        let operands = crate::sketch_dimension::dimension_measurement_input(
            &request.kind,
            request.entities.as_deref().unwrap_or_default(),
            &request.orientation,
        )?;
        let input = SketchDimensionCreation {
            operands,
            expression: request.expression.clone(),
            reference: request.driven,
            text_point: tool.placed,
        };
        Ok(matches!(
            session.preview_sketch_dimension(tool.edit.sketch, &input),
            Err(EditError::Mutation(
                SketchDimensionCreationError::ChoiceRequired
            ))
        ))
    }
    pub(super) fn dimension_ready(&self, tool: &SketchDimensionState) -> Result<bool, String> {
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|cause| cause.to_string())?;
        let geometry = session
            .snapshot()
            .sketches
            .get(tool.edit.sketch)
            .map_err(|cause| cause.to_string())?
            .geometry();
        tool.ready(geometry, &session.selected_sketch_picks(tool.edit.sketch))
    }
}
