// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl DocumentProperties {
    /// Restore a stored property snapshot, including its parameter-source name.
    /// This does not bind or evaluate the source parameter.
    /// Example: `properties.restore_property(USER_DEFINED, stored_property)?;`.
    pub fn restore_property(
        &mut self,
        set: &str,
        property: DocumentProperty,
    ) -> Result<(), PropertyError> {
        self.put(set, &property.name, property.value)?;
        self.find_mut(set, &property.name)
            .expect("validated property")
            .from_parameter = property.from_parameter;
        Ok(())
    }
}
