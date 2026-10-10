// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentPackage, PackageResource};
use oblikovati_model::identity::{GuidSource, IdentityTransitionError};
use std::{error::Error, fmt};

/// A resource import refused before package mutation.
#[derive(Debug)]
pub enum ResourceImportError {
    Identity(IdentityTransitionError),
    Collision(String),
}

impl DocumentPackage {
    /// Reuse an identical resource or allocate a fresh reference without changing this package.
    /// Equality includes type, encoding, origin, and bytes, matching Go AddResource.
    /// Example: `let (updated, id) = package.with_imported_resource(&resource, &mut guids)?;`.
    pub fn with_imported_resource(
        &self,
        resource: &PackageResource,
        source: &mut dyn GuidSource,
    ) -> Result<(Self, String), ResourceImportError> {
        // BTreeMap order makes reuse deterministic even in files with duplicate payloads.
        if let Some((id, _)) = self
            .resources
            .iter()
            .find(|(_, stored)| *stored == resource)
        {
            return Ok((self.clone(), id.clone()));
        }
        let id = source
            .generate()
            .map_err(ResourceImportError::Identity)?
            .to_string();
        if self.resources.contains_key(&id) {
            return Err(ResourceImportError::Collision(id));
        }
        Ok((self.with_resource(&id, resource), id))
    }
}

impl fmt::Display for ResourceImportError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Identity(cause) => write!(
                formatter,
                "resource identity: expected a fresh GUID: {cause}"
            ),
            Self::Collision(id) => write!(
                formatter,
                "resource GUID {id:?}: expected an unused package key"
            ),
        }
    }
}

impl Error for ResourceImportError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Identity(cause) => Some(cause),
            Self::Collision(_) => None,
        }
    }
}
