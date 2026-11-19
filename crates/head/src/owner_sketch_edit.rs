// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::HeadSketchEdit;
use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_PART, DocumentType};
use oblikovati_model::{document_sketch::DocumentSketchId, history::DocumentSketchCreation};
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(crate) fn sketch_inspection_edit(
        &self,
    ) -> Option<oblikovati_automation::SketchInspectionEdit> {
        self.sketch_edit.as_ref().map(|(document, watch, focus)| {
            oblikovati_automation::SketchInspectionEdit::new(
                *document,
                watch.clone(),
                focus.clone(),
            )
        })
    }
    pub(crate) fn gui_authoring_context(
        &self,
        document: u64,
    ) -> Result<oblikovati_gui::GuiAuthoringContext, String> {
        use oblikovati_gui::GuiAuthoringContext;
        if self.active_sketch_edit().is_some() {
            return Ok(GuiAuthoringContext::Sketch);
        }
        let kind = self
            .files
            .document_type(document)
            .map_err(|cause| cause.to_string())?;
        Ok(if kind == DOCUMENT_PART {
            GuiAuthoringContext::Part
        } else {
            GuiAuthoringContext::OtherDocument
        })
    }
    /// Enter an explicitly selected sketch without editing its recipe; example: `owner.begin_sketch_edit(document, sketch)?`.
    pub fn begin_sketch_edit(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
    ) -> Result<(), String> {
        self.require_sketch_edit_entry(document)?;
        self.watch_sketch_edit(document, sketch)?;
        // A feature took this sketch out of view: opening it brings it back while it is open.
        self.remember_hidden_sketch(document, sketch);
        // What the last command reported is over; an open sketch at rest reads Ready.
        self.command_notice = Some(DesktopCommandNotice::Ready);
        Ok(())
    }
    /// The open sketch, if it was hidden when it was opened; example: for what the view shows.
    pub(super) fn sketch_opened_while_hidden(&self) -> Option<DocumentSketchId> {
        let open = self.active_sketch_edit()?.sketch;
        self.opened_hidden_sketch.filter(|sketch| *sketch == open)
    }
    /// Create one history step and select the new sketch; example: `owner.create_sketch_for_edit(document, &request)?`.
    pub fn create_sketch_for_edit(
        &mut self,
        document: u64,
        request: &DocumentSketchCreation,
    ) -> Result<DocumentSketchId, String> {
        self.create_sketch_edit_entry(document, request, false)
    }
    pub(super) fn start_sketch_on_face_line(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        let Some(face) = self
            .part_scenes
            .sketch_support_face(&mut self.files, document, line)?
        else {
            return Ok(false);
        };
        self.create_sketch_on_face(document, face)
    }
    pub(super) fn start_sketch_on_selected_face(
        &mut self,
        document: u64,
        key: &str,
    ) -> Result<bool, String> {
        let face = self.checked_selected_named_face(document, key)?;
        let face = oblikovati_model::feature::work_plane::WorkPlaneFaceReference::new(face.key)?;
        self.create_sketch_on_face(document, face)
    }
    fn create_sketch_on_face(
        &mut self,
        document: u64,
        face: oblikovati_model::feature::work_plane::WorkPlaneFaceReference,
    ) -> Result<bool, String> {
        self.create_automatic_sketch_for_edit(
            document,
            &DocumentSketchCreation {
                name: String::new(),
                placement: oblikovati_model::document_sketch::SketchPlacement::Face(face),
                orientation: None,
                project_origin: true,
            },
        )?;
        Ok(true)
    }
    /// Create and select a sketch with the next available default name.
    /// Example: `owner.create_automatic_sketch_for_edit(document, &request)?`.
    pub(super) fn create_automatic_sketch_for_edit(
        &mut self,
        document: u64,
        request: &DocumentSketchCreation,
    ) -> Result<DocumentSketchId, String> {
        self.create_sketch_edit_entry(document, request, true)
    }
    fn create_sketch_edit_entry(
        &mut self,
        document: u64,
        request: &DocumentSketchCreation,
        automatic: bool,
    ) -> Result<DocumentSketchId, String> {
        self.require_sketch_edit_entry(document)?;
        let mut request = request.clone();
        request.project_origin &= self.sketch_document_kind(document)? == DOCUMENT_PART;
        let part_datums = self.sketch_creation_datums(document, &request)?;
        let mut commands = self
            .files
            .commands(document)
            .map_err(|cause| cause.to_string())?;
        let sketch = match (automatic, part_datums.as_ref()) {
            (true, Some(datums)) => commands
                .create_sketch_with_part_datums(&request, datums, |result| result)
                .map_err(|error| error.to_string())?,
            (false, Some(datums)) => commands
                .create_named_sketch_with_part_datums(&request, datums, |result| result)
                .map_err(|error| error.to_string())?,
            (true, None) => commands
                .create_automatic_sketch(&request)
                .map_err(|error| error.to_string())?,
            (false, None) => commands
                .create_sketch(&request)
                .map_err(|error| error.to_string())?,
        };
        commands
            .tag_sketch_creation(sketch)
            .map_err(|cause| cause.to_string())?;
        self.watch_sketch_edit(document, sketch)?;
        Ok(sketch)
    }
    fn sketch_creation_datums(
        &mut self,
        document: u64,
        request: &DocumentSketchCreation,
    ) -> Result<Option<oblikovati_session::part_evaluation::CurrentPartDatums>, String> {
        use oblikovati_automation::PartUpdateAccess;
        if let oblikovati_model::document_sketch::SketchPlacement::Face(face) = &request.placement {
            return self
                .part_scenes
                .current_part_datums_for_face(&mut self.files, document, face)
                .map(Some);
        }
        let needs_placement = matches!(
            request.placement,
            oblikovati_model::document_sketch::SketchPlacement::WorkPlane(_)
        ) || request.orientation.is_some();
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?
            .snapshot();
        if !needs_placement || !oblikovati_automation::requires_part_replay(snapshot) {
            return Ok(None);
        }
        self.part_scenes
            .current_part_datums(&mut self.files, document)
            .map(Some)
    }
    /// Leave the edit and discard unfinished placement; committed commands keep their own undo entries.
    /// Example: `owner.finish_sketch_edit()?` before selecting another sketch.
    pub fn finish_sketch_edit(&mut self) -> Result<(), String> {
        self.synchronize_sketch_edit();
        let edit = self.active_sketch_edit().ok_or_else(|| {
            format!(
                "sketch edit {:?}: expected a live active sketch",
                self.sketch_edit
            )
        })?;
        self.require_edit_document(edit.document)?;
        self.finish_sketch_history_current(edit.document, edit.sketch)?;
        self.retire_sketch_tool()?;
        self.clear_sketch_isolation(edit.document)?;
        self.sketch_edit = None;
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt {
            message: "Sketch finished.".into(),
        });
        Ok(())
    }
    pub(super) fn clear_sketch_isolation(&mut self, document: u64) -> Result<(), String> {
        if self.files.workspace().session(document).is_err() {
            return Ok(());
        }
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .set_sketch_isolated(false)?;
        Ok(())
    }
    fn finish_sketch_history_current(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
    ) -> Result<(), String> {
        use oblikovati_automation::PartUpdateAccess;
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?
            .snapshot();
        let part_datums = if oblikovati_automation::requires_part_replay(snapshot) {
            Some(
                self.part_scenes
                    .current_part_datums(&mut self.files, document)?,
            )
        } else {
            None
        };
        let mut commands = self
            .files
            .commands(document)
            .map_err(|error| error.to_string())?;
        match part_datums {
            Some(datums) => commands.finish_sketch_history_with_datums(sketch, &datums),
            None => commands.finish_sketch_history(sketch),
        }
        .map_err(|error| error.to_string())
    }
    /// Re-resolve identity after edits and history restore; example: `owner.active_sketch_edit()` for command enablement.
    pub fn active_sketch_edit(&self) -> Option<HeadSketchEdit> {
        let current = self.current_sketch_edit();
        let document = self.files.workspace().active_document()?;
        let navigation = self
            .files
            .workspace()
            .session(document)
            .ok()?
            .pending_sketch_navigation();
        let Some(navigation) = navigation else {
            return current;
        };
        let target = navigation.target(
            current.map(|edit| edit.sketch),
            self.focused_sketch_identity(),
        );
        target.map_or(current, |target| {
            target.and_then(|sketch| self.resolve_edit(document, sketch).ok())
        })
    }
    fn current_sketch_edit(&self) -> Option<HeadSketchEdit> {
        let (document, watch, focus) = self.sketch_edit.as_ref()?;
        if !watch.is_live() || !self.files.workspace().retains_focus(focus) {
            return None;
        }
        let document = *document;
        let sketch = watch.sketch();
        if self.files.workspace().active_document() != Some(document) {
            return None;
        }
        self.resolve_edit(document, sketch).ok()
    }
    fn focused_sketch_identity(&self) -> Option<DocumentSketchId> {
        let (document, watch, focus) = self.sketch_edit.as_ref()?;
        (self.files.workspace().retains_focus(focus)
            && self.files.workspace().active_document() == Some(*document))
        .then(|| watch.sketch())
    }
    fn watch_sketch_edit(&mut self, document: u64, sketch: DocumentSketchId) -> Result<(), String> {
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?
            .snapshot();
        if oblikovati_automation::requires_part_replay(snapshot) {
            let resolver = self
                .part_scenes
                .sketch_continuity_resolver(&self.files, document)?;
            self.files
                .commands(document)
                .map_err(|error| error.to_string())?
                .set_sketch_continuity_resolver(Box::new(resolver));
        }
        let watch = self
            .files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .watch_sketch_continuity(sketch)
            .map_err(|cause| cause.to_string())?;
        self.sketch_edit = Some((document, watch, self.files.workspace().focus_identity()));
        Ok(())
    }
    pub(super) fn resolve_edit(
        &self,
        document: u64,
        sketch: DocumentSketchId,
    ) -> Result<HeadSketchEdit, String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let snapshot = session.snapshot();
        let plane = if oblikovati_automation::requires_part_replay(snapshot) {
            match self.part_scenes.current(session) {
                Ok(current) => snapshot
                    .resolve_sketch_plane_with_datums(sketch, &current.datum_evaluation())
                    .map_err(|error| error.to_string())?,
                Err(_) => self
                    .part_scenes
                    .sketch_continuity_resolver(&self.files, document)?
                    .plane(session.history(), sketch)?,
            }
        } else {
            snapshot
                .resolve_sketch_plane(sketch)
                .map_err(|cause| cause.to_string())?
        };
        Ok(HeadSketchEdit {
            document,
            sketch,
            plane,
            look_at_plane: self
                .part_store
                .as_ref()
                .is_none_or(|store| store.part_options().look_at_sketch_plane),
        })
    }
    pub(super) fn require_edit_document(&self, document: u64) -> Result<(), String> {
        if self.files.workspace().active_document() != Some(document) {
            return Err(format!(
                "sketch edit document {document}: expected active document {:?}",
                self.files.workspace().active_document()
            ));
        }
        self.sketch_document_kind(document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if session.in_transaction() {
            return Err(format!(
                "sketch edit document {document}: expected no open transaction"
            ));
        }
        Ok(())
    }
    fn sketch_document_kind(&self, document: u64) -> Result<DocumentType, String> {
        let kind = self
            .files
            .document_type(document)
            .map_err(|cause| cause.to_string())?;
        if !matches!(kind, DOCUMENT_PART | DOCUMENT_ASSEMBLY) {
            return Err(format!(
                "sketch edit document {document}, kind {kind:?}: expected a part or assembly"
            ));
        }
        Ok(kind)
    }
    fn require_sketch_edit_entry(&mut self, document: u64) -> Result<(), String> {
        self.synchronize_sketch_edit();
        self.require_edit_document(document)?;
        if let Some(edit) = &self.sketch_edit {
            return Err(format!(
                "sketch edit {edit:?}: expected the current edit to be finished first"
            ));
        }
        Ok(())
    }
    pub(super) fn synchronize_sketch_edit(&mut self) -> bool {
        if self.apply_sketch_history_navigation() {
            return true;
        }
        if self.sketch_edit.is_none() || self.active_sketch_edit().is_some() {
            return false;
        }
        let document = self.sketch_edit.as_ref().map(|(document, _, _)| *document);
        let retired = self.retire_sketch_tool().and_then(|()| {
            document.map_or(Ok(()), |document| self.clear_sketch_isolation(document))
        });
        self.sketch_edit = None;
        self.command_notice = Some(match retired {
            Ok(()) => DesktopCommandNotice::PlacementPrompt {
                message: "Sketch edit ended because its document or plane is unavailable.".into(),
            },
            Err(cause) => DesktopCommandNotice::AuthoringFailed { cause },
        });
        true
    }
    fn apply_sketch_history_navigation(&mut self) -> bool {
        let Some(document) = self.files.workspace().active_document() else {
            return false;
        };
        let navigation = self
            .files
            .commands(document)
            .ok()
            .and_then(|mut commands| commands.take_sketch_navigation());
        let Some(navigation) = navigation else {
            return false;
        };
        let current = self.current_sketch_edit().map(|edit| edit.sketch);
        let Some(target) = navigation.target(current, self.focused_sketch_identity()) else {
            return false;
        };
        if current == target {
            return false;
        }
        let result = self.restore_sketch_history_target(document, target);
        if let Err(cause) = result {
            self.command_notice = Some(DesktopCommandNotice::AuthoringFailed { cause });
        }
        true
    }
    fn restore_sketch_history_target(
        &mut self,
        document: u64,
        target: Option<DocumentSketchId>,
    ) -> Result<(), String> {
        self.retire_sketch_tool()?;
        self.clear_sketch_isolation(document)?;
        self.sketch_edit = None;
        self.opened_hidden_sketch = None;
        if let Some(sketch) = target {
            self.watch_sketch_edit(document, sketch)?;
            self.remember_hidden_sketch(document, sketch);
        }
        Ok(())
    }
    fn remember_hidden_sketch(&mut self, document: u64, sketch: DocumentSketchId) {
        self.opened_hidden_sketch = self
            .files
            .workspace()
            .session(document)
            .ok()
            .and_then(|session| session.snapshot().sketches.get(sketch).ok())
            .filter(|placed| !placed.visible())
            .map(|_| sketch);
    }
    pub(super) fn select_placement_sketch(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
    ) -> Result<(), String> {
        self.synchronize_sketch_edit();
        if self
            .active_sketch_edit()
            .is_some_and(|edit| (edit.document, edit.sketch) == (document, sketch))
        {
            return self.require_edit_document(document);
        }
        self.begin_sketch_edit(document, sketch)
    }
}
