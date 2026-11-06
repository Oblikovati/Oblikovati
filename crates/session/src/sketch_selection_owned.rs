// SPDX-License-Identifier: GPL-2.0-only
use crate::selection::{SelectionClickMode, SelectionSet, SketchPick, SketchPickTarget};
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchEntityDeletion, SketchError},
};
struct EditorDeletionBaseline {
    generation: u64,
    sketches: oblikovati_model::document_sketch::DocumentSketches,
    parameters: oblikovati_model::parameter::ParameterCollection<u64>,
}
impl EditorDeletionBaseline {
    fn capture(snapshot: &oblikovati_model::history::DocumentSnapshot, generation: u64) -> Self {
        Self {
            generation,
            sketches: snapshot.sketches.clone(),
            parameters: snapshot.parameters.clone(),
        }
    }
    fn is_verified_rollback(
        &self,
        snapshot: &oblikovati_model::history::DocumentSnapshot,
        generation: u64,
    ) -> bool {
        generation != self.generation
            && snapshot.sketches.same_recipe(&self.sketches)
            && snapshot.parameters.same_recipe(&self.parameters)
    }
}
impl<O: TransactionObserver> DocumentSession<O> {
    /// Read current-generation editor picks; stale handles are never published.
    /// Example: `session.selected_sketch_picks(sketch)` for rendering or constraint tools.
    pub fn selected_sketch_picks(&self, sketch: DocumentSketchId) -> Vec<SketchPick> {
        self.selected_editor_picks()
            .into_iter()
            .filter(|pick| pick.sketch() == sketch)
            .collect()
    }
    /// Read all live editor picks from this document for shared API and scene queries.
    /// Example: `session.selected_editor_picks()` without relying on a GUI cache.
    pub fn selected_editor_picks(&self) -> Vec<SketchPick> {
        self.editor_selection
            .items()
            .iter()
            .copied()
            .filter(|pick| {
                pick.validate_owner(self.document, self.generation(), pick.sketch())
                    .is_ok()
                    && self.sketch_pick(pick.sketch(), pick.target()).is_ok()
            })
            .collect()
    }

    /// Retire all transient document picks without reading geometry or changing history.
    /// Example: `session.clear_transient_selection()` on tool cancellation.
    pub fn clear_transient_selection(&mut self) -> bool {
        let changed =
            !self.browser_selection.items().is_empty() || !self.editor_selection.items().is_empty();
        self.browser_selection.clear();
        self.editor_selection.clear();
        changed
    }

    /// Delete the session-owned editor selection through the checked command path.
    /// Example: `session.delete_editor_selection(Some(sketch), false)?;`.
    pub fn delete_editor_selection(
        &mut self,
        sketch: Option<DocumentSketchId>,
        tool_active: bool,
    ) -> Result<SketchEntityDeletion, CommandError<SketchError>> {
        let before = EditorDeletionBaseline::capture(self.snapshot(), self.generation());
        let mut candidate = self.editor_selection.clone();
        match self.delete_selected_sketch_picks(sketch, tool_active, &mut candidate) {
            Ok(report) => {
                self.editor_selection = candidate;
                Ok(report)
            }
            Err(error) => {
                self.restore_refused_editor_picks(&before, sketch, &candidate);
                Err(error)
            }
        }
    }
    fn restore_refused_editor_picks(
        &mut self,
        before: &EditorDeletionBaseline,
        sketch: Option<DocumentSketchId>,
        picks: &SelectionSet<SketchPick>,
    ) {
        let Some(sketch) = sketch else {
            return;
        };
        let original_scope = picks
            .validate_scope(self.document, before.generation, sketch)
            .is_ok();
        if !original_scope || !before.is_verified_rollback(self.snapshot(), self.generation()) {
            return;
        }
        // #3841: a verified veto rollback changes generation, not selected identities.
        self.editor_selection = restamped_editor_picks(picks, self.document, self.generation());
    }
    /// Store a checked editor click in the document owner without creating history.
    /// Example: `session.set_sketch_selection(sketch, Some(target), mode, false)?;`.
    pub fn set_sketch_selection(
        &mut self,
        sketch: DocumentSketchId,
        target: Option<SketchPickTarget>,
        mode: SelectionClickMode,
        tool_active: bool,
    ) -> Result<Option<usize>, SketchError> {
        if tool_active {
            return Ok(None);
        }
        let picked = self.checked_editor_click(sketch, target, mode)?;
        let retire_browser = (picked.is_some() || mode == SelectionClickMode::Replace)
            && !self.browser_selection.items().is_empty();
        let count = self.editor_selection.apply_click(picked, mode, false);
        if retire_browser {
            self.browser_selection.clear();
            return Ok(Some(self.editor_selection.items().len()));
        }
        Ok(count)
    }
    fn checked_editor_click(
        &self,
        sketch: DocumentSketchId,
        target: Option<SketchPickTarget>,
        mode: SelectionClickMode,
    ) -> Result<Option<SketchPick>, SketchError> {
        self.snapshot().sketches.get(sketch)?;
        let picked = target
            .map(|target| self.sketch_pick(sketch, target))
            .transpose()?;
        if picked.is_some() && mode == SelectionClickMode::Toggle {
            self.editor_selection
                .validate_scope(self.document, self.generation(), sketch)?;
        }
        Ok(picked)
    }
}

fn restamped_editor_picks(
    picks: &SelectionSet<SketchPick>,
    document: u64,
    generation: u64,
) -> SelectionSet<SketchPick> {
    let mut restored = SelectionSet::default();
    for pick in picks.items() {
        restored.add(SketchPick::checked(
            document,
            generation,
            pick.sketch(),
            pick.target(),
        ));
    }
    // A narrowed filter controls future picks, not existing membership (#3841).
    restored.set_filter(picks.filter().clone());
    restored
}

#[cfg(test)]
#[path = "../tests/lifecycle/owned_selection_restamp_unit.rs"]
mod tests;
