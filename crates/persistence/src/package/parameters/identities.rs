// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use std::collections::{BTreeMap, BTreeSet};

pub(super) fn restore_ids(
    package: &DocumentPackage,
    records: &[ParameterRecipe],
) -> Result<(Vec<u64>, RecipeIdAllocator), PackageCodecError> {
    let model = package.root.get("model");
    let bindings = model.and_then(|model| model.get("parameterIds"));
    let mark = model.and_then(|model| model.get("parameterHighWater"));
    if bindings.is_none() && mark.is_none() {
        let ids: Vec<_> = (1..=records.len() as u64).collect();
        return Ok((
            ids.clone(),
            RecipeIdAllocator::restore(None, &ids).map_err(parameter_error)?,
        ));
    }
    if package.manifest.schema_version < 3 || bindings.is_none() || mark.is_none() {
        return Err(PackageCodecError("parameter reservation metadata: expected schema 3 with both parameterIds and parameterHighWater".into()));
    }
    let bindings: BTreeMap<String, u64> =
        serde_yaml_ng::from_value(bindings.expect("validated field").clone())
            .map_err(codec_error)?;
    if bindings.keys().collect::<BTreeSet<_>>()
        != records.iter().map(|record| &record.name).collect()
    {
        return Err(PackageCodecError(format!(
            "parameterIds keys {:?}: expected exactly the saved parameter names",
            bindings.keys()
        )));
    }
    let ids = records
        .iter()
        .map(|record| bindings[&record.name])
        .collect::<Vec<_>>();
    let mark = unsigned_mark(mark.expect("validated field"))?;
    let allocator = RecipeIdAllocator::restore(Some(mark), &ids).map_err(parameter_error)?;
    Ok((ids, allocator))
}

fn unsigned_mark(value: &Value) -> Result<u64, PackageCodecError> {
    value.as_u64().ok_or_else(|| {
        PackageCodecError(format!(
            "parameterHighWater {value:?}: expected an unsigned 64-bit integer"
        ))
    })
}

pub(super) fn capture_ids(
    model: &mut serde_yaml_ng::Mapping,
    parameters: &ParameterCollection<u64>,
    requested: Option<u64>,
) -> Result<(), PackageCodecError> {
    let prior = model
        .get("parameterHighWater")
        .map(unsigned_mark)
        .transpose()?
        .unwrap_or(0);
    let minimum = parameters
        .ids()
        .iter()
        .copied()
        .max()
        .unwrap_or(0)
        .max(prior);
    let mark = requested.unwrap_or(minimum);
    if mark < minimum {
        return Err(PackageCodecError(format!(
            "parameterHighWater {mark}: expected at least live/retained mark {minimum}"
        )));
    }
    let bindings: BTreeMap<_, _> = parameters
        .ids()
        .iter()
        .map(|&id| (parameters.get(id).expect("owned ID").name.clone(), id))
        .collect();
    model.insert(
        "parameterIds".into(),
        serde_yaml_ng::to_value(bindings).map_err(codec_error)?,
    );
    model.insert("parameterHighWater".into(), mark.into());
    Ok(())
}

impl DocumentPackage {
    /// Persist live parameter bindings and the permanent reservation mark together.
    /// Example: `package.with_parameter_reservations(&snapshot.parameters, history.high_water())?`.
    pub fn with_parameter_reservations(
        &self,
        parameters: &ParameterCollection<u64>,
        mark: u64,
    ) -> Result<Self, PackageCodecError> {
        let previous = self.restore_document_parameters()?.ids.high_water();
        if mark < previous {
            return Err(PackageCodecError(format!(
                "parameterHighWater {mark}: expected at least retained mark {previous}"
            )));
        }
        let mut candidate = self.clone();
        let mut model = candidate
            .root
            .get("model")
            .and_then(Value::as_mapping)
            .cloned()
            .unwrap_or_default();
        capture_ids(&mut model, parameters, Some(mark))?;
        if mark == 0 && candidate.manifest.schema_version == 2 {
            return Ok(candidate);
        }
        candidate.manifest.schema_version = 3;
        candidate.root.insert("schemaVersion".into(), 3u32.into());
        candidate.root.insert("model".into(), Value::Mapping(model));
        candidate.restore_document_parameters()?;
        Ok(candidate)
    }
}
