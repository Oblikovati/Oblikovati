// SPDX-License-Identifier: GPL-2.0-only
//! Owned part design views and their stable, persisted identities (#4118).
use oblikovati_api::types::{
    PART_ANNOTATION_SCALE_AUTO, PartAnnotationScale, PartDesignViewCamera, ViewOrientationTypeEnum,
};
use serde::{Deserialize, Serialize};
use std::collections::BTreeSet;

mod edits;
mod validation;
pub use validation::validate_camera as validate_part_camera;

/// One view's authored display state. Example: `view.name()` identifies it.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct PartDesignView {
    id: u64,
    name: String,
    built_in: bool,
    active: bool,
    locked: bool,
    published: bool,
    autosave_camera: bool,
    annotation_scale: PartAnnotationScale,
    standard_orientation: ViewOrientationTypeEnum,
    camera: Option<PartDesignViewCamera>,
    hidden_body_keys: BTreeSet<String>,
}

/// The part's ordered views. Example: `views.active()` returns the checked row.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct PartDesignViews {
    views: Vec<PartDesignView>,
}

impl Default for PartDesignViews {
    fn default() -> Self {
        let names = ["Primary", "Isometric", "Front", "Top", "Right"];
        let views = names
            .into_iter()
            .enumerate()
            .map(|(i, name)| PartDesignView::built_in(i as u64 + 1, name))
            .collect();
        Self { views }
    }
}

impl PartDesignView {
    fn built_in(id: u64, name: &str) -> Self {
        Self {
            id,
            name: name.into(),
            built_in: true,
            active: id == 1,
            locked: false,
            published: true,
            autosave_camera: true,
            annotation_scale: PartAnnotationScale {
                mode: PART_ANNOTATION_SCALE_AUTO,
                numerator: 0,
                denominator: 0,
            },
            standard_orientation: standard_orientation(id),
            camera: None,
            hidden_body_keys: BTreeSet::new(),
        }
    }
    /// Read this view's camera; example: `view.camera()` before restoring it.
    pub fn camera(&self) -> Option<&PartDesignViewCamera> {
        self.camera.as_ref()
    }
    /// Read stored hidden keys; example: `view.hidden_body_keys()` during replay.
    pub fn hidden_body_keys(&self) -> &BTreeSet<String> {
        &self.hidden_body_keys
    }
}

impl PartDesignViews {
    /// List views in creation order; example: `views.list().len()`.
    pub fn list(&self) -> &[PartDesignView] {
        &self.views
    }
    /// Inspect the active view; example: `views.active()`.
    pub fn active(&self) -> Option<&PartDesignView> {
        self.views.iter().find(|view| view.active)
    }
    /// Find a view by stable ID; example: `views.get(1)?`.
    pub fn get(&self, id: u64) -> Result<&PartDesignView, String> {
        self.views
            .iter()
            .find(|view| view.id == id)
            .ok_or_else(|| format!("part view ID {id}: expected an existing view"))
    }
    fn get_mut(&mut self, id: u64) -> Result<&mut PartDesignView, String> {
        self.views
            .iter_mut()
            .find(|view| view.id == id)
            .ok_or_else(|| format!("part view ID {id}: expected an existing view"))
    }
}

impl oblikovati_api::contract::PartDesignView for PartDesignView {
    fn id(&self) -> u64 {
        self.id
    }
    fn name(&self) -> String {
        self.name.clone()
    }
    fn built_in(&self) -> bool {
        self.built_in
    }
    fn primary(&self) -> bool {
        self.id == 1
    }
    fn active(&self) -> bool {
        self.active
    }
    fn locked(&self) -> bool {
        self.locked
    }
    fn published(&self) -> bool {
        self.published
    }
    fn autosave_camera(&self) -> bool {
        self.autosave_camera
    }
    fn annotation_scale(&self) -> PartAnnotationScale {
        self.annotation_scale.clone()
    }
    fn standard_orientation(&self) -> ViewOrientationTypeEnum {
        self.standard_orientation
    }
}

fn standard_orientation(id: u64) -> ViewOrientationTypeEnum {
    use oblikovati_api::types::{
        FRONT_VIEW_ORIENTATION, ISO_TOP_RIGHT_VIEW_ORIENTATION, RIGHT_VIEW_ORIENTATION,
        TOP_VIEW_ORIENTATION,
    };
    match id {
        2 => ISO_TOP_RIGHT_VIEW_ORIENTATION,
        3 => FRONT_VIEW_ORIENTATION,
        4 => TOP_VIEW_ORIENTATION,
        5 => RIGHT_VIEW_ORIENTATION,
        _ => ViewOrientationTypeEnum(0),
    }
}

#[cfg(test)]
mod tests;
