// SPDX-License-Identifier: GPL-2.0-only
//! Open a product resource or local directory with the system desktop.
use std::{
    path::{Path, PathBuf},
    process::Command,
};

pub(super) fn open(url: &str) -> Result<(), String> {
    spawn_reaped(browser_command(url)?)
        .map_err(|error| format!("Team Web {url:?}: expected a working system browser: {error}"))
}

pub(crate) fn open_directory(directory: &Path) -> Result<(), String> {
    let command = directory_command(directory, directory_opener()?)?;
    spawn_reaped(command).map_err(|error| {
        format!("directory {directory:?}: expected a working file manager: {error}")
    })
}

fn directory_command(directory: &Path, opener: PathBuf) -> Result<Command, String> {
    if !directory.is_absolute() || !directory.is_dir() {
        return Err(format!(
            "directory {directory:?}: expected an existing absolute directory"
        ));
    }
    let mut command = Command::new(opener);
    command.arg(directory);
    Ok(command)
}

fn spawn_reaped(mut command: Command) -> std::io::Result<()> {
    let mut child = command.spawn()?;
    // The system opener can outlive the GUI request.
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
        .ok_or_else(|| format!("desktop opener: expected xdg-open in {paths:?}"))
}

#[cfg(target_os = "linux")]
fn directory_opener() -> Result<PathBuf, String> {
    system_opener()
}

#[cfg(target_os = "windows")]
fn system_opener() -> Result<PathBuf, String> {
    Ok(system_root()?.join(r"System32\rundll32.exe"))
}

#[cfg(target_os = "windows")]
fn directory_opener() -> Result<PathBuf, String> {
    Ok(system_root()?.join("explorer.exe"))
}

#[cfg(target_os = "windows")]
fn system_root() -> Result<PathBuf, String> {
    let root =
        std::env::var_os("SystemRoot").map_or_else(|| PathBuf::from(r"C:\Windows"), PathBuf::from);
    if !root.is_absolute() {
        return Err(format!(
            "SystemRoot {root:?}: expected an absolute Windows directory"
        ));
    }
    Ok(root)
}

#[cfg(target_arch = "wasm32")]
fn system_opener() -> Result<PathBuf, String> {
    Err("desktop opener: external process launch is unavailable in WebAssembly".into())
}

#[cfg(target_arch = "wasm32")]
fn directory_opener() -> Result<PathBuf, String> {
    system_opener()
}

#[cfg(test)]
#[test]
fn resource_browser_refuses_non_https_and_whitespace() {
    for url in ["file:///home/report", "https://", "https://example.org/a b"] {
        let error = browser_command(url).unwrap_err();
        assert!(error.contains(&format!("{url:?}")), "{error}");
    }
}

#[cfg(test)]
#[test]
fn file_manager_receives_the_directory_as_one_argument() {
    let folder = tempfile::Builder::new()
        .prefix("Part (metric) ")
        .tempdir()
        .unwrap();
    let command = directory_command(folder.path(), "file-manager".into()).unwrap();
    assert_eq!(
        command.get_args().collect::<Vec<_>>(),
        [folder.path().as_os_str()]
    );
    let missing = folder.path().join("missing");
    assert!(directory_command(&missing, "file-manager".into()).is_err());
    assert!(directory_command(Path::new("relative"), "file-manager".into()).is_err());
}
