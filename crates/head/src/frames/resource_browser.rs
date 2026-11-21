// SPDX-License-Identifier: GPL-2.0-only
//! Hand a product resource to the system browser without a shell.
use std::{path::PathBuf, process::Command};

pub(super) fn open(url: &str) -> Result<(), String> {
    let mut child = browser_command(url)?
        .spawn()
        .map_err(|error| format!("Team Web {url:?}: expected a working system browser: {error}"))?;
    // A browser can outlive the GUI command. Reap the opener outside the UI thread.
    std::thread::spawn(move || {
        let _ = child.wait();
    });
    Ok(())
}

fn browser_command(url: &str) -> Result<Command, String> {
    if !url.starts_with("https://") || url.len() == 8 || url.chars().any(char::is_whitespace) {
        return Err(format!(
            "Team Web URL {url:?}: expected an HTTPS resource URL"
        ));
    }
    let mut command = Command::new(system_opener()?);
    if cfg!(target_os = "windows") {
        command.arg("url.dll,FileProtocolHandler");
    }
    command.arg(url);
    Ok(command)
}

#[cfg(target_os = "linux")]
fn system_opener() -> Result<PathBuf, String> {
    let paths = [
        "/usr/bin/xdg-open",
        "/usr/local/bin/xdg-open",
        "/bin/xdg-open",
    ];
    paths
        .into_iter()
        .map(PathBuf::from)
        .find(|path| path.is_file())
        .ok_or_else(|| format!("Team Web opener: expected xdg-open in {paths:?}"))
}

#[cfg(target_os = "windows")]
fn system_opener() -> Result<PathBuf, String> {
    let root =
        std::env::var_os("SystemRoot").map_or_else(|| PathBuf::from(r"C:\Windows"), PathBuf::from);
    if !root.is_absolute() {
        return Err(format!(
            "SystemRoot {root:?}: expected an absolute Windows directory"
        ));
    }
    Ok(root.join(r"System32\rundll32.exe"))
}

#[cfg(test)]
#[test]
fn resource_browser_refuses_non_https_and_whitespace() {
    for url in ["file:///home/report", "https://", "https://example.org/a b"] {
        let error = browser_command(url).unwrap_err();
        assert!(error.contains(&format!("{url:?}")), "{error}");
    }
}
