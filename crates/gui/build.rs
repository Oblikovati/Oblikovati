// SPDX-License-Identifier: GPL-2.0-only
use std::{env, fs, path::Path};

fn main() {
    let root = Path::new("assets/raster");
    println!("cargo:rerun-if-changed={}", root.display());
    let mut entries = fs::read_dir(root)
        .expect("GUI icon raster directory must exist")
        .map(|entry| entry.expect("GUI icon raster entry must read"))
        .collect::<Vec<_>>();
    entries.sort_by_key(|entry| entry.file_name());
    let manifest = env::var("CARGO_MANIFEST_DIR").expect("Cargo must set the manifest directory");
    let mut source = String::from("static ICON_RASTERS: &[IconRaster] = &[\n");
    for entry in entries {
        let path = entry.path();
        if path.extension().is_none_or(|extension| extension != "rgba") {
            continue;
        }
        let name = path
            .file_stem()
            .expect("GUI icon raster must have a name")
            .to_string_lossy()
            .into_owned();
        let absolute = Path::new(&manifest).join(&path).display().to_string();
        source.push_str(&format!(
            "    IconRaster {{ name: {name:?}, rgba: include_bytes!({absolute:?}) }},\n"
        ));
    }
    source.push_str("];\n");
    let output =
        Path::new(&env::var("OUT_DIR").expect("Cargo must set OUT_DIR")).join("icon_rasters.rs");
    fs::write(output, source).expect("GUI icon raster registry must write");
}
