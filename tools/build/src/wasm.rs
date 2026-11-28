// SPDX-License-Identifier: GPL-2.0-only
//! Multi-threaded `wasm32-unknown-unknown` (+atomics) check, build, and packaging tasks.

use crate::{
    distribution::new_manifest,
    packaging::{copy_file, copy_tree},
    platform::validate_version,
    process::{BuildResult, Runner},
    release_config,
    tree::{read_text, write_text},
};
use std::{fs, path::Path};

/// Target triple for the browser WebAssembly + WebGPU platform.
pub(crate) const WASM_TARGET: &str = "wasm32-unknown-unknown";

/// Relative path to the isolated nightly toolchain pin used for `wasm32 +atomics`.
pub(crate) const WASM_TOOLCHAIN_PATH: &str = "tools/build/wasm-toolchain.toml";

/// Workspace crates that remain native desktop/host-only and are excluded from `wasm32`.
pub(crate) const NATIVE_ONLY_CRATES: &[&str] = &[
    "oblikovati-build",
    "oblikovati-cli",
    "oblikovati-gui",
    "oblikovati-head",
];

/// RUSTFLAGS for checking all WASM-eligible workspace crates with WebAssembly Threads.
pub(crate) const WASM_CHECK_RUSTFLAGS: &str =
    "-C target-feature=+atomics,+bulk-memory,+mutable-globals --cfg getrandom_backend=\"wasm_js\"";

/// RUSTFLAGS for linking `oblikovati-web` with shared `WebAssembly.Memory` for Web Workers.
pub(crate) const WASM_BUILD_RUSTFLAGS: &str = "-C target-feature=+atomics,+bulk-memory,+mutable-globals --cfg getrandom_backend=\"wasm_js\" -C link-arg=--shared-memory -C link-arg=--max-memory=1073741824 -C link-arg=--import-memory -C link-arg=--export=__wasm_init_tls -C link-arg=--export=__tls_size -C link-arg=--export=__tls_align -C link-arg=--export=__tls_base";

/// Read the pinned nightly toolchain channel from `tools/build/wasm-toolchain.toml`.
/// Example: `let channel = wasm::pinned_channel(root)?;`.
pub(crate) fn pinned_channel(root: &Path) -> BuildResult<String> {
    let path = root.join(WASM_TOOLCHAIN_PATH);
    let text = read_text(&path)?;
    parse_channel(&text).ok_or_else(|| {
        format!("{WASM_TOOLCHAIN_PATH}: expected a non-empty `channel = \"...\"` entry")
    })
}

fn parse_channel(toml: &str) -> Option<String> {
    for line in toml.lines() {
        let trimmed = line.trim();
        if let Some(rest) = trimmed.strip_prefix("channel") {
            let after_eq = rest.trim_start().strip_prefix('=')?.trim();
            let unquoted = after_eq.trim_matches('"').trim_matches('\'');
            if !unquoted.is_empty() {
                return Some(unquoted.to_owned());
            }
        }
    }
    None
}

/// Check every WASM-eligible workspace crate against `wasm32-unknown-unknown` (+atomics).
pub(crate) fn check(root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    let channel = pinned_channel(root)?;
    let toolchain = format!("+{channel}");
    let mut args = vec![
        toolchain.as_str(),
        "check",
        "--locked",
        "--target",
        WASM_TARGET,
        "-Z",
        "build-std=panic_abort,std",
        "--workspace",
    ];
    for excluded in NATIVE_ONLY_CRATES {
        args.push("--exclude");
        args.push(excluded);
    }
    runner.stream_env(root, "cargo", &args, &[("RUSTFLAGS", WASM_CHECK_RUSTFLAGS)])
}

/// Build `oblikovati-web` in release mode and generate `web/pkg/` with `wasm-bindgen`.
pub(crate) fn build(root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    let channel = pinned_channel(root)?;
    let toolchain = format!("+{channel}");
    let cargo_args = [
        toolchain.as_str(),
        "build",
        "--locked",
        "--release",
        "--target",
        WASM_TARGET,
        "-Z",
        "build-std=panic_abort,std",
        "-p",
        "oblikovati-web",
    ];
    runner.stream_env(
        root,
        "cargo",
        &cargo_args,
        &[
            ("RUSTFLAGS", WASM_BUILD_RUSTFLAGS),
            ("CARGO_PROFILE_RELEASE_DEBUG", "0"),
            ("CARGO_PROFILE_RELEASE_STRIP", "symbols"),
        ],
    )?;

    let out_dir = root.join("web/pkg");
    fs::create_dir_all(&out_dir)
        .map_err(|error| format!("create {}: {error}", out_dir.display()))?;
    let wasm_path = root
        .join("target")
        .join(WASM_TARGET)
        .join("release")
        .join("oblikovati_web.wasm");
    let out_dir_str = out_dir.to_string_lossy();
    let wasm_path_str = wasm_path.to_string_lossy();
    runner.stream_env(
        root,
        "wasm-bindgen",
        &["--target", "web", "--out-dir", &out_dir_str, &wasm_path_str],
        &[],
    )
}

