// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{
        SketchLineChainCreation, SketchLineChainCreationReport, SketchPlacementCreationReport,
        SketchPlacementFieldLock,
    },
    sketch::SketchPlacementRecipe,
};
pub(super) enum PlacementFailure {
    Choice(usize),
    Other(String),
}
fn placement_failure(
    error: oblikovati_session::CommandError<
        oblikovati_model::history::SketchPlacementCreationError,
    >,
) -> PlacementFailure {
    use oblikovati_model::history::{
        EditError, SketchDimensionCreationError, SketchPlacementCreationError,
    };
    use oblikovati_session::{CommandError, MutationError};
    match error {
        CommandError::Edit(EditError::Mutation(MutationError::Command(
            SketchPlacementCreationError::Field { index, cause, .. },
        ))) if matches!(*cause, SketchDimensionCreationError::ChoiceRequired) => {
            PlacementFailure::Choice(index)
        }
        error => PlacementFailure::Other(error.to_string()),
    }
}
fn part_placement_failure(
    error: oblikovati_session::PartSketchCreateError<
        oblikovati_model::history::SketchPlacementCreationError,
    >,
) -> PlacementFailure {
    match error {
        oblikovati_session::PartSketchCreateError::Command(error) => placement_failure(error),
        error => PlacementFailure::Other(error.to_string()),
    }
}
fn line_failure(
    error: oblikovati_session::CommandError<oblikovati_model::history::SketchLineCreationError>,
) -> PlacementFailure {
    use oblikovati_model::history::{
        EditError, SketchDimensionCreationError, SketchLineCreationError,
    };
    use oblikovati_session::{CommandError, MutationError};
    match error {
        CommandError::Edit(EditError::Mutation(MutationError::Command(
            SketchLineCreationError::Dimension {
                index,
                cause: SketchDimensionCreationError::ChoiceRequired,
            },
        ))) => PlacementFailure::Choice(index),
        error => PlacementFailure::Other(error.to_string()),
    }
}
impl PlacementFailure {
    fn message(self) -> String {
        match self {
            Self::Choice(index) => {
                format!("placement field {index}: expected a driven dimension choice")
            }
            Self::Other(cause) => cause,
        }
    }
}
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn resolved_placement_preview(
        &mut self,
        tool: &crate::placement_input::SketchPlacementState,
    ) -> Result<Option<oblikovati_viewport::ViewportSketchPreview>, String> {
        if tool.tool == crate::SketchPlacementTool::Line {
            let Some(input) = tool.line_input() else {
                return Ok(None);
            };
            return match self.try_preview_sketch_line_chain(tool.document, tool.sketch, &input) {
                Ok(preview) => Ok(Some(preview)),
                Err(PlacementFailure::Choice(_)) => Ok(None),
                Err(PlacementFailure::Other(error)) => Err(error),
            };
        }
        if tool.tool == crate::SketchPlacementTool::Point {
            return self.resolved_point_preview(tool);
        }
        if !tool.fields_engaged() || !tool.at_finishing_stage() {
            return Ok(None);
        }
        // #3975: zero size is an unfinished shape, so it has no valid preview.
        if tool.typed_size_is_zero() {
            return Ok(None);
        }
        // The preview shows what Enter would commit: the shape at its typed sizes.
        let seeded = tool.typed_seed_recipe(&self.sketch_construction)?;
        let Some(recipe) = seeded.as_ref().or(tool.preview.as_ref()) else {
            return Ok(None);
        };
        let locks = tool.locks_for(recipe)?;
        let preview = self.preview_placement_candidate(tool.document, tool.sketch, recipe, &locks);
        match preview {
            Ok(preview) => Ok(Some(placement_preview_packet(tool.plane, preview))),
            Err(PlacementFailure::Choice(_)) => Ok(None),
            Err(PlacementFailure::Other(error)) => Err(format!(
                "placement preview {}: {error}",
                tool.field_prompt()
            )),
        }
    }
    /// Commit the displayed recipe through the document owner; example: `owner.commit_sketch_placement(document,sketch,&recipe,&locks)?;`.
    pub fn commit_sketch_placement(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
    ) -> Result<SketchPlacementCreationReport, String> {
        self.try_commit_sketch_placement(document, sketch, recipe, locks)
            .map_err(PlacementFailure::message)
    }
    pub(super) fn try_commit_sketch_placement(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
    ) -> Result<SketchPlacementCreationReport, PlacementFailure> {
        if self.needs_dimension_choice_preview(document, locks.iter().any(Option::is_some))? {
            self.preview_placement_candidate(document, sketch, recipe, locks)?;
        }
        self.commit_placement_candidate(document, sketch, recipe, locks)
    }
    fn needs_dimension_choice_preview(
        &self,
        document: u64,
        has_locks: bool,
    ) -> Result<bool, PlacementFailure> {
        if !has_locks {
            return Ok(false);
        }
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|e| PlacementFailure::Other(e.to_string()))?;
        Ok(session
            .snapshot()
            .effective_sketch_settings()
            .over_constrained_behavior
            == oblikovati_api::types::OVER_CONSTRAINED_PROMPT)
    }
    fn preview_placement_candidate(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
    ) -> Result<oblikovati_session::SketchPlacementPreview, PlacementFailure> {
        let datums = self
            .placement_part_datums(document)
            .map_err(PlacementFailure::Other)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|e| PlacementFailure::Other(e.to_string()))?;
        match datums.as_ref() {
            Some(datums) => session
                .preview_sketch_placement_with_datums(
                    sketch,
                    recipe,
                    locks,
                    &self.sketch_construction,
                    datums,
                )
                .map_err(part_placement_failure),
            None => session
                .preview_sketch_placement(sketch, recipe, locks, &self.sketch_construction)
                .map_err(placement_failure),
        }
    }
    fn commit_placement_candidate(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
    ) -> Result<SketchPlacementCreationReport, PlacementFailure> {
        let datums = self
            .placement_part_datums(document)
            .map_err(PlacementFailure::Other)?;
        let mut commands = self
            .files
            .commands(document)
            .map_err(|e| PlacementFailure::Other(e.to_string()))?;
        match datums.as_ref() {
            Some(datums) => commands
                .create_sketch_placement_with_datums(
                    sketch,
                    recipe,
                    locks,
                    &self.sketch_construction,
                    datums,
                )
                .map_err(part_placement_failure),
            None => commands
                .create_sketch_placement_with_construction(
                    sketch,
                    recipe,
                    locks,
                    &self.sketch_construction,
                )
                .map_err(placement_failure),
        }
    }
    fn placement_part_datums(
        &mut self,
        document: u64,
    ) -> Result<Option<oblikovati_session::part_evaluation::CurrentPartDatums>, String> {
        use oblikovati_automation::PartUpdateAccess;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if !oblikovati_automation::requires_part_replay(session.snapshot()) {
            return Ok(None);
        }
        self.part_scenes
            .current_part_datums(&mut self.files, document)
            .map(Some)
    }
    /// Prepare current-document inference without publishing geometry or reserving permanent IDs.
    /// Example: `window.set_sketch_preview(Some(&owner.preview_sketch_line_chain(document, sketch, &input)?))?;`.
    pub fn preview_sketch_line_chain(
        &self,
        document: u64,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
    ) -> Result<oblikovati_viewport::ViewportSketchPreview, String> {
        self.try_preview_sketch_line_chain(document, sketch, input)
            .map_err(PlacementFailure::message)
    }
    fn try_preview_sketch_line_chain(
        &self,
        document: u64,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
    ) -> Result<oblikovati_viewport::ViewportSketchPreview, PlacementFailure> {
        let plane = self
            .resolve_edit(document, sketch)
            .map_err(PlacementFailure::Other)?
            .plane;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| PlacementFailure::Other(cause.to_string()))?;
        let preview = session
            .preview_inferred_sketch_line_chain(sketch, input)
            .map_err(line_failure)?;
        Ok(oblikovati_viewport::ViewportSketchPreview::Resolved {
            plane,
            geometry: Box::new(preview.geometry),
            members: preview.members,
        })
    }
    /// Commit the same inferred chain operation used by the current-document preview.
    /// Example: `owner.commit_sketch_line_chain(document, sketch, &input)?;`.
    pub fn commit_sketch_line_chain(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
    ) -> Result<SketchLineChainCreationReport, String> {
        self.try_commit_sketch_line_chain(document, sketch, input)
            .map_err(PlacementFailure::message)
    }
    pub(super) fn try_commit_sketch_line_chain(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
    ) -> Result<SketchLineChainCreationReport, PlacementFailure> {
        if self.needs_dimension_choice_preview(document, input.locks.iter().any(Option::is_some))? {
            self.try_preview_sketch_line_chain(document, sketch, input)?;
        }
        self.files
            .commands(document)
            .map_err(|e| PlacementFailure::Other(e.to_string()))?
            .create_inferred_sketch_line_chain(sketch, input)
            .map_err(line_failure)
    }
}
pub(super) fn placement_preview_packet(
    plane: oblikovati_model::sketch::SketchPlane,
    preview: oblikovati_session::SketchPlacementPreview,
) -> oblikovati_viewport::ViewportSketchPreview {
    oblikovati_viewport::ViewportSketchPreview::Resolved {
        plane,
        geometry: Box::new(preview.geometry),
        members: preview.members,
    }
}
