// SPDX-License-Identifier: GPL-2.0-only
//! Native path operations share the document owner's reference and save services.
use super::*;
use std::path::Path;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'_, O, W, F> {
    /// Open a drawing using its saved presentation; example: `host.open_deferred_drawing_path(path)?`.
    pub fn open_deferred_drawing_path(&self, path: &Path) -> Result<u64, Error> {
        let mut context = self
            .context
            .try_borrow_mut()
            .map_err(|_| busy("open_deferred_drawing_path"))?;
        let before = context.source_bindings()?;
        let response = context.open_deferred_drawing_path(path);
        let refresh =
            context.refresh_after_native_file_change(&before, "open_deferred_drawing_path");
        super::lifecycle::combine_lifecycle_result("open_deferred_drawing_path", response, refresh)
    }
    /// Open an OS path without converting it to UTF-8; example: `host.open_path(path, true)?`.
    pub fn open_path(&self, path: &Path, visible: bool) -> Result<u64, Error> {
        let mut context = self
            .context
            .try_borrow_mut()
            .map_err(|_| busy("open_path"))?;
        let before = context.source_bindings()?;
        let response = context.open_path(path, visible);
        let refresh = context.refresh_after_native_file_change(&before, "open_path");
        super::lifecycle::combine_lifecycle_result("open_path", response, refresh)
    }
    /// Save to an OS path and return nonfatal diagnostics; example: `host.save_as_path(id, path)?`.
    pub fn save_as_path(&self, id: u64, path: &Path) -> Result<Vec<SaveWarning>, Error> {
        let mut context = self
            .context
            .try_borrow_mut()
            .map_err(|_| busy("save_as_path"))?;
        let before = context.source_bindings()?;
        let response = context.stage_drawing_views(id).and_then(|()| {
            let warnings = context.stage_assembly_view(id, path)?.unwrap_or_default();
            let result = {
                let FileContext {
                    files,
                    version,
                    source,
                    ..
                } = &mut *context;
                files
                    .save_as(id, path, version, *source)
                    .map_err(|error| Error(error.to_string()))
            };
            context.files.clear_assembly_view(id);
            result.map(|()| warnings)
        });
        let refresh = context.refresh_after_native_file_change(&before, "save_as_path");
        super::lifecycle::combine_lifecycle_result("save_as_path", response, refresh)
    }
}
