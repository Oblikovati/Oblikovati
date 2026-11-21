// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentPackage, PackageCodecError, Value, codec_error};
use oblikovati_api::types::ProjectionTypeEnum;
use oblikovati_model::named_view::{NamedViewCamera, NamedViews};
use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
struct StoredNamedView {
    name: String,
    eye: [f64; 3],
    target: [f64; 3],
    up: [f64; 3],
    fov: f64,
    projection: i32,
}

impl DocumentPackage {
    /// Restore document-owned named cameras; example: `package.restore_named_views()?`.
    pub fn restore_named_views(&self) -> Result<NamedViews, PackageCodecError> {
        let stored: Vec<StoredNamedView> = match self.root.get("namedViews") {
            None | Some(Value::Null) => return Ok(NamedViews::default()),
            Some(section) => serde_yaml_ng::from_value(section.clone()).map_err(codec_error)?,
        };
        let mut views = NamedViews::default();
        for view in stored {
            views
                .capture(
                    &view.name,
                    NamedViewCamera {
                        eye: view.eye,
                        target: view.target,
                        up: view.up,
                        fov: view.fov,
                        projection: ProjectionTypeEnum(view.projection),
                    },
                )
                .map_err(PackageCodecError)?;
        }
        Ok(views)
    }
    /// Replace the named-camera section; example: `package.with_named_views(&views)?`.
    pub fn with_named_views(&self, views: &NamedViews) -> Result<Self, PackageCodecError> {
        let stored: Vec<StoredNamedView> = views
            .entries()
            .map(|(name, camera)| StoredNamedView {
                name: name.into(),
                eye: camera.eye,
                target: camera.target,
                up: camera.up,
                fov: camera.fov,
                projection: camera.projection.0,
            })
            .collect();
        let mut candidate = self.clone();
        if stored.is_empty() {
            candidate.root.remove("namedViews");
        } else {
            candidate.root.insert(
                "namedViews".into(),
                serde_yaml_ng::to_value(stored).map_err(codec_error)?,
            );
        }
        Ok(candidate)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn named_views_round_trip_through_a_package_section() {
        let mut views = NamedViews::default();
        views
            .capture(
                "Iso",
                NamedViewCamera {
                    eye: [0., 0., 2.],
                    target: [0.; 3],
                    up: [0., 1., 0.],
                    fov: 1.,
                    projection: ProjectionTypeEnum(86273),
                },
            )
            .unwrap();
        let package = DocumentPackage::empty(oblikovati_api::types::DOCUMENT_PART, "Part")
            .with_named_views(&views)
            .unwrap();
        assert_eq!(package.restore_named_views().unwrap(), views);
    }
}
