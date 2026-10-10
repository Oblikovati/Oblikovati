// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_model::quantity::Quantity;

fn linked_files() -> (DiscoveryWorkspace, u64, u64, QueryGuids) {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let source_id = files.create(Path::new("/source.opd"), DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 2\nmodel:\n  parameters:\n    - {name: width, kind: user, expression: '2 cm'}\n    - {name: height, kind: user, expression: '3 cm'}\n    - {name: computed, kind: model, expression: '4 cm'}\n    - {name: title, kind: user, valueType: text, text: 'note'}\n").unwrap(), QueryObserver::default(), "test", &mut source).unwrap().session_id;
    let target = files.create(Path::new("/target.opd"), DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters: [{name: width, kind: derived, value: 2, unit: length}]\n  derivedParameterTables: [{id: 7, sourceDocument: /source.opd, linked: [width], ownedByFeature: true}, {id: 8, sourceDocument: absent.opd, linked: []}]\n").unwrap(), QueryObserver::default(), "test", &mut source).unwrap().session_id;
    (files, source_id, target, source)
}

#[test]
fn table_queries_resolve_live_candidates_without_changing_focus_or_recipe() {
    let (mut files, source_id, target, _) = linked_files();
    let list = Client::new(FileQueryHost::new(&files))
        .parameters()
        .list_derived_tables()
        .unwrap()
        .tables
        .unwrap();
    assert_eq!(
        list.iter().map(|table| table.id).collect::<Vec<_>>(),
        [7, 8]
    );
    assert_eq!(
        list[0].available,
        Some(vec!["width".into(), "height".into()])
    );
    assert_eq!(list[0].linked, Some(vec!["width".into()]));
    assert_eq!(
        list[0].references.as_ref().unwrap()[0].source_parameter,
        "width"
    );
    assert!(list[0].has_reference_component);
    assert_eq!(list[0].reference_component, "/source.opd");
    assert!(list[0].health.contains("not been synchronized"));
    assert_eq!(list[1].available, None);
    assert!(list[1].health.contains("unavailable"));
    assert_eq!(files.workspace().active_document(), Some(target));
    assert!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
    files
        .commands(source_id)
        .unwrap()
        .run("Rename", |snapshot, _| {
            snapshot.parameters.rename(2, "tall")
        })
        .unwrap();
    let refreshed = Client::new(FileQueryHost::new(&files))
        .parameters()
        .list_derived_tables()
        .unwrap()
        .tables
        .unwrap();
    assert_eq!(
        refreshed[0].available,
        Some(vec!["width".into(), "tall".into()])
    );
    files.close_discarding_changes(source_id).unwrap();
    assert_eq!(files.resolve_parameter_source("/source.opd").unwrap(), None);
}

#[test]
fn derived_health_reports_refresh_results_and_unavailable_sources() {
    let (mut files, _, target, _) = linked_files();
    for (values, expected) in [
        (
            Some(BTreeMap::from([(
                "width".into(),
                Quantity::new(2.0, Dimension::Length),
            )])),
            "",
        ),
        (Some(BTreeMap::new()), "removed at source: [width]"),
        (None, "source document /source.opd is unavailable"),
    ] {
        files
            .commands(target)
            .unwrap()
            .run("Refresh", |snapshot, _| {
                snapshot.parameters.sync_derived_table(7, values.as_ref())
            })
            .unwrap();
        let list = Client::new(FileQueryHost::new(&files))
            .parameters()
            .list_derived_tables()
            .unwrap()
            .tables
            .unwrap();
        assert_eq!(list[0].health, expected);
    }
}

#[test]
fn source_lookup_refuses_duplicate_unbound_names_and_excludes_drawings() {
    let (mut files, source_id, _, mut source) = linked_files();
    assert_eq!(
        files.resolve_parameter_source("/source.opd").unwrap(),
        Some(source_id)
    );
    assert_eq!(files.resolve_parameter_source("source").unwrap(), None);
    assert_eq!(files.resolve_parameter_source("").unwrap(), None);
    let package = DocumentPackage::empty(DOCUMENT_PART, "Untitled");
    let id = files
        .create_unbound(
            package.clone(),
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    assert_eq!(
        files.resolve_parameter_source("Untitled").unwrap(),
        Some(id)
    );
    let _created = files
        .create_unbound(package, QueryObserver::default(), "test", &mut source)
        .unwrap();
    assert!(
        files
            .resolve_parameter_source("Untitled")
            .unwrap_err()
            .to_string()
            .contains("unique")
    );
    let _created = files
        .create_unbound(
            DocumentPackage::empty(DOCUMENT_DRAWING, "Sheet"),
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap();
    assert_eq!(files.resolve_parameter_source("Sheet").unwrap(), None);
    assert!(
        Client::new(FileQueryHost::new(&files))
            .parameters()
            .list_derived_tables()
            .is_err()
    );
    assert!(files.files().packages.borrow().is_empty());
}

#[test]
fn no_active_owner_is_refused_and_empty_tables_use_canonical_null() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    assert!(
        Client::new(FileQueryHost::new(&files))
            .parameters()
            .list_derived_tables()
            .is_err()
    );
    let _created = files
        .create_unbound(
            DocumentPackage::empty(DOCUMENT_ASSEMBLY, "Assembly"),
            QueryObserver::default(),
            "test",
            &mut QueryGuids(0),
        )
        .unwrap();
    assert_eq!(
        Client::new(FileQueryHost::new(&files))
            .parameters()
            .list_derived_tables()
            .unwrap()
            .tables,
        None
    );
}

#[test]
fn source_values_use_target_scale_and_exclude_failed_or_nonuser_values() {
    let (mut files, source_id, _, mut source) = linked_files();
    let target = files
        .create_unbound(
            DocumentPackage::decode(
                b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  units: {workingScale: '0.5'}\n",
            )
            .unwrap(),
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    let values = files
        .derived_source_values("/source.opd", target)
        .unwrap()
        .unwrap();
    assert_eq!(
        values.keys().cloned().collect::<Vec<_>>(),
        ["height", "width"]
    );
    assert_eq!(values["width"], Quantity::new(4.0, Dimension::Length));
    files
        .commands(source_id)
        .unwrap()
        .run("Break source", |snapshot, _| {
            snapshot.parameters.set_expression(1, "missing + 1 cm")
        })
        .unwrap();
    let values = files
        .derived_source_values("/source.opd", target)
        .unwrap()
        .unwrap();
    assert!(!values.contains_key("width"));
    assert_eq!(values["height"].value, 6.0);
    assert_eq!(files.derived_source_values("absent", target).unwrap(), None);
    assert!(files.derived_source_values("/source.opd", 999).is_err());
    assert_eq!(files.workspace().active_document(), Some(target));
    assert!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
    assert!(files.files().packages.borrow().is_empty());
}

#[test]
fn source_transfer_failure_names_the_parameter_and_does_not_change_target() {
    let (mut files, source_id, _, mut source) = linked_files();
    files
        .commands(source_id)
        .unwrap()
        .run("Large value", |snapshot, _| {
            snapshot.parameters.set_expression(1, "1e300 cm")
        })
        .unwrap();
    let target = files
        .create_unbound(
            DocumentPackage::decode(
                b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  units: {workingScale: '1e-100'}\n",
            )
            .unwrap(),
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    let error = files
        .derived_source_values("/source.opd", target)
        .unwrap_err()
        .to_string();
    assert!(
        error.contains("/source.opd") && error.contains("width") && error.contains("representable")
    );
    assert!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .snapshot()
            .parameters
            .ids()
            .is_empty()
    );
}

#[path = "derived_refresh.rs"]
mod refresh;

#[path = "derived_propagation.rs"]
mod propagation;

#[path = "derived_automatic.rs"]
mod automatic;
