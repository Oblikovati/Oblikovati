// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
use std::{
    path::Path,
    process::{Command, Output},
};
fn cli(arguments: &[&str]) -> Output {
    Command::new(env!("CARGO_BIN_EXE_oblikovati-cli"))
        .args(arguments)
        .output()
        .unwrap()
}
fn write_source(directory: &Path, source: &str) -> std::path::PathBuf {
    let path = directory.join("model.lua");
    std::fs::write(&path, source).unwrap();
    path
}
#[test]
fn executable_script_run_creates_a_real_native_document_that_info_can_read() {
    let directory = tempfile::tempdir().unwrap();
    let script = write_source(
        directory.path(),
        r#"oblikovati.documents.create{type="part",name="Lua native"}; oblikovati.parameters.add{name="width",expression="4 cm"}; print("done")"#,
    );
    let saved = directory.path().join("lua.opd");
    let output = cli(&[
        "script",
        "run",
        script.to_str().unwrap(),
        "--save",
        saved.to_str().unwrap(),
    ]);
    verify_native_result(&output, &saved);
}
fn verify_native_result(output: &Output, saved: &Path) {
    assert!(
        output.status.success(),
        "{}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(String::from_utf8_lossy(&output.stdout).contains("done"));
    assert!(saved.is_file());
    let info = cli(&["info", saved.to_str().unwrap()]);
    assert!(info.status.success());
    assert!(String::from_utf8_lossy(&info.stdout).contains("Lua native"));
}

#[test]
fn failed_script_returns_nonzero_and_never_writes_the_requested_save() {
    let directory = tempfile::tempdir().unwrap();
    let script = write_source(
        directory.path(),
        r#"oblikovati.documents.create{type="part",name="script part"}; error("intentional failure value")"#,
    );
    let saved = directory.path().join("failed.opd");
    verify_failed_native_save(&script, &saved);
}
fn verify_failed_native_save(script: &Path, saved: &Path) {
    let output = cli(&[
        "script",
        "run",
        script.to_str().unwrap(),
        "--save",
        saved.to_str().unwrap(),
    ]);
    assert!(!output.status.success());
    assert!(
        String::from_utf8_lossy(&output.stderr).contains("intentional failure value"),
        "stderr: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(!saved.exists());
}

#[test]
fn native_source_io_refuses_missing_non_utf8_and_oversize_files() {
    verify_invalid_native_sources();
}
fn verify_invalid_native_sources() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("missing.lua");
    verify_source_decline(&path);
    std::fs::write(&path, [0xff]).unwrap();
    verify_source_decline(&path);
    std::fs::write(&path, vec![b' '; (1 << 20) + 1]).unwrap();
    verify_source_decline(&path);
}

fn verify_source_decline(path: &Path) {
    assert!(
        !cli(&["script", "run", path.to_str().unwrap()])
            .status
            .success()
    );
}

#[test]
fn executable_lua_reads_physical_and_visual_catalogs_without_a_document() {
    let directory = tempfile::tempdir().unwrap();
    let script = write_source(
        directory.path(),
        r#"local m=oblikovati.materials.list().materials;assert(#m==126);local a=oblikovati.appearances.list().appearances;assert(#a==94);local s=oblikovati.materials.get{id='steel'};assert(s.isotropyClass=='isotropic');assert(s.density==7.87);local v=oblikovati.appearances.get{id=s.appearanceId};assert(v.id==s.appearanceId);assert(#oblikovati.documents.list().documents==0);print('catalog-ready')"#,
    );
    let output = cli(&["script", "run", script.to_str().unwrap()]);
    assert!(
        output.status.success(),
        "{}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(String::from_utf8_lossy(&output.stdout).contains("catalog-ready"));
}

#[test]
fn executable_lua_reads_embedded_assets_after_document_load_without_the_project() {
    let directory = tempfile::tempdir().unwrap();
    let document = directory.path().join("portable.opd");
    std::fs::write(&document, portable_asset_package().encode().unwrap()).unwrap();
    let script = write_source(
        directory.path(),
        "assert(#oblikovati.materials.list().materials==127);local p=oblikovati.materials.get{id='portable'};assert(p.source=='document');assert(p.displayName=='Portable');local a=oblikovati.appearances.get{id='portable-look'};assert(a.source=='document');print('portable-assets')",
    );
    let output = cli(&[
        "script",
        "run",
        script.to_str().unwrap(),
        "--doc",
        document.to_str().unwrap(),
    ]);
    assert!(
        output.status.success(),
        "{}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(String::from_utf8_lossy(&output.stdout).contains("portable-assets"));
}
fn portable_asset_package() -> oblikovati_persistence::package::DocumentPackage {
    use oblikovati_api::types::ASSET_PROJECT;
    use oblikovati_model::assets::DocumentAssets;
    let catalog = oblikovati_persistence::assets::load_builtin_catalog().unwrap();
    let mut assets = DocumentAssets::default();
    assets
        .embed_material(
            &catalog
                .materials()
                .get("steel")
                .unwrap()
                .duplicate("portable", "Portable", ASSET_PROJECT)
                .unwrap(),
        )
        .unwrap();
    assets
        .embed_appearance(
            &catalog
                .appearances()
                .get("steel")
                .unwrap()
                .duplicate("portable-look", "Portable look", ASSET_PROJECT)
                .unwrap(),
        )
        .unwrap();
    oblikovati_persistence::package::DocumentPackage::empty(
        oblikovati_api::types::DOCUMENT_PART,
        "Portable part",
    )
    .with_assets(&assets)
    .unwrap()
}

#[test]
fn executable_lua_loads_and_saves_a_legacy_appearance_as_grouped_data() {
    let directory = tempfile::tempdir().unwrap();
    let original = directory.path().join("legacy.opd");
    let saved = directory.path().join("upgraded.opd");
    std::fs::write(&original,b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  materials:\n    appearances:\n      - id: old-look\n        name: Old look\n        albedo: '#804020'\n        metallic: 0.3\n        roughness: 0.6\n        opacity: 0.75\n").unwrap();
    let source = write_source(
        directory.path(),
        "local a=oblikovati.appearances.get{id='old-look'};assert(a.base.metalness>0.29 and a.base.metalness<0.31);assert(a.geometry.opacity==0.75);assert(a.emission.luminance==0);print('legacy-loaded')",
    );
    let output = cli(&[
        "script",
        "run",
        source.to_str().unwrap(),
        "--doc",
        original.to_str().unwrap(),
        "--save",
        saved.to_str().unwrap(),
    ]);
    assert!(
        output.status.success(),
        "{}",
        String::from_utf8_lossy(&output.stderr)
    );
    let bytes = std::fs::read(&saved).unwrap();
    let package = oblikovati_persistence::package::DocumentPackage::decode(&bytes).unwrap();
    let appearance = package
        .restore_assets()
        .unwrap()
        .appearances()
        .get("old-look")
        .unwrap()
        .info();
    assert_eq!(appearance.geometry.opacity, 0.75);
    let yaml = String::from_utf8(bytes).unwrap();
    assert!(!yaml.contains("albedo:"));
    assert!(yaml.contains("base:"));
}

#[test]
fn executable_lua_edits_and_saves_large_physical_scalars_and_visual_groups() {
    let directory = tempfile::tempdir().unwrap();
    let original = directory.path().join("portable.opd");
    let saved = directory.path().join("edited.opd");
    std::fs::write(&original, portable_asset_package().encode().unwrap()).unwrap();
    let script = write_source(
        directory.path(),
        "local p=oblikovati.materials.get{id='portable'};p.electrical.resistivity=1e16;p.density=4.25;local r=oblikovati.materials.update(p);assert(r.electrical.resistivity==1e16);local a=oblikovati.appearances.get{id='portable-look'};a.coat.weight=0.5;assert(oblikovati.appearances.update(a).coat.weight==0.5);assert(not pcall(oblikovati.documents.save,{document=9007199254740993}));print('edited-assets')",
    );
    let output = cli(&[
        "script",
        "run",
        script.to_str().unwrap(),
        "--doc",
        original.to_str().unwrap(),
        "--save",
        saved.to_str().unwrap(),
    ]);
    assert!(
        output.status.success(),
        "{}",
        String::from_utf8_lossy(&output.stderr)
    );
    let package =
        oblikovati_persistence::package::DocumentPackage::decode(&std::fs::read(saved).unwrap())
            .unwrap();
    let assets = package.restore_assets().unwrap();
    let material = assets.materials().get("portable").unwrap().info();
    assert_eq!(material.electrical.resistivity, 1e16);
    assert_eq!(material.density, 4.25);
    assert_eq!(
        assets
            .appearances()
            .get("portable-look")
            .unwrap()
            .info()
            .coat
            .weight,
        0.5
    );
}
