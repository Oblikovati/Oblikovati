// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{ASSET_PROJECT, DOCUMENT_PART};
use oblikovati_model::assets::DocumentAssets;
use oblikovati_persistence::{assets::load_builtin_catalog, package::DocumentPackage};

fn part_package(assets: &DocumentAssets) -> DocumentPackage {
    let package = DocumentPackage::empty(DOCUMENT_PART, "Fixture");
    let bytes = package.with_assets(assets).unwrap().encode().unwrap();
    DocumentPackage::decode(&bytes).unwrap()
}
#[test]
fn recipe_material_preserves_unassigned_and_unresolved_intent() {
    let empty = AssetCollection::default();
    let mut assets = DocumentAssets::default();
    assert_eq!(
        recipe_material(
            &part_package(&assets).restore_assets().unwrap(),
            &empty,
            &empty
        ),
        GuiTemplateMaterial::Unassigned
    );
    assets
        .assignments
        .set_part_material("missing-steel")
        .unwrap();
    assert_eq!(
        recipe_material(
            &part_package(&assets).restore_assets().unwrap(),
            &empty,
            &empty
        ),
        GuiTemplateMaterial::Unresolved {
            id: "missing-steel".into()
        }
    );
}
#[test]
fn recipe_material_resolves_the_embedded_definition_before_project_and_builtin() {
    let catalog = load_builtin_catalog().unwrap();
    let steel = catalog.materials().get("steel").unwrap();
    let custom = steel
        .duplicate("fixture-steel", "Fixture steel", ASSET_PROJECT)
        .unwrap();
    let mut assets = DocumentAssets::default();
    assets.embed_material(&custom).unwrap();
    assets
        .assignments
        .set_part_material("fixture-steel")
        .unwrap();
    let project_steel = steel
        .duplicate("fixture-steel", "Project steel", ASSET_PROJECT)
        .unwrap();
    let mut project = AssetCollection::default();
    project.insert(project_steel);
    let actual = recipe_material(
        &part_package(&assets).restore_assets().unwrap(),
        &project,
        catalog.materials(),
    );
    assert_eq!(
        actual,
        GuiTemplateMaterial::Assigned {
            id: "fixture-steel".into(),
            name: "Fixture steel".into()
        }
    );
}

#[test]
fn recipe_material_resolves_project_then_builtin_definitions() {
    let catalog = load_builtin_catalog().unwrap();
    let steel = catalog.materials().get("steel").unwrap();
    let project_steel = steel
        .duplicate("steel", "Project steel", ASSET_PROJECT)
        .unwrap();
    let mut project = AssetCollection::default();
    project.insert(project_steel);
    let mut assets = DocumentAssets::default();
    assets.assignments.set_part_material("steel").unwrap();
    let restored = part_package(&assets).restore_assets().unwrap();
    assert_eq!(
        recipe_material(&restored, &project, catalog.materials()),
        GuiTemplateMaterial::Assigned {
            id: "steel".into(),
            name: "Project steel".into()
        }
    );
    let empty = AssetCollection::default();
    assert_eq!(
        recipe_material(&restored, &empty, catalog.materials()),
        GuiTemplateMaterial::Assigned {
            id: "steel".into(),
            name: steel.display_name()
        }
    );
}
