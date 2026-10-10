// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::parameter::ParameterCollection;

impl DocumentProperties {
    /// Copy a numeric parameter's selected model value into its user-defined property.
    /// The host calls this again after an exposed parameter changes.
    /// Example: `properties.expose_parameter(&parameters, width_id)?;`.
    pub fn expose_parameter<Id: Copy + Ord + fmt::Debug>(
        &mut self,
        parameters: &ParameterCollection<Id>,
        id: Id,
    ) -> Result<(), PropertyError> {
        let parameter = parameters.get(id).ok_or_else(|| {
            PropertyError(format!(
                "parameter {id:?}: expected an existing numeric parameter"
            ))
        })?;
        let value = parameters
            .model_value(id)
            .map_err(|error| PropertyError(error.to_string()))?;
        self.put(
            USER_DEFINED,
            &parameter.name,
            VariantValue::Double(value.value),
        )?;
        self.find_mut(USER_DEFINED, &parameter.name)
            .expect("inserted property")
            .from_parameter = parameter.name.clone();
        Ok(())
    }
}
