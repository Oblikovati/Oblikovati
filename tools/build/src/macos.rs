// SPDX-License-Identifier: GPL-2.0-only
// Native dependency relocation replaces the Go cgo packaging path from aa8297d1 (#3560).

use crate::{
    packaging::{copy_file, copy_tree},
    process::{BuildResult, Runner, execute},
    tree::{all_files as source_files, relative_name, write_text},
};
use std::{collections::BTreeSet, fs, path::Path};

pub(crate) fn stage_libraries(root: &Path, runner: &dyn Runner, stage: &Path) -> BuildResult<()> {
    for (formula, name) in [
        ("vulkan-loader", "libvulkan.1.dylib"),
        ("molten-vk", "libMoltenVK.dylib"),
        ("glfw", "libglfw.3.dylib"),
    ] {
        let prefix = runner.output(root, "brew", &["--prefix", formula])?;
        copy_file(
            &Path::new(prefix.trim()).join("lib").join(name),
            &stage.join("lib").join(name),
        )?;
    }
    relocate(
        root,
        runner,
        &stage.join("bin"),
        &stage.join("lib"),
        "@executable_path/../lib",
    )
}

fn linked_paths(output: &str) -> Vec<String> {
    output
        .lines()
        .skip(1)
        .map(str::trim)
        .filter_map(|line| line.split_once(" ("))
        .map(|(path, _)| path.to_owned())
        .collect()
}

fn bundled_dependency(path: &str) -> bool {
    !path.starts_with("/usr/lib/") && !path.starts_with("/System/Library/")
}

fn relocate(
    root: &Path,
    runner: &dyn Runner,
    binaries: &Path,
    libraries: &Path,
    rpath: &str,
) -> BuildResult<()> {
    let mut pending = source_files(binaries)?;
    pending.extend(source_files(libraries)?);
    let mut seen = BTreeSet::new();
    while let Some(binary) = pending.pop() {
        if !seen.insert(binary.clone()) {
            continue;
        }
        let output = runner.output(root, "otool", &["-L", &binary.to_string_lossy()])?;
        for dependency in linked_paths(&output)
            .iter()
            .filter(|path| bundled_dependency(path))
        {
            let staged = relocate_dependency(root, runner, &binary, dependency, libraries)?;
            pending.push(staged);
        }
        set_install_paths(root, runner, &binary, libraries, rpath)?;
    }
    Ok(())
}

fn relocate_dependency(
    root: &Path,
    runner: &dyn Runner,
    binary: &Path,
    dependency: &str,
    libraries: &Path,
) -> BuildResult<std::path::PathBuf> {
    let basename = Path::new(dependency)
        .file_name()
        .ok_or_else(|| format!("dylib path {dependency:?}: expected a filename"))?;
    let destination = libraries.join(basename);
    if !destination.is_file() {
        copy_file(Path::new(dependency), &destination)?;
    }
    let replacement = format!("@rpath/{}", basename.to_string_lossy());
    execute(
        runner,
        root,
        &[
            "install_name_tool",
            "-change",
            dependency,
            &replacement,
            &binary.to_string_lossy(),
        ],
    )?;
    Ok(destination)
}

fn set_install_paths(
    root: &Path,
    runner: &dyn Runner,
    binary: &Path,
    libraries: &Path,
    rpath: &str,
) -> BuildResult<()> {
    let library = binary.starts_with(libraries);
    if library {
        let name = binary.file_name().unwrap_or_default().to_string_lossy();
        execute(
            runner,
            root,
            &[
                "install_name_tool",
                "-id",
                &format!("@rpath/{name}"),
                &binary.to_string_lossy(),
            ],
        )?;
    }
    let wanted = if library { "@loader_path" } else { rpath };
    let commands = runner.output(root, "otool", &["-l", &binary.to_string_lossy()])?;
    if !commands.contains(wanted) {
        execute(
            runner,
            root,
            &[
                "install_name_tool",
                "-add_rpath",
                wanted,
                &binary.to_string_lossy(),
            ],
        )?;
    }
    Ok(())
}

