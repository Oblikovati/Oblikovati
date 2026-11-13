// SPDX-License-Identifier: GPL-2.0-only
//! Place a component through the shared document owner.
use super::*;
use oblikovati_session::{files::FileOpenOutcome, workspace::OpenActivation};
use std::path::Path;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'_, O, W, F> {
    /// Place a file without activating it; example: `host.place_component_file(path, "Part:1", matrix)?`.
    pub fn place_component_file(
        &self,
        path: &Path,
        name: &str,
        transform: oblikovati_api::types::Matrix,
    ) -> Result<(), Error> {
        let mut context = self
            .context
            .try_borrow_mut()
            .map_err(|_| busy("place_component_file"))?;
        context.place_component_file(path, name, transform)
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    fn place_component_file(
        &mut self,
        path: &Path,
        name: &str,
        transform: oblikovati_api::types::Matrix,
    ) -> Result<(), Error> {
        let owner = self
            .files
            .workspace()
            .active_document()
            .ok_or_else(|| Error("place component: expected an active assembly".into()))?;
        let observers = &mut self.observers;
        let outcome = self
            .files
            .open_with_factory(
                path,
                || super::open::create_observer(observers),
                self.version,
                self.source,
                OpenActivation::Background,
            )
            .map_err(|cause| Error(cause.to_string()))?;
        let component = outcome.session_id();
        let newly_opened = matches!(outcome, FileOpenOutcome::Opened(_));
        let result = self.place_opened_component(owner, component, name, transform, newly_opened);
        if result.is_err() && newly_opened {
            self.files
                .close_discarding_changes(component)
                .map_err(|cause| {
                    Error(format!(
                        "placement failed; component {component} cleanup failed: {cause}"
                    ))
                })?;
        }
        result
    }
    fn place_opened_component(
        &mut self,
        owner: u64,
        component: u64,
        name: &str,
        transform: oblikovati_api::types::Matrix,
        newly_opened: bool,
    ) -> Result<(), Error> {
        if newly_opened {
            self.files
                .set_visible(component, false)
                .map_err(|cause| Error(cause.to_string()))?;
        }
        if self
            .files
            .workspace()
            .session(component)
            .map_err(|cause| Error(cause.to_string()))?
            .snapshot()
            .assembly
            .is_some()
        {
            self.resolve_assembly_references(component)?;
        }
        place(
            self.files,
            PlaceOccurrenceArgs {
                document: component,
                name: name.into(),
                transform,
            },
        )?;
        if newly_opened {
            let reference = self
                .files
                .path(component)
                .map_err(|cause| Error(cause.to_string()))?
                .to_str()
                .ok_or_else(|| Error(format!("component {component}: expected UTF-8 path")))?
                .to_owned();
            self.files
                .keep_background_reference(owner, &reference, component)
                .map_err(|cause| Error(cause.to_string()))?;
        }
        Ok(())
    }
}

pub(super) fn place<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    args: PlaceOccurrenceArgs,
) -> Result<Vec<u8>, Error> {
    let owner = files
        .workspace()
        .active_document()
        .ok_or_else(|| Error("assembly.place: expected an active assembly".into()))?;
    let target = files
        .workspace()
        .session(args.document)
        .map_err(|cause| Error(cause.to_string()))?;
    let children = if target.snapshot().assembly.is_some() {
        Some(crate::file_queries::assembly::rows(
            files,
            args.document,
            1,
        )?)
    } else {
        None
    };
    files
        .place_occurrence(
            owner,
            args.document,
            &args.name,
            args.transform,
            oblikovati_martensite::validate_occurrence_placement,
            |snapshot, id| {
                let record = snapshot
                    .assembly
                    .as_ref()
                    .and_then(|records| records.iter().find(|row| row.session_id == id))
                    .ok_or_else(|| {
                        format!("occurrence {id}: expected a newly placed occurrence")
                    })?;
                let occurrence = crate::file_queries::assembly::occurrence_info(record, children)
                    .map_err(|cause| cause.to_string())?;
                encode(&OccurrenceResult { occurrence }).map_err(|cause| cause.to_string())
            },
        )
        .map_err(Error)
}
