// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    CommandError, DocumentSession, TransactionObserver, sketch_recompute::SketchUpdateMode,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId, history::DocumentSnapshot, sketch::SketchError,
};

impl<O: TransactionObserver> DocumentSession<O> {
    /// Change saved display state without solving; example: `session.set_sketch_visibility(id, false)?`.
    pub fn set_sketch_visibility(
        &mut self,
        sketch: DocumentSketchId,
        visible: bool,
    ) -> Result<(), CommandError<SketchError>> {
        self.set_sketch_visibility_projected(sketch, visible, |reply| reply.map(|_| ()))
    }
    /// Prepare a reply before metadata adoption; example: `session.set_sketch_visibility_projected(id, false, |reply| encode(reply?))?`.
    pub fn set_sketch_visibility_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        visible: bool,
        project: impl FnOnce(Result<&DocumentSnapshot, SketchError>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        // #3826: an unchanged display value must not retire a live tool by advancing generation.
        if self
            .snapshot()
            .sketches
            .get(sketch)
            .is_ok_and(|current| current.visible() == visible)
        {
            return project(Ok(self.snapshot())).map_err(|cause| {
                CommandError::Edit(oblikovati_model::history::EditError::Mutation(
                    crate::MutationError::Command(cause),
                ))
            });
        }
        self.run_with_recompute(
            "Set sketch visibility",
            |snapshot, _| {
                let result = snapshot.sketches.set_visible(sketch, visible);
                project(result.map(|_| &*snapshot))
            },
            SketchUpdateMode::Explicit,
        )
    }
}

impl<O: TransactionObserver> DocumentSession<O> {
    /// Show or hide the relation marks as one history step; repeating the state adds no step.
    /// Example: `session.set_sketch_constraints_shown(id, true)?`.
    pub fn set_sketch_constraints_shown(
        &mut self,
        sketch: DocumentSketchId,
        shown: bool,
    ) -> Result<(), CommandError<SketchError>> {
        let unchanged = self
            .snapshot()
            .sketches
            .get(sketch)
            .is_ok_and(|current| current.constraints_shown() == shown);
        if unchanged {
            return Ok(());
        }
        let label = if shown {
            "Show All Constraints"
        } else {
            "Hide All Constraints"
        };
        self.run(label, |snapshot, _| {
            snapshot.sketches.set_constraints_shown(sketch, shown)
        })
    }
}

impl<O: TransactionObserver> DocumentSession<O> {
    /// Change one point's role as a history step; position and solve are untouched.
    /// Example: `session.set_sketch_center_point(id, point, true)?`.
    pub fn set_sketch_center_point(
        &mut self,
        sketch: DocumentSketchId,
        point: oblikovati_model::sketch::SketchEntityId,
        center: bool,
    ) -> Result<(), CommandError<SketchError>> {
        self.run("Center Point", |snapshot, _| {
            let mut geometry = snapshot.sketches.get(sketch)?.geometry().clone();
            geometry.set_center_point(point, center)?;
            snapshot.sketches.replace_geometry(sketch, geometry)
        })
    }
}

impl<O: TransactionObserver> DocumentSession<O> {
    /// Name a design view of the document; example: `session.create_design_view("Front")`.
    pub fn create_design_view(&mut self, name: &str) -> Result<(), CommandError<SketchError>> {
        self.run("Create Design View", |snapshot, _| {
            if name.is_empty() || snapshot.design_views.iter().any(|known| known == name) {
                return Err(SketchError::Definition(format!(
                    "design view {name:?}: expected a nonempty name not already in {:?}",
                    snapshot.design_views
                )));
            }
            snapshot.design_views.push(name.to_owned());
            Ok(())
        })
    }
}
