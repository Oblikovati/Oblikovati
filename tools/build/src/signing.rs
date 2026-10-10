// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    process::{BuildResult, Runner, execute},
    tree::all_files,
};
use std::path::Path;

pub(crate) struct SigningIdentity {
    pub(crate) developer_id: String,
    pub(crate) keychain: String,
}

pub(crate) fn credentials() -> BuildResult<SigningIdentity> {
    let get = |key: &str| {
        std::env::var(key).ok().filter(|value| !value.is_empty())
        .ok_or_else(|| format!("{key}: expected macOS signing credentials; publication cannot skip this platform"))
    };
    Ok(SigningIdentity {
        developer_id: get("DEVELOPER_ID_APP")?,
        keychain: get("OBLIKOVATI_SIGNING_KEYCHAIN")?,
    })
}

fn sign_code(
    root: &Path,
    runner: &dyn Runner,
    path: &Path,
    identity: &SigningIdentity,
) -> BuildResult<()> {
    execute(
        runner,
        root,
        &[
            "codesign",
            "--force",
            "--options",
            "runtime",
            "--timestamp",
            "--sign",
            &identity.developer_id,
            &path.to_string_lossy(),
        ],
    )
}

pub(crate) fn sign_tree(
    root: &Path,
    runner: &dyn Runner,
    directory: &Path,
    identity: &SigningIdentity,
) -> BuildResult<()> {
    for file in all_files(directory)? {
        let name = file.file_name().unwrap_or_default().to_string_lossy();
        if name.ends_with(".dylib")
            || ["oblikovati-cli", "oblikovati-head"].contains(&name.as_ref())
        {
            sign_code(root, runner, &file, identity)?;
        }
    }
    Ok(())
}

fn notarize(
    root: &Path,
    runner: &dyn Runner,
    archive: &Path,
    identity: &SigningIdentity,
) -> BuildResult<()> {
    // Apple credentials are stored by CI in a temporary keychain, never in command logs.
    execute(
        runner,
        root,
        &[
            "xcrun",
            "notarytool",
            "submit",
            &archive.to_string_lossy(),
            "--keychain-profile",
            "oblikovati-rust",
            "--keychain",
            &identity.keychain,
            "--wait",
        ],
    )
}

pub(crate) fn sign_stage(
    root: &Path,
    runner: &dyn Runner,
    stage: &Path,
    identity: &SigningIdentity,
) -> BuildResult<()> {
    sign_tree(root, runner, stage, identity)?;
    let archive = root.join("target/notarize-cli.zip");
    execute(
        runner,
        root,
        &[
            "ditto",
            "-c",
            "-k",
            "--keepParent",
            &stage.to_string_lossy(),
            &archive.to_string_lossy(),
        ],
    )?;
    notarize(root, runner, &archive, identity)
}

pub(crate) fn sign_app(
    root: &Path,
    runner: &dyn Runner,
    app: &Path,
    output: &Path,
    identity: &SigningIdentity,
) -> BuildResult<()> {
    sign_tree(root, runner, app, identity)?;
    sign_code(root, runner, app, identity)?;
    execute(
        runner,
        root,
        &[
            "codesign",
            "--verify",
            "--deep",
            "--strict",
            &app.to_string_lossy(),
        ],
    )?;
    execute(
        runner,
        root,
        &[
            "ditto",
            "-c",
            "-k",
            "--keepParent",
            &app.to_string_lossy(),
            &output.to_string_lossy(),
        ],
    )?;
    notarize(root, runner, output, identity)?;
    execute(
        runner,
        root,
        &["xcrun", "stapler", "staple", &app.to_string_lossy()],
    )?;
    execute(
        runner,
        root,
        &["xcrun", "stapler", "validate", &app.to_string_lossy()],
    )?;
    execute(
        runner,
        root,
        &[
            "ditto",
            "-c",
            "-k",
            "--keepParent",
            &app.to_string_lossy(),
            &output.to_string_lossy(),
        ],
    )
}

#[cfg(test)]
#[path = "../tests/signing.rs"]
mod tests;
