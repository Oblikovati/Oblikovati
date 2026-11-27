// SPDX-License-Identifier: GPL-2.0-only

use super::*;

pub(super) fn restore_mark(
    package: &DocumentPackage,
    parameters: &ParameterCollection<u64>,
) -> Result<i64, PackageCodecError> {
    let minimum = parameters
        .derived_tables()
        .iter()
        .map(|table| table.id)
        .max()
        .unwrap_or(0);
    let Some(value) = package
        .root
        .get("model")
        .and_then(|model| model.get("derivedTableHighWater"))
    else {
        return Ok(minimum);
    };
    let mark = value.as_i64().ok_or_else(|| {
        PackageCodecError(format!(
            "derivedTableHighWater {value:?}: expected a signed integer at least {minimum}"
        ))
    })?;
    if package.manifest.schema_version < 3 || mark < minimum {
        return Err(PackageCodecError(format!(
            "derivedTableHighWater {mark} in schema {}: expected schema 3 and a mark at least {minimum}",
            package.manifest.schema_version
        )));
    }
    Ok(mark)
}

impl DocumentPackage {
    /// Capture permanent table reservations before capturing the current snapshot.
    /// Example: `package.with_derived_reservations(history.derived_high_water())?`.
    pub fn with_derived_reservations(&self, mark: i64) -> Result<Self, PackageCodecError> {
        let previous = self.restore_document_parameters()?.derived_high_water;
        if mark < previous {
            return Err(PackageCodecError(format!(
                "derivedTableHighWater {mark}: expected at least retained mark {previous}"
            )));
        }
        let mut candidate = self.clone();
        if mark == 0 {
            return Ok(candidate);
        }
        let mut model = candidate
            .root
            .get("model")
            .and_then(Value::as_mapping)
            .cloned()
            .unwrap_or_default();
        model.insert("derivedTableHighWater".into(), mark.into());
        candidate.root.insert("model".into(), Value::Mapping(model));
        candidate.root.insert("schemaVersion".into(), 3u32.into());
        candidate.manifest.schema_version = 3;
        Ok(candidate)
    }
}
