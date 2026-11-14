// SPDX-License-Identifier: GPL-2.0-only

use serde::Deserialize;

/// Projection of the embedded release manifest; the full original JSON is also retained.
#[derive(Debug, Deserialize, PartialEq, Eq)]
pub struct BuildIdentity {
    pub format: u32,
    pub version: String,
    pub application_commit: String,
}

impl BuildIdentity {
    /// Validate the build identity before reporting it; example: `BuildIdentity::parse(manifest)?`.
    pub fn parse(manifest: &str) -> Result<Self, String> {
        let identity: Self = serde_json::from_str(manifest).map_err(|error| {
            format!("build manifest: expected format/version/application_commit fields: {error}")
        })?;
        if identity.format != 1 || identity.version.is_empty() {
            return Err(format!(
                "build identity {identity:?}: expected format 1 and a nonempty version"
            ));
        }
        if identity.application_commit != "unknown"
            && (identity.application_commit.len() != 40
                || !identity
                    .application_commit
                    .bytes()
                    .all(|byte| byte.is_ascii_hexdigit()))
        {
            return Err(format!(
                "build commit {:?}: expected a 40-digit SHA or unknown for a development build",
                identity.application_commit
            ));
        }
        Ok(identity)
    }
}
