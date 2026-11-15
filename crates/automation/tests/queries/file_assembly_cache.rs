// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;

fn assembly_without_presenter() -> (DiscoveryWorkspace, u64, QueryGuids) {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let package = DocumentPackage::empty(DOCUMENT_ASSEMBLY, "Assembly");
    let id = files
        .create(
            Path::new("/fixtures/assembly.oad"),
            package,
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    (files, id, source)
}
fn warning(warnings: Option<Vec<SaveWarning>>) {
    let warnings = warnings.expect("expected save warnings");
    assert_eq!(warnings.len(), 1);
    assert_eq!(warnings[0].code, SAVE_WARNING_ASSEMBLY_CACHE_UNAVAILABLE);
    assert!(warnings[0].detail.contains("presentation service"));
}
#[test]
fn every_typed_save_variant_reports_missing_presentation_service() {
    let (mut files, id, mut source) = assembly_without_presenter();
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    warning(
        client
            .documents()
            .save(&SaveDocumentArgs { document: id })
            .unwrap()
            .warnings,
    );
    warning(
        client
            .documents()
            .save_as(&SaveDocumentAsArgs {
                document: id,
                new_full_document_name: "/fixtures/renamed.oad".into(),
            })
            .unwrap()
            .warnings,
    );
    warning(
        client
            .documents()
            .save_copy_as(&SaveCopyAsArgs {
                document: id,
                target_file_name: "/fixtures/copy.oad".into(),
                metadata: None,
            })
            .unwrap()
            .warnings,
    );
    for operation in ["save", "saveAs", "saveCopyAs"] {
        let args = BatchSaveArgs {
            operation: operation.into(),
            items: Some(vec![BatchSaveItem {
                document: id,
                target_file_name: format!("/fixtures/{operation}.oad"),
            }]),
        };
        let result = client.documents().batch_save(&args).unwrap();
        let item = result.results.unwrap().pop().unwrap();
        assert!(item.ok, "{}", item.error);
        warning(item.warnings);
    }
}
#[test]
fn save_on_close_reports_missing_presentation_service() {
    let (mut files, id, mut source) = assembly_without_presenter();
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .documents()
        .save(&SaveDocumentArgs { document: id })
        .unwrap();
    files
        .commands(id)
        .unwrap()
        .run(
            "Add virtual occurrence",
            |snapshot, ids| -> Result<(), ()> {
                snapshot
                    .assembly
                    .as_mut()
                    .unwrap()
                    .push(AssemblyOccurrenceRecipe {
                        session_id: ids.allocate_occurrence().unwrap(),
                        name: "Virtual".into(),
                        virtual_: true,
                        ..Default::default()
                    });
                Ok(())
            },
        )
        .unwrap();
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .documents()
        .close(&CloseDocumentArgs { id, force: false })
        .unwrap();
    warning(result.warnings);
}

#[test]
fn close_all_reports_warnings_for_each_saved_assembly() {
    let (mut files, first, mut source) = assembly_without_presenter();
    let second = files
        .create(
            Path::new("/fixtures/other.oad"),
            DocumentPackage::empty(DOCUMENT_ASSEMBLY, "Other"),
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .documents()
        .close_all(&CloseAllDocumentsArgs { force: false })
        .unwrap();
    let warnings = result.warnings.unwrap();
    assert_eq!(warnings.len(), 2);
    assert!(
        warnings
            .iter()
            .any(|item| item.document == first
                && item.code == SAVE_WARNING_ASSEMBLY_CACHE_UNAVAILABLE)
    );
    assert!(warnings.iter().any(
        |item| item.document == second && item.code == SAVE_WARNING_ASSEMBLY_CACHE_UNAVAILABLE
    ));
}
