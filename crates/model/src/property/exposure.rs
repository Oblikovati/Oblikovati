// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::parameter::{ParameterCollection, ParameterValue};

impl DocumentProperties {
    /// Copy a parameter's typed value into its user-defined property.
    /// The host calls this again after an exposed parameter changes.
    /// Example: `properties.expose_parameter(&parameters, width_id)?;`.
    pub fn expose_parameter<Id: Copy + Ord + fmt::Debug>(
        &mut self,
        parameters: &ParameterCollection<Id>,
        id: Id,
    ) -> Result<(), PropertyError> {
        let parameter = parameters.get(id).ok_or_else(|| {
            PropertyError(format!("parameter {id:?}: expected an existing parameter"))
        })?;
        let value = match parameter.current() {
            Some(ParameterValue::Text(value)) => VariantValue::String(value.clone()),
            Some(ParameterValue::Boolean(value)) => VariantValue::Boolean(*value),
            _ => VariantValue::Double(
                parameters
                    .model_value(id)
                    .map_err(|error| PropertyError(error.to_string()))?
                    .value,
            ),
        };
        self.put(USER_DEFINED, &parameter.name, value)?;
        self.find_mut(USER_DEFINED, &parameter.name)
            .expect("inserted property")
            .from_parameter = parameter.name.clone();
        Ok(())
    }
}

impl DocumentProperties {
    /// Refresh exported parameters and remove obsolete owned projections atomically.
    /// Example: `properties.sync_exposed_parameters(&parameters)?;`.
    pub fn sync_exposed_parameters<Id: Copy + Ord + fmt::Debug>(
        &mut self,
        parameters: &ParameterCollection<Id>,
    ) -> Result<(), PropertyError> {
        let selected: Vec<Id> = parameters
            .ids()
            .iter()
            .copied()
            .filter(|id| {
                parameters
                    .get(*id)
                    .expect("owned parameter")
                    .annotations
                    .exposed_as_property
            })
            .collect();
        let names: std::collections::BTreeSet<&str> = selected
            .iter()
            .map(|id| {
                parameters
                    .get(*id)
                    .expect("selected parameter")
                    .name
                    .as_str()
            })
            .collect();
        let mut candidate = self.clone();
        for set in &mut candidate.sets {
            set.properties.retain(|property| {
                property.from_parameter.is_empty()
                    || names.contains(property.from_parameter.as_str())
            });
        }
        for id in selected {
            let parameter = parameters.get(id).expect("selected parameter");
            if parameter.current().is_none() {
                // #3659: a stale property must not block publication of failed parameter health.
                candidate.remove(USER_DEFINED, &parameter.name);
                continue;
            }
            candidate.expose_parameter(parameters, id)?;
        }
        *self = candidate;
        Ok(())
    }
}
