// SPDX-License-Identifier: GPL-2.0-only
//! Read and remove relations under the sketch context pointer.
use super::*;
use oblikovati_api::wire::DeleteConstraintArgs;

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
            point: point.map(|point| point.0),
            center_point: point.is_some_and(|point| geometry.is_center_point(point)),
            candidates: target.candidates.clone(),
            relations,
        }))
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
    fn require_context_relation(&self, id: u64) -> Result<(), String> {
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
