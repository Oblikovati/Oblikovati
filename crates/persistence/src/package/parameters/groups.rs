// SPDX-License-Identifier: GPL-2.0-only

use super::{PackageCodecError, Value, codec_error, parameter_error};
use oblikovati_model::parameter::ParameterCollection;
use serde::Deserialize;

#[derive(Default, Deserialize)]
#[serde(default, rename_all = "camelCase", deny_unknown_fields)]
struct GroupRecipe {
    internal_name: String,
    display_name: String,
    client_id: String,
    members: Vec<String>,
}

pub(super) fn restore_groups(
    model: Option<&Value>,
    parameters: &mut ParameterCollection<u64>,
) -> Result<(), PackageCodecError> {
    let section = model
        .and_then(Value::as_mapping)
        .and_then(|model| model.get("parameterGroups"));
    let records: Vec<GroupRecipe> = match section {
        None | Some(Value::Null) => return Ok(()),
        Some(section) => serde_yaml_ng::from_value(section.clone()).map_err(codec_error)?,
    };
    for record in records {
        record.restore(parameters)?;
    }
    Ok(())
}

impl GroupRecipe {
    fn restore(self, parameters: &mut ParameterCollection<u64>) -> Result<(), PackageCodecError> {
        parameters
            .add_group(&self.internal_name, &self.display_name, &self.client_id)
            .map_err(parameter_error)?;
        for name in self.members {
            let id = parameters.find(&name).ok_or_else(|| {
                PackageCodecError(format!(
                    "group {:?} member {name:?}: expected an existing parameter name",
                    self.internal_name
                ))
            })?;
            parameters
                .add_to_group(id, &self.internal_name)
                .map_err(parameter_error)?;
        }
        Ok(())
    }
}
