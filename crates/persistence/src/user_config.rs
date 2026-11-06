// SPDX-License-Identifier: GPL-2.0-only
use std::path::{Path, PathBuf};

/// Resolve the Go per-user root without I/O; example: `user_config_directory(None, Some(home), None, false)?`.
pub fn user_config_directory(
    explicit: Option<&Path>,
    home: Option<&Path>,
    roaming: Option<&Path>,
    windows: bool,
) -> Result<PathBuf, String> {
    if let Some(root) = explicit.filter(|root| !root.as_os_str().is_empty()) {
        return Ok(root.into());
    }
    let (root, suffix, variable) = if windows {
        (roaming, "oblikovati", "APPDATA")
    } else {
        (home, ".oblikovati", "HOME")
    };
    root.filter(|root| !root.as_os_str().is_empty())
        .map(|root| root.join(suffix))
        .ok_or_else(|| {
            format!("user config {variable}={root:?}: expected a nonempty platform directory")
        })
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn platform_roots_match_go_and_explicit_root_takes_precedence() {
        let home = Path::new("home");
        let roaming = Path::new("roaming");
        assert_eq!(
            user_config_directory(None, Some(home), Some(roaming), false).unwrap(),
            home.join(".oblikovati")
        );
        assert_eq!(
            user_config_directory(None, Some(home), Some(roaming), true).unwrap(),
            roaming.join("oblikovati")
        );
        let explicit = Path::new("isolated-config");
        for windows in [false, true] {
            assert_eq!(
                user_config_directory(Some(explicit), None, None, windows).unwrap(),
                explicit
            );
            assert!(user_config_directory(None, None, None, windows).is_err());
            assert!(
                user_config_directory(None, Some(Path::new("")), Some(Path::new("")), windows)
                    .is_err()
            );
        }
    }
}
