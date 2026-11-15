// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_session::workspace::OpenActivation;
use std::path::Path;

/// Application-owned observer construction for new document sessions.
pub trait DocumentObserverFactory {
    type Observer: TransactionObserver;
    /// Build one observer; cached document opens do not call this method.
    /// Example: `factory.create()?` before registering a newly loaded document.
    fn create(&mut self) -> Result<Self::Observer, String>;
}

impl<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'a, O, W, F> {
    /// Enable document creation and new opens with an injected observer factory.
    /// Example: `FileHost::new(files, guids, "alpha").with_observer_factory(factory)`.
    pub fn with_observer_factory(
        mut self,
        factory: &'a mut dyn DocumentObserverFactory<Observer = O>,
    ) -> Self {
        self.context.get_mut().observers = Some(factory);
        self
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn open_deferred_drawing_path(&mut self, path: &Path) -> Result<u64, Error> {
        let observers = &mut self.observers;
        let outcome = self
            .files
            .open_deferred_with_factory(
                path,
                || create_observer(observers),
                self.version,
                self.source,
            )
            .map_err(|cause| Error(cause.to_string()))?;
        let id = outcome.session_id();
        self.files
            .set_visible(id, true)
            .map_err(|cause| Error(cause.to_string()))?;
        Ok(id)
    }
    pub(super) fn open(&mut self, request: OpenDocumentArgs) -> Result<Vec<u8>, Error> {
        validate_open(&request)?;
        let id = if request.express {
            self.open_express_path(Path::new(&request.full_document_name), request.visible)?
        } else {
            self.open_path(Path::new(&request.full_document_name), request.visible)?
        };
        encode(
            &self
                .files
                .document_info(id)
                .map_err(|error| Error(error.to_string()))?,
        )
    }

    fn open_express_path(&mut self, path: &Path, visible: bool) -> Result<u64, Error> {
        let limits = self.assembly_display_limits.ok_or_else(|| {
            Error("express assembly display limits: expected an injected display policy".into())
        })?;
        let producer = self
            .assembly_producer
            .as_ref()
            .ok_or_else(|| {
                Error("express assembly producer: expected an injected producer identity".into())
            })?
            .as_ref()
            .map_err(|error| Error(error.clone()))?;
        let observers = &mut self.observers;
        let outcome = self
            .files
            .open_express_with_factory(
                path,
                || create_observer(observers),
                self.version,
                self.source,
                limits,
                producer,
            )
            .map_err(|error| Error(error.to_string()))?;
        let id = outcome.session_id();
        self.files
            .set_visible(id, visible)
            .map_err(|error| Error(error.to_string()))?;
        Ok(id)
    }

    pub(super) fn load_full(&mut self, request: LoadFullDocumentArgs) -> Result<Vec<u8>, Error> {
        let before = self.source_bindings()?;
        let prepared = self.prepare_load_full(request.id);
        let refresh = self.refresh_after_native_file_change(&before, "documents.loadFull");
        super::lifecycle::combine_lifecycle_result("documents.loadFull", prepared, refresh)?;
        self.files
            .finish_express_assembly(request.id)
            .map_err(|e| Error(e.to_string()))?;
        encode(
            &self
                .files
                .document_info(request.id)
                .map_err(|e| Error(e.to_string()))?,
        )
    }

    fn prepare_load_full(&mut self, id: u64) -> Result<(), Error> {
        if self
            .files
            .document_type(id)
            .map_err(|e| Error(e.to_string()))?
            != oblikovati_api::types::DOCUMENT_ASSEMBLY
        {
            return Err(Error(format!(
                "document {}: expected an assembly for Load Full",
                id
            )));
        }
        if self
            .files
            .express_assembly_view(id)
            .map_err(|e| Error(e.to_string()))?
            .is_some()
        {
            let report = self.resolve_assembly_references(id)?;
            if !report.failures.is_empty() {
                return Err(Error(format!(
                    "assembly {}: expected all components loaded: {:?}",
                    id, report.failures
                )));
            }
        }
        Ok(())
    }

    pub(super) fn open_path(&mut self, path: &Path, visible: bool) -> Result<u64, Error> {
        let observers = &mut self.observers;
        let outcome = self
            .files
            .open_with_factory(
                path,
                || create_observer(observers),
                self.version,
                self.source,
                OpenActivation::Activate,
            )
            .map_err(|error| Error(error.to_string()))?;
        let id = outcome.session_id();
        self.files
            .set_visible(id, visible)
            .map_err(|error| Error(error.to_string()))?;
        self.open_drawing_reference(id);
        if self
            .files
            .workspace()
            .session(id)
            .map_err(|error| Error(error.to_string()))?
            .snapshot()
            .assembly
            .is_some()
        {
            self.resolve_assembly_references(id)?;
        }
        Ok(id)
    }

    pub(super) fn open_drawing_reference(&mut self, drawing: u64) {
        let is_drawing = self
            .files
            .workspace()
            .session(drawing)
            .ok()
            .is_some_and(|session| session.snapshot().drawing.is_some());
        if !is_drawing {
            return;
        }
        let reference = self
            .files
            .workspace()
            .session(drawing)
            .ok()
            .and_then(|session| session.snapshot().drawing.as_ref())
            .filter(|drawing| drawing.sheets().iter().any(|sheet| !sheet.views.is_empty()))
            .map(|drawing| drawing.model_reference().to_owned());
        let Some(reference) = reference.filter(|name| !name.is_empty()) else {
            self.files.clear_drawing_reference(drawing);
            return;
        };
        if self.files.drawing_reference(drawing, &reference).is_some() {
            return;
        }
        if let Err(cause) = self.resolve_document_reference(drawing, &reference, true) {
            self.files
                .reject_drawing_reference(drawing, reference, cause);
        }
    }
}

fn validate_open(request: &OpenDocumentArgs) -> Result<(), Error> {
    if request.full_document_name.is_empty() {
        return Err(Error(
            "documents.open fullDocumentName \"\": expected a nonempty file path".into(),
        ));
    }
    if request.defer_content {
        return Err(Error("documents.open deferContent=true: expected false; reference-stub loading is not yet ported".into()));
    }
    Ok(())
}

pub(super) fn create_observer<O: TransactionObserver>(
    observers: &mut Option<&mut dyn DocumentObserverFactory<Observer = O>>,
) -> Result<O, String> {
    observers
        .as_mut()
        .ok_or_else(|| "new document requires an injected factory".to_owned())?
        .create()
}
