// SPDX-License-Identifier: GPL-2.0-only
//! The persisted selection-lock preference and current exact selection pivot.
use super::*;
use oblikovati_model::orientation_preferences::OrientationPreferenceAccess;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Attach the saved orientation preference; example: `owner.with_orientation_preferences(store)`.
    pub fn with_orientation_preferences(
        mut self,
        store: Box<dyn OrientationPreferenceAccess>,
    ) -> Self {
        self.orientation_lock = store.orientation_lock();
        self.orientation_store = Some(store);
        self
    }
    pub(super) fn change_orientation_lock(&mut self, locked: bool) -> Result<bool, String> {
        if self.orientation_lock == locked {
            return Ok(false);
        }
        if locked && self.orientation_pivot()?.is_none() {
            return Err("orientation lock: expected a selected model face or body".into());
        }
        if let Some(store) = self.orientation_store.as_mut() {
            store.set_orientation_lock(locked)?;
        }
        self.orientation_lock = locked;
        Ok(true)
    }
    pub(super) fn orientation_pivot(&mut self) -> Result<Option<[f64; 3]>, String> {
        let Some(document) = self.files.workspace().active_document() else {
            return Ok(None);
        };
        if let Some(face) = self.chosen_model_face(document) {
            return self
                .part_scenes
                .selection_center(
                    &mut self.files,
                    document,
                    face.feature,
                    Some(face.role),
                    None,
                )
                .map(Some);
        }
        let body = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .selected_browser_body()
            .map(|(key, feature)| (key.to_owned(), feature));
        let Some((key, feature)) = body else {
            return Ok(None);
        };
        self.part_scenes
            .selection_center(&mut self.files, document, feature, None, Some(&key))
            .map(Some)
    }
    pub(super) fn orientation_fit_bounds(&mut self) -> Result<Vec<[[f64; 3]; 2]>, String> {
        let document = self
            .files
            .workspace()
            .active_document()
            .ok_or("selection fit: expected an active document")?;
        self.part_scenes
            .visible_body_bounds(&mut self.files, document)
    }
}
