// SPDX-License-Identifier: GPL-2.0-only
use super::*;
/// Enforce the existing major/equal-or-older minor handshake, including API 0.x.
/// Example: `check_api_version(Some((0, 158)), (0, 158))?;`.
pub fn check_api_version(
    reported: Option<(i32, i32)>,
    host: (i32, i32),
) -> Result<(i32, i32), Error> {
    if host.0 < 0 || host.1 < 0 {
        return Err(Error(format!(
            "host API {}.{}: expected a nonnegative host version",
            host.0, host.1
        )));
    }
    let version = reported.ok_or_else(|| Error(format!(
        "add-in API version {reported:?}: expected ObkAddInApiMajor and ObkAddInApiMinor exports for host {}.{}", host.0, host.1
    )))?;
    if version.0 < 0 || version.1 < 0 || version.0 != host.0 || version.1 > host.1 {
        return Err(Error(format!(
            "add-in API {}.{}: expected nonnegative version with major {} and minor <= {} for host {}.{}",
            version.0, version.1, host.0, host.1, host.0, host.1
        )));
    }
    Ok(version)
}
