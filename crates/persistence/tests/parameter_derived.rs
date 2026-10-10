// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::{
    parameter::DerivedTableHealth,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};
use oblikovati_persistence::package::{DocumentPackage, PackageCodecError, RestoredParameters};
use std::collections::BTreeMap;

fn restore(tables: &str) -> Result<RestoredParameters, PackageCodecError> {
    let yaml = format!(
        "schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters: [{{name: module, kind: derived, value: 0.2, unit: length}}, {{name: pitch, kind: user, expression: 'module * 24'}}]\n  derivedParameterTables: {tables}\n"
    );
    DocumentPackage::decode(yaml.as_bytes())?.restore_parameters(UnitsOfMeasure::default())
}

#[test]
fn loaded_table_refreshes_reconnected_parameters_and_dependents() {
    let mut restored = restore("[{id: 7, sourceDocument: gears.obk, linked: [module], ownedByFeature: true}, {id: 2, sourceDocument: pending.obk, linked: [missing]}]").unwrap();
    assert_eq!(
        restored
            .parameters
            .derived_tables()
            .iter()
            .map(|table| table.id)
            .collect::<Vec<_>>(),
        [7, 2]
    );
    assert!(
        restored
            .parameters
            .derived_table(7)
            .unwrap()
            .owned_by_feature
    );
    assert_eq!(
        restored.parameters.derived_references(7).unwrap()[0].parameter,
        "module"
    );
    assert!(
        restored
            .parameters
            .derived_references(2)
            .unwrap()
            .is_empty()
    );
    assert!(restored.parameters.drain_changed().is_empty());
    let source = BTreeMap::from([("module".into(), Quantity::new(0.25, Dimension::Length))]);
    restored
        .parameters
        .sync_derived_table(7, Some(&source))
        .unwrap();
    assert_eq!(
        restored
            .parameters
            .get(2)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        6.0
    );
    restored.parameters.sync_derived_table(7, None).unwrap();
    assert_eq!(
        restored.parameters.derived_table(7).unwrap().health,
        DerivedTableHealth::Unavailable
    );
}

#[test]
fn malformed_or_ambiguous_tables_refuse_the_candidate() {
    for tables in [
        "{}",
        "[{}]",
        "[{id: -1, sourceDocument: x}]",
        "[{id: 1}]",
        "[{id: 1, sourceDocument: x, linked: [pitch]}]",
        "[{id: 1, sourceDocument: x, future: true}]",
        "[{id: 1, sourceDocument: x}, {id: 1, sourceDocument: y}]",
        "[{id: 1, sourceDocument: x, linked: [module]}, {id: 2, sourceDocument: y, linked: [module]}]",
    ] {
        assert!(restore(tables).is_err(), "{tables}");
    }
}

#[test]
fn null_or_empty_tables_are_empty() {
    for tables in ["null", "[]"] {
        assert!(
            restore(tables)
                .unwrap()
                .parameters
                .derived_tables()
                .is_empty()
        );
    }
}

#[test]
fn pending_loaded_links_use_the_document_allocator_when_selected() {
    let mut restored =
        restore("[{id: 7, sourceDocument: gears.obk, linked: [module, teeth]}]").unwrap();
    let allocated = restored.ids.allocate().unwrap();
    assert_eq!(allocated, 3);
    let source = BTreeMap::from([
        ("module".into(), Quantity::new(0.25, Dimension::Length)),
        ("teeth".into(), Quantity::new(30.0, Dimension::Unitless)),
    ]);
    let ids = BTreeMap::from([("teeth".into(), allocated)]);
    restored
        .parameters
        .set_derived_table_linked(7, &["module".into(), "teeth".into()], &source, &ids)
        .unwrap();
    assert_eq!(restored.parameters.find("module"), Some(1));
    assert_eq!(restored.parameters.find("teeth"), Some(allocated));
    assert_eq!(
        restored
            .parameters
            .get(2)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        6.0
    );
}
