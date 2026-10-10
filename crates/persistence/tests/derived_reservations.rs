// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_persistence::package::DocumentPackage;

fn recipe(schema: u32, mark: &str, tables: &str) -> DocumentPackage {
    DocumentPackage::decode(format!("schemaVersion: {schema}\ndocumentType: 1\nmodel:\n  derivedTableHighWater: {mark}\n  derivedParameterTables: {tables}\n").as_bytes()).unwrap()
}

#[test]
fn saved_marks_cover_live_and_deleted_tables_and_do_not_rewind() {
    let package = recipe(3, "9", "[{id: 7, sourceDocument: source.opd}]");
    assert_eq!(
        package
            .restore_document_parameters()
            .unwrap()
            .derived_high_water,
        9
    );
    assert!(package.with_derived_reservations(8).is_err());
    let updated = package.with_derived_reservations(10).unwrap();
    let reopened = DocumentPackage::decode(&updated.encode().unwrap()).unwrap();
    assert_eq!(
        reopened
            .restore_document_parameters()
            .unwrap()
            .derived_high_water,
        10
    );
    assert_eq!(
        package
            .restore_document_parameters()
            .unwrap()
            .derived_high_water,
        9
    );
}

#[test]
fn invalid_marks_and_schema_two_extension_use_are_refused() {
    for (schema, mark) in [
        (2, "9"),
        (3, "-1"),
        (3, "6"),
        (3, "null"),
        (3, "1.5"),
        (3, "9223372036854775808"),
    ] {
        let error = recipe(schema, mark, "[{id: 7, sourceDocument: source.opd}]")
            .restore_document_parameters()
            .unwrap_err();
        assert!(error.to_string().contains("expected"), "{error}");
    }
    assert_eq!(
        recipe(3, "9223372036854775807", "[]")
            .restore_document_parameters()
            .unwrap()
            .derived_high_water,
        i64::MAX
    );
}

#[test]
fn unused_table_namespaces_keep_the_legacy_schema() {
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    assert_eq!(package.with_derived_reservations(0).unwrap(), package);
    assert!(package.with_derived_reservations(-1).is_err());
    let reserved = package.with_derived_reservations(2).unwrap();
    assert_eq!(reserved.manifest().schema_version, 3);
    assert_eq!(
        reserved
            .restore_document_parameters()
            .unwrap()
            .derived_high_water,
        2
    );
}
