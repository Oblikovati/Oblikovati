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
    pub(super) fn open(&mut self, request: OpenDocumentArgs) -> Result<Vec<u8>, Error> {
        validate_open(&request)?;
        let observers = &mut self.observers;
        let outcome = self
            .files
            .open_with_factory(
                Path::new(&request.full_document_name),
                || create_observer(observers),
                self.version,
                self.source,
                OpenActivation::Activate,
            )
            .map_err(|error| Error(error.to_string()))?;
        let id = outcome.session_id();
        self.files
            .set_visible(id, request.visible)
            .map_err(|error| Error(error.to_string()))?;
        self.open_drawing_reference(id);
        encode(
            &self
                .files
                .document_info(id)
                .map_err(|error| Error(error.to_string()))?,
        )
    }

    fn open_drawing_reference(&mut self, drawing: u64) {
        let reference = self
            .files
            .workspace()
            .session(drawing)
            .ok()
            .and_then(|session| session.snapshot().drawing.as_ref())
            .filter(|drawing| drawing.sheets().iter().any(|sheet| !sheet.views.is_empty()))
            .map(|drawing| drawing.model_reference().to_owned());
        let Some(reference) = reference.filter(|name| !name.is_empty()) else {
            return;
        };
        if self.files.drawing_reference(drawing, &reference).is_some() {
            return;
        }
        match self.files.resolve_parameter_source(&reference) {
            Ok(Some(model)) => {
                self.files
                    .attach_drawing_reference(drawing, reference, model, false);
                return;
            }
            Ok(None) => {}
            Err(cause) => {
                self.files
                    .reject_drawing_reference(drawing, reference, cause.to_string());
                return;
            }
        }
        let path = Path::new(&reference);
        let path = if path.is_absolute() {
            path.to_path_buf()
        } else {
            match self.files.path(drawing).ok().and_then(Path::parent) {
                Some(folder) => folder.join(path),
                None => return,
            }
        };
        let observers = &mut self.observers;
        let outcome = self.files.open_with_factory(
            &path,
            || create_observer(observers),
            self.version,
            self.source,
            OpenActivation::Background,
        );
        let outcome = match outcome {
            Ok(outcome) => outcome,
            Err(cause) => {
                self.files
                    .reject_drawing_reference(drawing, reference, cause.to_string());
                return;
            }
        };
        let model = outcome.session_id();
        let background = matches!(
            outcome,
            oblikovati_session::files::FileOpenOutcome::Opened(_)
        );
        if background && let Err(cause) = self.files.set_visible(model, false) {
            self.files
                .reject_drawing_reference(drawing, reference, cause.to_string());
            return;
        }
        self.files
            .attach_drawing_reference(drawing, reference, model, background);
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
