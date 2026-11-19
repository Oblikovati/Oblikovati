// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentPackage, Mapping, PackageCodecError, Value};
use oblikovati_model::sketch_settings::SketchSettings;
use serde::Deserialize;

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
        for name in sketch_setting_names()? {
            section.remove(Value::String((*name).to_owned()));
        }
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

// Read the DTO's declared serde fields so newly omitted values cannot leave stale keys (#4161).
fn sketch_setting_names() -> Result<&'static [&'static str], PackageCodecError> {
    let mut names = None;
    let _ = SketchSettings::deserialize(SketchSettingsFieldNames(&mut names));
    names.ok_or_else(|| PackageCodecError("sketchSettings: expected a serde struct".into()))
}

struct SketchSettingsFieldNames<'a>(&'a mut Option<&'static [&'static str]>);

impl<'de> serde::Deserializer<'de> for SketchSettingsFieldNames<'_> {
    type Error = serde::de::value::Error;

    fn deserialize_struct<V>(
        self,
        _name: &'static str,
        fields: &'static [&'static str],
        _visitor: V,
    ) -> Result<V::Value, Self::Error>
    where
        V: serde::de::Visitor<'de>,
    {
        *self.0 = Some(fields);
        Err(serde::de::Error::custom("field names captured"))
    }

    fn deserialize_any<V>(self, _visitor: V) -> Result<V::Value, Self::Error>
    where
        V: serde::de::Visitor<'de>,
    {
        Err(serde::de::Error::custom("expected a struct"))
    }

    serde::forward_to_deserialize_any! {
        bool i8 i16 i32 i64 u8 u16 u32 u64 f32 f64 char str string bytes
        byte_buf option unit unit_struct newtype_struct seq tuple tuple_struct
        map enum identifier ignored_any
    }
}
