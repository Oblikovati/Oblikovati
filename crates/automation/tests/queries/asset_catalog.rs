// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_automation::{ASSET_QUERY_METHODS, FileHost, file_host_methods};
use oblikovati_model::assets::BuiltinAssetCatalog;
use oblikovati_session::files::WorkspaceAssetCatalogSource;
use std::sync::{
    Arc,
    atomic::{AtomicUsize, Ordering},
};
struct ObservedCatalogSource {
    loads: Arc<AtomicUsize>,
    refuse: bool,
}
impl WorkspaceAssetCatalogSource for ObservedCatalogSource {
    fn load_catalog(&self) -> Result<BuiltinAssetCatalog, String> {
        self.loads.fetch_add(1, Ordering::SeqCst);
        if self.refuse {
            return Err("fixture catalog broken.yaml: expected complete catalog records".into());
        }
        oblikovati_persistence::assets::load_builtin_catalog()
    }
}
fn observed_catalog_files(refuse: bool) -> (DiscoveryWorkspace, Arc<AtomicUsize>) {
    let loads = Arc::new(AtomicUsize::new(0));
    let source = ObservedCatalogSource {
        loads: Arc::clone(&loads),
        refuse,
    };
    let files = DiscoveryWorkspace::new_with_asset_catalog_source(
        QueryObserver::default(),
        DiscoveryFiles::default(),
        Default::default(),
        std::boxed::Box::new(source),
    );
    (files, loads)
}
#[test]
fn typed_catalog_queries_work_without_an_active_document_and_load_only_once() {
    let (files, loads) = observed_catalog_files(false);
    assert_eq!(loads.load(Ordering::SeqCst), 0);
    let client = Client::new(FileQueryHost::new(&files));
    let materials = client.materials().list().unwrap().materials.unwrap();
    let appearances = client.appearances().list().unwrap().appearances.unwrap();
    assert_eq!(materials.len(), 126);
    assert_eq!(appearances.len(), 94);
    let steel = client
        .materials()
        .get(&AssetRefArgs { id: "steel".into() })
        .unwrap();
    assert_eq!(steel.isotropy_class, "isotropic");
    assert_eq!(steel.density, 7.87);
    assert_eq!(
        materials.iter().find(|asset| asset.id == "steel").unwrap(),
        &steel
    );
    let visual = client
        .appearances()
        .get(&AssetRefArgs {
            id: steel.appearance_id,
        })
        .unwrap();
    assert_eq!(
        appearances
            .iter()
            .find(|asset| asset.id == visual.id)
            .unwrap(),
        &visual
    );
    assert_eq!(loads.load(Ordering::SeqCst), 1);
    assert!(
        client
            .documents()
            .list()
            .unwrap()
            .documents
            .unwrap()
            .is_empty()
    );
    assert!(files.files().packages.borrow().is_empty());
}
#[test]
fn mutable_file_host_forwards_the_same_catalog_without_starting_a_transaction() {
    let (mut files, loads) = observed_catalog_files(false);
    let mut guids = QueryGuids(0);
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    assert_eq!(
        client.materials().list().unwrap().materials.unwrap().len(),
        126
    );
    assert_eq!(
        client
            .appearances()
            .get(&AssetRefArgs {
                id: "default".into()
            })
            .unwrap()
            .display_name,
        "Default"
    );
    assert!(
        client
            .documents()
            .list()
            .unwrap()
            .documents
            .unwrap()
            .is_empty()
    );
    assert_eq!(loads.load(Ordering::SeqCst), 1);
}
#[test]
fn unknown_assets_and_malformed_get_requests_are_named_without_loading_on_bad_arguments() {
    let (files, loads) = observed_catalog_files(false);
    let host = FileQueryHost::new(&files);
    for method in [METHOD_MATERIALS_GET, METHOD_APPEARANCES_GET] {
        assert!(
            host.call(method, b"[")
                .unwrap_err()
                .to_string()
                .contains(method)
        );
    }
    assert_eq!(loads.load(Ordering::SeqCst), 0);
    let client = Client::new(host);
    let unknown = AssetRefArgs {
        id: "missing-asset".into(),
    };
    assert!(
        client
            .materials()
            .get(&unknown)
            .unwrap_err()
            .to_string()
            .contains("missing-asset")
    );
    assert!(
        client
            .appearances()
            .get(&unknown)
            .unwrap_err()
            .to_string()
            .contains("missing-asset")
    );
    assert_eq!(loads.load(Ordering::SeqCst), 1);
}
#[test]
fn failed_catalog_admission_is_retained_and_does_not_block_document_queries() {
    let (files, loads) = observed_catalog_files(true);
    let client = Client::new(FileQueryHost::new(&files));
    assert!(
        client
            .materials()
            .list()
            .unwrap_err()
            .to_string()
            .contains("broken.yaml")
    );
    assert!(
        client
            .appearances()
            .list()
            .unwrap_err()
            .to_string()
            .contains("broken.yaml")
    );
    assert!(
        client
            .documents()
            .list()
            .unwrap()
            .documents
            .unwrap()
            .is_empty()
    );
    assert_eq!(loads.load(Ordering::SeqCst), 1);
}
#[test]
fn script_inventory_advertises_asset_commands_and_verified_assignments() {
    let methods = file_host_methods();
    assert!(
        ASSET_QUERY_METHODS
            .iter()
            .chain(oblikovati_automation::ASSET_COMMAND_METHODS.iter())
            .all(|method| methods.iter().any(|value| value == method))
    );
    for assignment in [METHOD_MODEL_ASSIGN_MATERIAL, METHOD_MODEL_ASSIGN_APPEARANCE] {
        assert!(methods.iter().any(|method| method == assignment));
    }
}

#[path = "document_assets.rs"]
mod document_assets;

#[path = "project_assets.rs"]
mod project_assets;
