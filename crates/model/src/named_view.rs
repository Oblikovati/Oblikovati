// SPDX-License-Identifier: GPL-2.0-only
//! Persistable camera frames owned by a document.
use oblikovati_api::types::ProjectionTypeEnum;
use std::collections::BTreeMap;

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct NamedViewCamera {
    pub eye: [f64; 3],
    pub target: [f64; 3],
    pub up: [f64; 3],
    pub fov: f64,
    pub projection: ProjectionTypeEnum,
}

impl NamedViewCamera {
    /// Validate a camera before it becomes document data; example: `camera.validate()?`.
    pub fn validate(self) -> Result<(), String> {
        let values = [self.eye, self.target, self.up];
        if values.iter().flatten().any(|value| !value.is_finite())
            || !self.fov.is_finite()
            || self.eye == self.target
            || self.up == [0.; 3]
            || self.fov <= 0.
        {
            return Err(format!(
                "named view camera {self:?}: expected a finite usable camera"
            ));
        }
        Ok(())
    }
}

#[derive(Clone, Debug, Default, PartialEq)]
pub struct NamedViews(BTreeMap<String, NamedViewCamera>);

impl NamedViews {
    /// Save or replace a named camera; example: `views.capture("Iso", camera)?`.
    pub fn capture(&mut self, name: &str, camera: NamedViewCamera) -> Result<(), String> {
        camera.validate()?;
        if name.trim().is_empty() {
            return Err(format!("named view name {name:?}: expected nonempty text"));
        }
        self.0.insert(name.into(), camera);
        Ok(())
    }
    /// Look up a named camera; example: `views.get("Iso")`.
    pub fn get(&self, name: &str) -> Option<NamedViewCamera> {
        self.0.get(name).copied()
    }
    /// Remove one saved camera; example: `views.remove("Iso")`.
    pub fn remove(&mut self, name: &str) -> bool {
        self.0.remove(name).is_some()
    }
    /// Enumerate saved cameras in stable name order; example: `views.entries()`.
    pub fn entries(&self) -> impl Iterator<Item = (&str, NamedViewCamera)> {
        self.0.iter().map(|(name, camera)| (name.as_str(), *camera))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn named_views_validate_replace_and_sort_entries() {
        let camera = NamedViewCamera {
            eye: [0., 0., 2.],
            target: [0.; 3],
            up: [0., 1., 0.],
            fov: 1.,
            projection: ProjectionTypeEnum(86273),
        };
        let mut views = NamedViews::default();
        views.capture("Top", camera).unwrap();
        views.capture("Iso", camera).unwrap();
        assert_eq!(
            views.entries().map(|(name, _)| name).collect::<Vec<_>>(),
            ["Iso", "Top"]
        );
        assert!(views.remove("Top"));
        assert!(views.get("Top").is_none());
    }
}
