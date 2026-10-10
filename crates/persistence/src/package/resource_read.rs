// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentPackage, ResourceContents};
use std::{borrow::Cow, error::Error, fmt, io};

/// Application-owned bundle access; origin is a bundle key, never a filesystem path.
pub trait ApplicationResources {
    /// Return the exact bundled content or an error; do not substitute another resource.
    /// Example: `provider.read_resource("TrueTypeFont", "Liberation Sans")?;`.
    fn read_resource(&mut self, kind: &str, origin: &str) -> io::Result<Vec<u8>>;
}

/// A missing package entry or a failed lookup of its application-provided content.
#[derive(Debug)]
pub enum ResourceReadError {
    Missing {
        id: String,
    },
    Application {
        id: String,
        kind: String,
        origin: String,
        cause: io::Error,
    },
}

impl DocumentPackage {
    /// Resolve bytes by exact resource ID, without consulting the original import path.
    /// Stored bytes borrow the package; bundled bytes are owned by the result.
    /// Example: `let bytes = package.resource_bytes("font-id", &mut bundles)?;`.
    pub fn resource_bytes<'package>(
        &'package self,
        id: &str,
        application: &mut dyn ApplicationResources,
    ) -> Result<Cow<'package, [u8]>, ResourceReadError> {
        let resource = self
            .resources
            .get(id)
            .ok_or_else(|| ResourceReadError::Missing { id: id.into() })?;
        match &resource.contents {
            ResourceContents::Utf8(text) => Ok(Cow::Borrowed(text.as_bytes())),
            ResourceContents::Binary(bytes) => Ok(Cow::Borrowed(bytes)),
            ResourceContents::Embedded => application
                .read_resource(&resource.kind, &resource.origin)
                .map(Cow::Owned)
                .map_err(|cause| ResourceReadError::Application {
                    id: id.into(),
                    kind: resource.kind.clone(),
                    origin: resource.origin.clone(),
                    cause,
                }),
        }
    }
}

impl fmt::Display for ResourceReadError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Missing { id } => write!(
                formatter,
                "resource {id:?}: expected an existing package resource"
            ),
            Self::Application {
                id,
                kind,
                origin,
                cause,
            } => write!(
                formatter,
                "resource {id:?} type {kind:?} origin {origin:?}: expected exact application-provided content: {cause}"
            ),
        }
    }
}

impl Error for ResourceReadError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Missing { .. } => None,
            Self::Application { cause, .. } => Some(cause),
        }
    }
}
