// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentPackage, Mapping, PackageCodecError, Value};
use oblikovati_model::sketch_settings::SketchSettings;

impl DocumentPackage {
    /// Restore explicit settings, keeping absent/null records distinct from document defaults.
    /// Example: `let settings = package.restore_sketch_settings()?;`.
    pub fn restore_sketch_settings(&self) -> Result<Option<SketchSettings>, PackageCodecError> {
        let Some(section) = self
            .root
            .get("sketchSettings")
            .filter(|value| !value.is_null())
        else {
            return Ok(None);
        };
        // Legacy present records use zero values for omitted fields, not document defaults (#3772).
        let settings = serde_yaml_ng::from_value(section.clone()).map_err(|error| {
            PackageCodecError(format!(
                "sketchSettings {section:?}: expected a typed authoring record: {error}"
            ))
        })?;
        require_finite_spacing(&settings)?;
        Ok(Some(settings))
    }
    /// Replace known settings while retaining extension fields and all other package sections.
    /// Example: `let saved = package.with_sketch_settings(snapshot.sketch_settings.as_ref())?;`.
    pub fn with_sketch_settings(
        &self,
        settings: Option<&SketchSettings>,
    ) -> Result<Self, PackageCodecError> {
        if self.restore_sketch_settings()?.as_ref() == settings {
            return Ok(self.clone());
        }
        let mut candidate = self.clone();
        candidate.root.remove("sketchSettings");
        if let Some(settings) = settings {
            candidate.root.insert(
                "sketchSettings".into(),
                Value::Mapping(self.updated_sketch_settings(settings)?),
            );
        }
        Ok(candidate)
    }
    fn updated_sketch_settings(
        &self,
        settings: &SketchSettings,
    ) -> Result<Mapping, PackageCodecError> {
        require_finite_spacing(settings)?;
        let mut section = self
            .root
            .get("sketchSettings")
            .and_then(Value::as_mapping)
            .cloned()
            .unwrap_or_default();
        let encoded = serde_yaml_ng::to_value(settings).map_err(super::codec_error)?;
        section.extend(
            encoded
                .as_mapping()
                .expect("SketchSettings serializes as a struct")
                .clone(),
        );
        Ok(section)
    }
}
fn require_finite_spacing(settings: &SketchSettings) -> Result<(), PackageCodecError> {
    for (name, spacing) in [
        ("xSnapSpacing", settings.x_snap_spacing),
        ("ySnapSpacing", settings.y_snap_spacing),
    ] {
        if !spacing.is_finite() {
            return Err(PackageCodecError(format!(
                "sketchSettings {name} {spacing:?}: expected finite centimetres"
            )));
        }
    }
    Ok(())
}
