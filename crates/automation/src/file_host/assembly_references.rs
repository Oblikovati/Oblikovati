// SPDX-License-Identifier: GPL-2.0-only
//! Resolve authored component references without activating their documents.
use super::*;
use oblikovati_session::{files::FileOpenOutcome, workspace::OpenActivation};
use std::path::Path;

/// One unresolved component and its cause; example: `failure.reference` identifies the occurrence source.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct AssemblyReferenceFailure {
    pub reference: String,
    pub cause: String,
}
/// The independently resolved and failed references; example: inspect `report.failures` after resolution.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct AssemblyReferenceReport {
    pub resolved: Vec<(String, u64)>,
    pub failures: Vec<AssemblyReferenceFailure>,
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'_, O, W, F> {
    /// Resolve an assembly's component references; example: `host.resolve_assembly_references(id)?`.
    pub fn resolve_assembly_references(
        &self,
        owner: u64,
    ) -> Result<AssemblyReferenceReport, Error> {
        let mut context = self
            .context
            .try_borrow_mut()
            .map_err(|_| busy("resolve_assembly_references"))?;
        let before = context.source_bindings()?;
        let response = context.resolve_assembly_references(owner);
        let refresh =
            context.refresh_after_native_file_change(&before, "resolve_assembly_references");
        super::lifecycle::combine_lifecycle_result("resolve_assembly_references", response, refresh)
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn resolve_assembly_references(
        &mut self,
        owner: u64,
    ) -> Result<AssemblyReferenceReport, Error> {
        let mut visited = std::collections::BTreeSet::new();
        let mut pending = std::collections::VecDeque::from([owner]);
        let mut result = AssemblyReferenceReport::default();
        while let Some(document) = pending.pop_front() {
            if !visited.insert(document) {
                continue;
            }
            let report = self.resolve_one_assembly(document)?;
            for (_, target) in &report.resolved {
                if self
                    .files
                    .workspace()
                    .session(*target)
                    .map_err(|cause| Error(cause.to_string()))?
                    .snapshot()
                    .assembly
                    .is_some()
                {
                    pending.push_back(*target);
                }
            }
            result.resolved.extend(report.resolved);
            result.failures.extend(report.failures);
        }
        Ok(result)
    }
    fn resolve_one_assembly(&mut self, owner: u64) -> Result<AssemblyReferenceReport, Error> {
        let session = self
            .files
            .workspace()
            .session(owner)
            .map_err(|cause| Error(cause.to_string()))?;
        let records = session
            .snapshot()
            .assembly
            .as_ref()
            .ok_or_else(|| Error(format!("document {owner}: expected an assembly")))?;
        let names: Vec<String> = oblikovati_model::assembly::component_references(records)
            .into_iter()
            .map(str::to_owned)
            .collect();
        if !session.in_transaction() {
            self.files
                .reconcile_document_references(owner)
                .map_err(|cause| Error(cause.to_string()))?;
        }
        let mut report = AssemblyReferenceReport::default();
        for name in names {
            self.resolve_assembly_reference(owner, name, &mut report)?;
        }
        Ok(report)
    }
    fn resolve_assembly_reference(
        &mut self,
        owner: u64,
        name: String,
        report: &mut AssemblyReferenceReport,
    ) -> Result<(), Error> {
        let result = self
            .files
            .document_reference(owner, &name)
            .map_or_else(|| self.resolve_document_reference(owner, &name, false), Ok);
        match result {
            Ok(model) => report.resolved.push((name, model)),
            Err(cause) => {
                self.files
                    .reject_document_reference(owner, name.clone(), cause.clone())
                    .map_err(|error| Error(error.to_string()))?;
                report.failures.push(AssemblyReferenceFailure {
                    reference: name,
                    cause,
                });
            }
        }
        Ok(())
    }
    pub(super) fn resolve_document_reference(
        &mut self,
        owner: u64,
        name: &str,
        drawing: bool,
    ) -> Result<u64, String> {
        let (model, background) = self.acquire_document_reference(owner, name)?;
        let attached = if drawing {
            self.files
                .attach_drawing_reference(owner, name.into(), model, background)
        } else {
            self.files
                .attach_document_reference(owner, name.into(), model, background)
        };
        if let Err(cause) = attached {
            return Err(self.cleanup_failed_reference(model, background, cause.to_string()));
        }
        Ok(model)
    }
    fn acquire_document_reference(
        &mut self,
        owner: u64,
        name: &str,
    ) -> Result<(u64, bool), String> {
        if let Some(id) = self
            .files
            .resolve_parameter_source(name)
            .map_err(|cause| cause.to_string())?
        {
            return Ok((id, false));
        }
        let path = self
            .files
            .reference_path(owner, Path::new(name))
            .map_err(|cause| cause.to_string())?;
        let observers = &mut self.observers;
        let outcome = self
            .files
            .open_with_factory(
                &path,
                || super::open::create_observer(observers),
                self.version,
                self.source,
                OpenActivation::Background,
            )
            .map_err(|cause| cause.to_string())?;
        let model = outcome.session_id();
        let background = matches!(outcome, FileOpenOutcome::Opened(_));
        if background && let Err(cause) = self.files.set_visible(model, false) {
            return Err(self.cleanup_failed_reference(model, true, cause.to_string()));
        }
        Ok((model, background))
    }
    fn cleanup_failed_reference(&mut self, model: u64, background: bool, cause: String) -> String {
        if !background {
            return cause;
        }
        match self.files.close_discarding_changes(model) {
            Ok(_) => cause,
            Err(cleanup) => {
                format!("{cause}; newly opened model {model} could not be closed: {cleanup}")
            }
        }
    }
}
