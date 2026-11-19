// SPDX-License-Identifier: GPL-2.0-only
//! Read and remove relations under the sketch context pointer.
use super::*;
use oblikovati_api::wire::DeleteConstraintArgs;
use oblikovati_session::selection::SelectionClickMode;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn gui_sketch_context(
        &self,
    ) -> Result<Option<oblikovati_gui::GuiSketchContext>, String> {
        let Some(target) = self.sketch_memory.context.as_ref() else {
            return Ok(None);
        };
        let Some(edit) = self.active_sketch_edit() else {
            return Ok(None);
        };
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|e| e.to_string())?;
        if target.document != edit.document
            || target.sketch != edit.sketch
            || target.generation != session.generation()
        {
            return Ok(None);
        }
        let snapshot = session.snapshot();
        let geometry = snapshot
            .sketches
            .get(edit.sketch)
            .map_err(|e| e.to_string())?
            .geometry();
        let point = target.point.filter(|point| geometry.point(*point).is_ok());
        let relations = if let Some(curve) = target.curve {
            let index =
                crate::sketch_geometric::wire_sketch_index(&snapshot.sketches, edit.sketch)?;
            let rows = oblikovati_api::client::Client::new(
                oblikovati_automation::FileQueryHost::new(&self.files),
            )
            .sketch()
            .constraints(&oblikovati_api::wire::SketchArgs {
                sketch_index: index,
            })
            .map_err(|e| e.to_string())?
            .constraints
            .unwrap_or_default();
            super::sketch_relation_context::related_relations(geometry, curve, &rows)?
        } else {
            Vec::new()
        };
        Ok(Some(oblikovati_gui::GuiSketchContext {
            token: target.token,
            point: point.map(|point| point.0),
            block: target.block,
            glyph: target
                .glyph
                .filter(|id| geometry.has_geometric_relation(*id)),
            glyph_candidates: target.glyph_candidates.clone(),
            center_point: point.is_some_and(|point| geometry.is_center_point(point)),
            candidates: target.candidates.clone(),
            relations,
        }))
    }
    pub(super) fn select_sketch_context_glyph(
        &mut self,
        token: u64,
        glyph: oblikovati_model::sketch::SketchConstraintId,
    ) -> Result<bool, String> {
        let edit = self.require_glyph_context(token, Some(glyph))?;
        self.store_sketch_target(
            edit,
            Some(oblikovati_session::selection::SketchPickTarget::GeometricConstraint(glyph)),
            SelectionClickMode::Replace,
        )?;
        self.publish_selection_count(edit)?;
        self.sketch_memory.context = None;
        Ok(true)
    }
    pub(super) fn hide_sketch_context_glyph(
        &mut self,
        token: u64,
        glyph: oblikovati_model::sketch::SketchConstraintId,
    ) -> Result<bool, String> {
        let edit = self.require_glyph_context(token, Some(glyph))?;
        let index = self.sketch_glyph_wire_index(edit)?;
        oblikovati_api::client::Client::new(self.document_host())
            .constrain()
            .set_glyph_visible(&oblikovati_api::wire::ConstraintGlyphVisibilityArgs {
                sketch_index: index,
                constraint_id: glyph.0,
                visible: false,
            })
            .map_err(|error| error.to_string())?;
        self.sketch_memory.context = None;
        Ok(true)
    }
    pub(super) fn set_sketch_context_glyphs(
        &mut self,
        token: u64,
        shown: bool,
    ) -> Result<bool, String> {
        self.require_glyph_context(token, None)?;
        self.set_constraints_shown(shown)?;
        self.sketch_memory.context = None;
        Ok(true)
    }
    pub(super) fn sketch_glyph_wire_index(
        &self,
        edit: crate::HeadSketchEdit,
    ) -> Result<i64, String> {
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|error| error.to_string())?;
        crate::sketch_geometric::wire_sketch_index(&session.snapshot().sketches, edit.sketch)
    }
    fn require_glyph_context(
        &self,
        token: u64,
        glyph: Option<oblikovati_model::sketch::SketchConstraintId>,
    ) -> Result<crate::HeadSketchEdit, String> {
        let target = self
            .sketch_memory
            .context
            .as_ref()
            .ok_or_else(|| format!("glyph context {token}: expected a current context"))?;
        let edit = self
            .active_sketch_edit()
            .ok_or_else(|| format!("glyph context {token}: expected an active sketch"))?;
        if target.token != token
            || target.document != edit.document
            || target.sketch != edit.sketch
            || self.sketch_tool.is_some()
            || target.glyph_candidates.is_empty()
            || glyph.is_some_and(|id| {
                !target
                    .glyph_candidates
                    .iter()
                    .any(|candidate| candidate.constraint == id)
            })
            || !self.sketch_context_valid(edit.document, edit.sketch, target.generation)
        {
            return Err(format!(
                "glyph context {token}, glyph {glyph:?}: expected the current captured glyph"
            ));
        }
        Ok(edit)
    }
    pub(super) fn delete_context_relation(&mut self, id: u64) -> Result<bool, String> {
        self.require_context_relation(id)?;
        let request = self.context_relation_request(id)?;
        oblikovati_api::client::Client::new(self.document_host())
            .constrain()
            .delete(&request)
            .map_err(|e| e.to_string())?;
        Ok(true)
    }
    pub(super) fn retire_sketch_context(&mut self, token: u64) -> bool {
        if !self
            .sketch_memory
            .context
            .as_ref()
            .is_some_and(|context| context.token == token)
        {
            return false;
        }
        self.sketch_memory.context = None;
        true
    }

    pub(super) fn select_context_candidate(
        &mut self,
        token: u64,
        entity: oblikovati_model::sketch::SketchEntityId,
    ) -> Result<bool, String> {
        let edit = self.checked_context_candidate(token, entity)?;
        self.store_sketch_click(edit, Some(entity), SelectionClickMode::Replace)?;
        self.publish_selection_count(edit)?;
        self.sketch_memory.context = None;
        Ok(true)
    }

    fn checked_context_candidate(
        &self,
        token: u64,
        entity: oblikovati_model::sketch::SketchEntityId,
    ) -> Result<crate::HeadSketchEdit, String> {
        let target = self.sketch_memory.context.as_ref().ok_or_else(|| {
            format!("sketch context {token}, entity {entity:?}: expected an open context")
        })?;
        let edit = self.active_sketch_edit().ok_or_else(|| {
            format!("sketch context {token}, entity {entity:?}: expected an active sketch")
        })?;
        if !self.context_choice_is_current(target, edit, token, entity) {
            return Err(format!(
                "sketch context {token}, entity {entity:?}: expected a current offered candidate"
            ));
        }
        Ok(edit)
    }

    fn context_choice_is_current(
        &self,
        target: &super::owner_precise_input::SketchContextTarget,
        edit: crate::HeadSketchEdit,
        token: u64,
        entity: oblikovati_model::sketch::SketchEntityId,
    ) -> bool {
        target.token == token
            && target.document == edit.document
            && target.sketch == edit.sketch
            && self.sketch_tool.is_none()
            && target.candidates.iter().any(|item| item.entity == entity)
            && self.sketch_context_valid(edit.document, edit.sketch, target.generation)
    }
    fn require_context_relation(&mut self, id: u64) -> Result<(), String> {
        let state = self.gui_command_state()?;
        let action = oblikovati_gui::GuiShellAction::DeleteSketchRelation(id);
        if action.enabled(&state)
            && state
                .sketch_context
                .as_ref()
                .is_some_and(|context| context.relations.iter().any(|relation| relation.id == id))
        {
            return Ok(());
        }
        Err(format!(
            "sketch relation {id}: expected an available relation under the context pointer"
        ))
    }
    fn context_relation_request(&self, id: u64) -> Result<DeleteConstraintArgs, String> {
        let edit = self
            .active_sketch_edit()
            .ok_or_else(|| format!("sketch relation {id}: expected an active sketch"))?;
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|e| e.to_string())?;
        let snapshot = session.snapshot();
        let sketch_index =
            crate::sketch_geometric::wire_sketch_index(&snapshot.sketches, edit.sketch)?;
        let geometry = snapshot
            .sketches
            .get(edit.sketch)
            .map_err(|e| e.to_string())?
            .geometry();
        let index = geometry
            .geometric_constraint_index(oblikovati_model::sketch::SketchConstraintId(id))
            .map_err(|e| e.to_string())?;
        let constraint_index = i64::try_from(index)
            .map_err(|_| format!("constraint index {index}: expected an i64"))?;
        Ok(DeleteConstraintArgs {
            sketch_index,
            constraint_index,
        })
    }
}