pub(crate) fn merge(
    root: &Path,
    runner: &dyn Runner,
    intel: &Path,
    arm: &Path,
    output: &Path,
) -> BuildResult<()> {
    let names = |directory: &Path| -> BuildResult<BTreeSet<String>> {
        source_files(directory)?
            .iter()
            .map(|file| relative_name(directory, file))
            .collect()
    };
    let intel_names = names(intel)?;
    if intel_names != names(arm)? {
        return Err("macOS stages: expected identical Intel and ARM file inventories".into());
    }
    copy_tree(arm, output)?;
    for name in intel_names {
        if name.starts_with("bin/") || name.starts_with("lib/") {
            execute(
                runner,
                root,
                &[
                    "lipo",
                    "-create",
                    &intel.join(&name).to_string_lossy(),
                    &arm.join(&name).to_string_lossy(),
                    "-output",
                    &output.join(&name).to_string_lossy(),
                ],
            )?;
            execute(
                runner,
                root,
                &[
                    "lipo",
                    "-verify_arch",
                    "x86_64",
                    "arm64",
                    &output.join(&name).to_string_lossy(),
                ],
            )?;
        } else if fs::read(intel.join(&name)).ok() != fs::read(arm.join(&name)).ok() {
            return Err(format!(
                "macOS resource {name}: expected identical architecture-independent bytes"
            ));
        }
    }
    Ok(())
}

pub(crate) fn assemble(
    root: &Path,
    runner: &dyn Runner,
    stage: &Path,
    version: &str,
) -> BuildResult<std::path::PathBuf> {
    crate::platform::validate_version(version)?;
    let app = root.join("dist/Oblikovati Rust.app");
    if app.exists() {
        return Err(format!("{}: expected a new app bundle", app.display()));
    }
    copy_tree(&stage.join("bin"), &app.join("Contents/MacOS"))?;
    copy_tree(&stage.join("lib"), &app.join("Contents/Frameworks"))?;
    copy_tree(
        &stage.join("share/oblikovati-rust"),
        &app.join("Contents/Resources"),
    )?;
    for name in ["LICENSE", "release-manifest.json"] {
        copy_file(
            &stage.join(name),
            &app.join("Contents/Resources").join(name),
        )?;
    }
    let epoch = runner.output(root, "git", &["show", "-s", "--format=%ct", "HEAD"])?;
    write_text(
        &app.join("Contents/Info.plist"),
        &plist(version, epoch.trim())?,
    )?;
    write_text(
        &app.join("Contents/Resources/vulkan/icd.d/MoltenVK_icd.json"),
        "{\"file_format_version\":\"1.0.0\",\"ICD\":{\"library_path\":\"../../../Frameworks/libMoltenVK.dylib\",\"api_version\":\"1.2.0\"}}\n",
    )?;
    relocate(
        root,
        runner,
        &app.join("Contents/MacOS"),
        &app.join("Contents/Frameworks"),
        "@executable_path/../Frameworks",
    )?;
    Ok(app)
}

fn plist(version: &str, epoch: &str) -> BuildResult<String> {
    let minutes = epoch
        .parse::<u64>()
        .map_err(|error| format!("commit time {epoch:?}: expected UNIX seconds: {error}"))?
        / 60;
    if minutes >= 100_000_000 {
        return Err(format!(
            "commit minutes {minutes}: expected an Apple build number below 100000000"
        ));
    }
    let build = format!(
        "{}.{}.{}",
        minutes / 10000,
        (minutes / 100) % 100,
        minutes % 100
    );
    Ok(format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>oblikovati-head</string>
<key>CFBundleIdentifier</key><string>org.oblikovati.rust</string>
<key>CFBundleName</key><string>Oblikovati Rust</string>
<key>CFBundleIconFile</key><string>icons/Oblikovati</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleShortVersionString</key><string>0.1.0</string>
<key>CFBundleVersion</key><string>{build}</string>
<key>OblikovatiProductVersion</key><string>{version}</string>
<key>LSMinimumSystemVersion</key><string>11.0</string>
<key>NSHighResolutionCapable</key><true/>
</dict></plist>
"#
    ))
}

#[cfg(test)]
#[path = "../tests/macos.rs"]
mod tests;