/// Build and package the static WebAssembly + WebGPU bundle for `version`.
pub(crate) fn package(root: &Path, runner: &dyn Runner, version: &str) -> BuildResult<()> {
    validate_version(version)?;
    build(root, runner)?;
    let config = release_config::load(root)?;
    let manifest = new_manifest(root, runner, config, version)?;
    let stage = root.join("target/package").join(WASM_TARGET);
    if stage.exists() {
        fs::remove_dir_all(&stage)
            .map_err(|error| format!("clear web stage {}: {error}", stage.display()))?;
    }
    for asset in ["index.html", "styles.css", "main.js", "orchestrator.js"] {
        copy_file(&root.join("web").join(asset), &stage.join(asset))?;
    }
    copy_tree(&root.join("web/pkg"), &stage.join("pkg"))?;
    copy_file(&root.join("LICENSE"), &stage.join("LICENSE"))?;
    write_text(
        &stage.join("release-manifest.json"),
        &serde_json::to_string_pretty(&manifest).map_err(|error| error.to_string())?,
    )?;
    let output = root.join("dist").join(format!(
        "oblikovati-rust-web-{version}-wasm32-webgpu.tar.gz"
    ));
    crate::archive::create(&stage, &output)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{process::tests::FakeRunner, tree::tests::FixtureTree};

    #[test]
    fn parses_pinned_channel_and_rejects_missing_channel() {
        let tree = FixtureTree::new();
        tree.put(
            WASM_TOOLCHAIN_PATH,
            "[toolchain]\nchannel = \"nightly-2026-09-25\"\n",
        );
        assert_eq!(pinned_channel(tree.root()).unwrap(), "nightly-2026-09-25");

        tree.put(WASM_TOOLCHAIN_PATH, "[toolchain]\nprofile = \"minimal\"\n");
        assert!(pinned_channel(tree.root()).is_err());
    }

    #[test]
    fn wasm_check_and_build_invoke_expected_commands() {
        let tree = FixtureTree::new();
        tree.put(
            WASM_TOOLCHAIN_PATH,
            "[toolchain]\nchannel = \"nightly-2026-09-25\"\n",
        );
        let runner = FakeRunner::default();
        runner.replies.borrow_mut().extend([
            Ok(String::new()),
            Ok(String::new()),
            Ok(String::new()),
        ]);
        check(tree.root(), &runner).unwrap();
        build(tree.root(), &runner).unwrap();
        let calls = runner.calls.borrow();
        assert_eq!(calls.len(), 3);
        assert!(calls[0].join(" ").contains("+nightly-2026-09-25"));
        assert!(calls[0].join(" ").contains("--exclude oblikovati-head"));
        assert!(calls[1].join(" ").contains("-p oblikovati-web"));
        assert!(calls[2].join(" ").contains("wasm-bindgen"));
    }

    #[test]
    fn wasm_package_stages_assets_and_creates_archive() {
        let tree = FixtureTree::new();
        tree.put(
            WASM_TOOLCHAIN_PATH,
            "[toolchain]\nchannel = \"nightly-2026-09-25\"\n",
        );
        tree.put("release.toml", include_str!("../../../release.toml"));
        tree.put("LICENSE", "GPL-2.0-only\n");
        for asset in ["index.html", "styles.css", "main.js", "orchestrator.js"] {
            tree.put(&format!("web/{asset}"), "content\n");
        }
        tree.put(
            "web/pkg/oblikovati_web.js",
            "export default function() {}\n",
        );
        let runner = FakeRunner::default();
        runner.replies.borrow_mut().extend([
            Ok(String::new()),
            Ok(String::new()),
            Ok("a".repeat(40)),
        ]);
        package(tree.root(), &runner, "0.166.0.0").unwrap();
        assert!(
            tree.root()
                .join("dist/oblikovati-rust-web-0.166.0.0-wasm32-webgpu.tar.gz")
                .is_file()
        );
        assert!(package(tree.root(), &runner, "bad-version").is_err());
    }
}
