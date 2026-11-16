// SPDX-License-Identifier: GPL-2.0-only
//! The persisted selection-lock preference and current exact selection pivot.
use super::*;
use oblikovati_model::orientation_preferences::OrientationPreferenceAccess;
use oblikovati_model::sketch::SketchPlane;

#[derive(Clone, PartialEq, Eq)]
struct LookAtKey {
    document: u64,
    generation: u64,
    feature: Option<oblikovati_model::feature::FeatureId>,
    face: Option<(oblikovati_model::feature::FeatureId, Vec<u8>)>,
}
pub(super) struct LookAtCache {
    key: LookAtKey,
    plane: Option<SketchPlane>,
}

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn look_at_plane(&self) -> Result<Option<SketchPlane>, String> {
        let Some(key) = self.look_at_key()? else {
            return Ok(None);
        };
        if let Some(cache) = self
            .look_at_cache
            .borrow()
            .as_ref()
            .filter(|cache| cache.key == key)
        {
            return Ok(cache.plane);
        }
        let Some(plane) = self.current_look_at_plane(&key)? else {
            return Ok(None);
        };
        *self.look_at_cache.borrow_mut() = Some(LookAtCache { key, plane });
        Ok(plane)
    }
    fn look_at_key(&self) -> Result<Option<LookAtKey>, String> {
        let Some(document) = self.files.workspace().active_document() else {
            return Ok(None);
        };
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let face = self
            .chosen_model_face(document)
            .map(|face| (face.feature, face.key));
        let key = LookAtKey {
            document,
            generation: session.generation(),
            feature: session.selected_browser_feature(),
            face,
        };
        if key.feature.is_none() && key.face.is_none() {
            return Ok(None);
        }
        Ok(Some(key))
    }
    fn current_look_at_plane(
        &self,
        key: &LookAtKey,
    ) -> Result<Option<Option<SketchPlane>>, String> {
        let session = self
            .files
            .workspace()
            .session(key.document)
            .map_err(|cause| cause.to_string())?;
        let Ok(current) = self.part_scenes.current(session) else {
            return Ok(None);
        };
        let datums = current.datum_evaluation();
        let selected = datums
            .planes()
            .find(|(id, _)| Some(*id) == key.feature)
            .map(|(_, plane)| plane);
        let plane = match selected {
            Some(plane) => Some(plane),
            None => self.selected_face_plane(key.document, key)?,
        };
        Ok(Some(plane))
    }
    fn selected_face_plane(
        &self,
        document: u64,
        key: &LookAtKey,
    ) -> Result<Option<SketchPlane>, String> {
        let Some((feature, face)) = &key.face else {
            return Ok(None);
        };
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let body = self
            .part_scenes
            .current(session)?
            .body(*feature)
            .ok_or_else(|| format!("face feature {feature:?}: expected a current body"))?;
        if body.face_topology(session.history(), guid, face)?.surface
            != oblikovati_martensite::PrismSurfaceKind::Plane
        {
            return Ok(None);
        }
        let plane = match body {
            oblikovati_martensite::NamedSolid::Prism(body) => {
                body.face_plane(session.history(), guid, face)?
            }
            oblikovati_martensite::NamedSolid::Fillet(body) => {
                body.face_plane(session.history(), guid, face)?
            }
        };
        let frame = plane.recipe();
        let center = body.face_bounds_center(session.history(), guid, face)?;
        SketchPlane::new(center, frame.x_axis, frame.y_axis, plane.options())
            .map(Some)
            .map_err(|cause| cause.to_string())
    }
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
                    Some(&face.key),
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
}
