// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiParameterSnapshot;
use std::collections::{BTreeMap, BTreeSet};
/// Text cells retain separate identities within a parameter row.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub enum GuiParameterTextField {
    Name,
    Equation,
    Comment,
}
/// A widget target uses model identity, independent of its display name or table position.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub struct GuiParameterTextTarget {
    pub document: u64,
    pub parameter: u64,
    pub field: GuiParameterTextField,
}
/// An owned edit leaves the GUI context and waits for one matching owner response.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiParameterTextEdit {
    pub target: GuiParameterTextTarget,
    pub request: u64,
    pub value: String,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiParameterAck {
    Applied,
    Failed,
    Obsolete,
}
struct ParameterTextDraft {
    original: String,
    edited: String,
    attempted: Option<String>,
    pending: Option<u64>,
    issue: Option<String>,
}
/// Bounded owner-thread widget state; the model remains in the session.
pub struct GuiParameterDrafts {
    document: Option<u64>,
    parameters: BTreeSet<u64>,
    cells: BTreeMap<GuiParameterTextTarget, ParameterTextDraft>,
    byte_limit: usize,
    cell_limit: usize,
    next_request: u64,
}
impl GuiParameterDrafts {
    /// Bound retained text and cells explicitly; example: `GuiParameterDrafts::new(4096, 128)?`.
    pub fn new(byte_limit: usize, cell_limit: usize) -> Result<Self, String> {
        if byte_limit == 0 || cell_limit == 0 {
            return Err(format!(
                "GUI parameter draft limits {byte_limit}/{cell_limit}: expected positive text and cell budgets"
            ));
        }
        Ok(Self {
            document: None,
            parameters: BTreeSet::new(),
            cells: BTreeMap::new(),
            byte_limit,
            cell_limit,
            next_request: 1,
        })
    }
    /// Retire cells for closed documents and deleted rows; example: `drafts.synchronize(Some(&snapshot))?`.
    pub fn synchronize(&mut self, snapshot: Option<&GuiParameterSnapshot>) -> Result<(), String> {
        if let Some(snapshot) = snapshot {
            snapshot.validate()?;
        }
        self.document = snapshot.map(|snapshot| snapshot.document);
        self.parameters = snapshot
            .into_iter()
            .flat_map(|snapshot| &snapshot.rows)
            .map(|row| row.parameter)
            .collect();
        self.cells.retain(|target, _| {
            Some(target.document) == self.document && self.parameters.contains(&target.parameter)
        });
        Ok(())
    }
    /// Seed a text cell once, then preserve its draft across model updates.
    /// Example: `ui.input_text("Equation", drafts.text_buffer(target, seed)?).build()`.
    pub fn text_buffer(
        &mut self,
        target: GuiParameterTextTarget,
        seed: &str,
    ) -> Result<&mut String, String> {
        self.require_target(target)?;
        self.admit_cell(target, seed)?;
        let draft = self.cells.get_mut(&target).expect("admitted text cell");
        if draft.pending.is_some() {
            return Err(format!(
                "GUI parameter cell {target:?}: expected an idle cell before further editing"
            ));
        }
        Ok(&mut draft.edited)
    }
    /// Read retained text while an edit is pending; example: `drafts.text(target)` for a disabled widget.
    pub fn text(&self, target: GuiParameterTextTarget) -> Option<&str> {
        self.cells.get(&target).map(|draft| draft.edited.as_str())
    }
    /// Inspect the owner decline without dropping authored text; example: `drafts.issue(target)`.
    pub fn issue(&self, target: GuiParameterTextTarget) -> Option<&str> {
        self.cells
            .get(&target)
            .and_then(|draft| draft.issue.as_deref())
    }
    /// Disable a submitted cell until its matching result arrives; example: `drafts.is_pending(target)`.
    pub fn is_pending(&self, target: GuiParameterTextTarget) -> bool {
        self.cells
            .get(&target)
            .is_some_and(|draft| draft.pending.is_some())
    }
    /// Admit one changed focus-loss edit and suppress duplicate or failed automatic submissions.
    /// Example: `if let Some(edit) = drafts.submit(target)? { actions.push(edit); }`.
    pub fn submit(
        &mut self,
        target: GuiParameterTextTarget,
    ) -> Result<Option<GuiParameterTextEdit>, String> {
        self.require_target(target)?;
        let draft = self
            .cells
            .get(&target)
            .ok_or_else(|| missing_cell(target))?;
        if draft.pending.is_some()
            || draft.edited == draft.original
            || draft.attempted.as_ref() == Some(&draft.edited)
        {
            return Ok(None);
        }
        require_draft_text(&draft.edited, self.byte_limit)?;
        let value = draft.edited.clone();
        let request = self.admit_request()?;
        let draft = self.cells.get_mut(&target).expect("admitted changed cell");
        draft.pending = Some(request);
        draft.attempted = Some(value.clone());
        draft.issue = None;
        Ok(Some(GuiParameterTextEdit {
            target,
            request,
            value,
        }))
    }
    /// Apply only a matching response; a delayed old response cannot clear another cell.
    /// Example: `drafts.acknowledge(&edit, owner_result)` after the session processes the edit.
    pub fn acknowledge(
        &mut self,
        edit: &GuiParameterTextEdit,
        result: Result<(), String>,
    ) -> GuiParameterAck {
        let Some(draft) = self.cells.get_mut(&edit.target) else {
            return GuiParameterAck::Obsolete;
        };
        if draft.pending != Some(edit.request) || draft.attempted.as_ref() != Some(&edit.value) {
            return GuiParameterAck::Obsolete;
        }
        if let Err(cause) = result {
            draft.pending = None;
            draft.issue = Some(cause);
            return GuiParameterAck::Failed;
        }
        self.cells.remove(&edit.target);
        GuiParameterAck::Applied
    }
    fn require_target(&self, target: GuiParameterTextTarget) -> Result<(), String> {
        if Some(target.document) != self.document || !self.parameters.contains(&target.parameter) {
            return Err(format!(
                "GUI parameter cell {target:?}: expected a current document and registered parameter"
            ));
        }
        Ok(())
    }
    fn admit_cell(&mut self, target: GuiParameterTextTarget, seed: &str) -> Result<(), String> {
        if let Some(draft) = self.cells.get_mut(&target) {
            rebase_idle_draft(draft, seed, self.byte_limit)?;
            return Ok(());
        }
        require_draft_text(seed, self.byte_limit)?;
        self.make_cell_room()?;
        self.cells.insert(
            target,
            ParameterTextDraft {
                original: seed.into(),
                edited: seed.into(),
                attempted: None,
                pending: None,
                issue: None,
            },
        );
        Ok(())
    }
    fn make_cell_room(&mut self) -> Result<(), String> {
        if self.cells.len() < self.cell_limit {
            return Ok(());
        }
        let removable = self.cells.iter().find_map(|(target, draft)| {
            (draft.pending.is_none() && draft.attempted.is_none() && draft.edited == draft.original)
                .then_some(*target)
        });
        if let Some(target) = removable {
            self.cells.remove(&target);
            return Ok(());
        }
        Err(format!(
            "GUI parameter retained cells {}: expected fewer than {} protected cells",
            self.cells.len(),
            self.cell_limit
        ))
    }
    fn admit_request(&mut self) -> Result<u64, String> {
        let request = self.next_request;
        self.next_request = request.checked_add(1).ok_or_else(|| {
            format!("GUI parameter request {request}: expected an unused request identity")
        })?;
        Ok(request)
    }
}
fn rebase_idle_draft(
    draft: &mut ParameterTextDraft,
    seed: &str,
    byte_limit: usize,
) -> Result<(), String> {
    if draft.pending.is_some() || draft.attempted.is_some() || draft.edited != draft.original {
        return Ok(());
    }
    require_draft_text(seed, byte_limit)?;
    draft.original = seed.into();
    draft.edited = seed.into();
    Ok(())
}
fn missing_cell(target: GuiParameterTextTarget) -> String {
    format!("GUI parameter cell {target:?}: expected an initialized text draft")
}
pub(crate) fn require_draft_text(text: &str, byte_limit: usize) -> Result<(), String> {
    if text.len() > byte_limit || text.contains('\0') {
        return Err(format!(
            "GUI parameter draft length {}, NUL={}: expected NUL-free text within {byte_limit} bytes",
            text.len(),
            text.contains('\0')
        ));
    }
    Ok(())
}
#[cfg(test)]
#[path = "../tests/parameters/drafts.rs"]
mod tests;
