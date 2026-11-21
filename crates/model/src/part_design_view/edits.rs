// SPDX-License-Identifier: GPL-2.0-only
//! Atomic edits of part design-view definitions (#4118).
use super::{PartDesignView, PartDesignViews};
use oblikovati_api::types::{PartAnnotationScale, PartDesignViewCamera};
use std::collections::BTreeSet;

impl PartDesignViews {
    /// Store visibility for known bodies in an active unlocked view.
    /// Example: `views.set_body_visibility(6, &keys, &available, false)?`.
    pub fn set_body_visibility(
        &mut self,
        id: u64,
        keys: &[String],
        available: &BTreeSet<String>,
        visible: bool,
    ) -> Result<(), String> {
        let selected = self.check_body_selection(id, keys, available)?;
        let view = self.get_mut(id)?;
        for key in selected {
            if visible {
                view.hidden_body_keys.remove(key);
            } else {
                view.hidden_body_keys.insert(key.clone());
            }
        }
        Ok(())
    }

    fn check_body_selection<'a>(
        &self,
        id: u64,
        keys: &'a [String],
        available: &'a BTreeSet<String>,
    ) -> Result<Vec<&'a String>, String> {
        let view = self.get(id)?;
        if !view.active || view.locked {
            return Err(format!("part view ID {id}: expected active unlocked view"));
        }
        let selected: Vec<_> = if keys.is_empty() {
            available.iter().collect()
        } else {
            keys.iter().collect()
        };
        let unique: BTreeSet<_> = selected.iter().copied().collect();
        if unique.len() != selected.len() || selected.iter().any(|key| !available.contains(*key)) {
            return Err(format!(
                "body keys {keys:?}: expected distinct current body keys"
            ));
        }
        Ok(selected)
    }

    /// Capture one view's camera; example: `views.capture_camera(6, camera)?`.
    pub fn capture_camera(&mut self, id: u64, camera: PartDesignViewCamera) -> Result<(), String> {
        self.get_mut(id)?.capture_camera(camera)
    }

    /// Apply supplied view options atomically; example: `views.set_options(&request)?`.
    pub fn set_options(
        &mut self,
        request: &oblikovati_api::wire::SetPartDesignViewOptionsArgs,
    ) -> Result<(), String> {
        let mut candidate = self.get(request.id)?.clone();
        candidate.apply_options(request)?;
        *self.get_mut(request.id)? = candidate;
        Ok(())
    }
    /// Copy a view without changing selection; example: `views.copy(1, 6, "Review")?`.
    pub fn copy(&mut self, source: u64, id: u64, name: &str) -> Result<(), String> {
        self.require_unused(id, name)?;
        let mut view = self.get(source)?.clone();
        view.id = id;
        view.name = name.into();
        view.built_in = false;
        view.active = false;
        self.views.push(view);
        Ok(())
    }
    /// Rename an authored view; example: `views.rename(6, "Front Review")?`.
    pub fn rename(&mut self, id: u64, name: &str) -> Result<(), String> {
        self.require_name(name, Some(id))?;
        let view = self.get_mut(id)?;
        if view.built_in {
            return Err(format!(
                "part view ID {id}: built-in view rename is not supported pending behavior evidence"
            ));
        }
        view.name = name.into();
        Ok(())
    }
    /// Change active selection; example: `views.activate(6)?`.
    pub fn activate(&mut self, id: u64) -> Result<(), String> {
        self.get(id)?;
        for view in &mut self.views {
            view.active = view.id == id;
        }
        Ok(())
    }
    /// Delete an authored view; example: `views.delete(6)?`.
    pub fn delete(&mut self, id: u64) -> Result<(), String> {
        if self.get(id)?.built_in {
            return Err(format!(
                "part view ID {id}: built-in view deletion is not supported pending behavior evidence"
            ));
        }
        let index = self
            .views
            .iter()
            .position(|view| view.id == id)
            .ok_or_else(|| format!("part view ID {id}: expected an existing view"))?;
        let was_active = self.views.remove(index).active;
        if was_active {
            self.activate(1)?;
        }
        Ok(())
    }
    /// Choose an unused generated name; example: `views.generated_name()`.
    pub fn generated_name(&self) -> String {
        (1usize..)
            .map(|n| format!("View{n}"))
            .find(|name| self.views.iter().all(|view| view.name != *name))
            .expect("finite view set leaves a generated name")
    }
    fn require_unused(&self, id: u64, name: &str) -> Result<(), String> {
        if id <= 5 || self.views.iter().any(|view| view.id == id) {
            return Err(format!(
                "part view ID {id}: expected an unused authored ID above 5"
            ));
        }
        self.require_name(name, None)
    }
    fn require_name(&self, name: &str, except: Option<u64>) -> Result<(), String> {
        if name.trim().is_empty()
            || name.contains('\0')
            || self
                .views
                .iter()
                .any(|v| v.name == name && Some(v.id) != except)
        {
            return Err(format!(
                "part view name {name:?}: expected unique nonempty text without NUL"
            ));
        }
        Ok(())
    }
}

impl PartDesignView {
    fn apply_options(
        &mut self,
        request: &oblikovati_api::wire::SetPartDesignViewOptionsArgs,
    ) -> Result<(), String> {
        if let Some(value) = request.locked.as_deref() {
            self.set_locked(*value);
        }
        if let Some(value) = request.annotation_scale.as_deref() {
            self.set_annotation_scale(value.clone())?;
        }
        if let Some(value) = request.published.as_deref() {
            self.set_published(*value);
        }
        if let Some(value) = request.autosave_camera.as_deref() {
            self.set_autosave_camera(*value);
        }
        Ok(())
    }
    /// Change publication; example: `view.set_published(false)`.
    pub fn set_published(&mut self, value: bool) {
        self.published = value;
    }
    /// Change camera autosave; example: `view.set_autosave_camera(false)`.
    pub fn set_autosave_camera(&mut self, value: bool) {
        self.autosave_camera = value;
    }
    /// Capture a validated camera; example: `view.capture_camera(camera)?`.
    pub fn capture_camera(&mut self, camera: PartDesignViewCamera) -> Result<(), String> {
        if self.locked {
            return Err(format!("part view ID {}: expected unlocked view", self.id));
        }
        super::validation::validate_camera(&camera)?;
        self.camera = Some(camera);
        Ok(())
    }
    /// Change lock state; example: `view.set_locked(true)`.
    pub fn set_locked(&mut self, value: bool) {
        self.locked = value;
    }
    /// Replace the annotation scale; example: `view.set_annotation_scale(scale)?`.
    pub fn set_annotation_scale(&mut self, scale: PartAnnotationScale) -> Result<(), String> {
        if !self.active {
            return Err(format!("part view ID {}: expected active view", self.id));
        }
        super::validation::validate_scale(&scale)?;
        self.annotation_scale = scale;
        Ok(())
    }
}
