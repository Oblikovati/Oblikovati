// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    packaging::copy_file,
    platform::Platform,
    process::{BuildResult, Runner},
    tree::all_files as source_files,
};
use std::{collections::BTreeSet, path::Path};

fn system_dll(name: &str) -> bool {
    let lower = name.to_ascii_lowercase();
    lower.starts_with("api-ms-win-")
        || lower.starts_with("ext-ms-win-")
        || [
            "kernel32.dll",
            "ntdll.dll",
            "user32.dll",
            "advapi32.dll",
            "gdi32.dll",
            "shell32.dll",
            "ole32.dll",
            "oleaut32.dll",
            "ws2_32.dll",
            "bcrypt.dll",
            "crypt32.dll",
            "secur32.dll",
            "userenv.dll",
            "dbghelp.dll",
            "shlwapi.dll",
            "rpcrt4.dll",
            "winmm.dll",
            "imm32.dll",
            "comdlg32.dll",
            "comctl32.dll",
            "dwmapi.dll",
            "setupapi.dll",
            "cfgmgr32.dll",
            "shcore.dll",
            "uxtheme.dll",
            "version.dll",
            "normaliz.dll",
            "ucrtbase.dll",
            "vulkan-1.dll",
        ]
        .contains(&lower.as_str())
}

fn imports(output: &str) -> Vec<String> {
    output
        .lines()
        .filter_map(|line| line.trim().strip_prefix("Name: "))
        .filter(|name| name.to_ascii_lowercase().ends_with(".dll"))
        .map(str::to_owned)
        .collect()
}

pub(crate) fn bundle(
    root: &Path,
    runner: &dyn Runner,
    platform: Platform,
    stage: &Path,
) -> BuildResult<()> {
    let sysroot = runner.output(root, "rustc", &["--print", "sysroot"])?;
    let readobj = Path::new(sysroot.trim())
        .join("lib/rustlib")
        .join(platform.target)
        .join("bin/llvm-readobj.exe");
    let mut pending = source_files(&stage.join("bin"))?;
    let mut copied = BTreeSet::new();
    while let Some(binary) = pending.pop() {
        let output = runner.output(
            root,
            &readobj.to_string_lossy(),
            &["--coff-imports", &binary.to_string_lossy()],
        )?;
        for name in imports(&output)
            .into_iter()
            .filter(|name| !system_dll(name))
        {
            if !copied.insert(name.to_ascii_lowercase()) {
                continue;
            }
            let source = root
                .join("target")
                .join(platform.target)
                .join("release")
                .join(&name);
            let destination = stage.join("bin").join(&name);
            copy_file(&source, &destination)?;
            pending.push(destination);
        }
    }
    Ok(())
}

#[cfg(test)]
#[path = "../tests/windows.rs"]
mod tests;
