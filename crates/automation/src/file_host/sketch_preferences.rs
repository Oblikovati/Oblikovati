// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::application_options::SketchPreferenceAccess;
impl<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'a, O, W, F> {
    /// Share the application's preference owner across calls and client instances.
    /// Example: `FileHost::new(files, guids, "alpha").with_sketch_preferences(&mut preferences)`.
    pub fn with_sketch_preferences(
        mut self,
        preferences: &'a mut dyn SketchPreferenceAccess,
    ) -> Self {
        self.context.get_mut().sketch_preferences = Some(preferences);
        self
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn sketch_command(
        &mut self,
        method: &str,
        request: &[u8],
    ) -> Result<Vec<u8>, Error> {
        if method == METHOD_SKETCH_ADD_ENTITY {
            return crate::sketches::add_entity::apply(
                self.files,
                crate::decode(method, request)?,
                self.sketch_construction,
            );
        }
        if method != METHOD_SKETCH_CREATE {
            return crate::sketches::apply(self.files, method, request);
        }
        let project_origin = self
            .sketch_preferences
            .as_deref()
            .map(|preferences| preferences.sketch_options().auto_project_origin)
            .unwrap_or_else(|| {
                oblikovati_model::application_options::SketchApplicationOptions::default()
                    .auto_project_origin
            });
        crate::sketches::create::apply(self.files, crate::decode(method, request)?, project_origin)
    }
    pub(super) fn sketch_option_command(
        &mut self,
        method: &str,
        request: &[u8],
    ) -> Result<Vec<u8>, Error> {
        let preferences = self.sketch_preferences.as_deref_mut().ok_or_else(|| {
            Error(format!("application option call {method:?}: expected an injected application preference owner"))
        })?;
        crate::options::apply(preferences, method, request)
    }
}
