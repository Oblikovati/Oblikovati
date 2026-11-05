// SPDX-License-Identifier: GPL-2.0-only

mod build_identity;
use std::{env, fs, path::PathBuf};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("cargo:rerun-if-changed=build_identity.rs");
    println!("cargo:rerun-if-env-changed=OBLIKOVATI_BUILD_MANIFEST");
    println!("cargo:rerun-if-env-changed=SOURCE_DATE_EPOCH");
    let manifest = match env::var_os("OBLIKOVATI_BUILD_MANIFEST") {
        Some(path) => {
            println!("cargo:rerun-if-changed={}", PathBuf::from(&path).display());
            fs::read_to_string(path)?
        }
        None => format!(
            r#"{{"format":1,"version":"{}-dev","application_commit":"unknown"}}"#,
            env!("CARGO_PKG_VERSION")
        ),
    };
    build_identity::BuildIdentity::parse(&manifest)?;
    fs::write(
        PathBuf::from(
            env::var_os("OUT_DIR").ok_or("OUT_DIR: expected Cargo build output directory")?,
        )
        .join("build-manifest.json"),
        manifest,
    )?;
    Ok(())
}
