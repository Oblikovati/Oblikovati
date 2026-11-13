// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;
use oblikovati_persistence::package::DocumentPackage;

#[test]
fn reads_an_assembly_package_written_by_the_go_host() {
    let package =
        DocumentPackage::decode(include_bytes!("fixtures/go-assembly-occurrences.yaml")).unwrap();
    let records = package.assembly_occurrences().unwrap().unwrap();
    assert_eq!(records.len(), 2);
    assert_eq!(records[0].name, "widget:1");
    assert_eq!(records[1].name, "widget:2");
    assert_eq!(records[0].component, "parts/widget.opd");
    assert_eq!(records[0].component, records[1].component);
    assert!(records[0].grounded && !records[0].suppressed);
    assert!(records[1].suppressed && !records[1].grounded);
    assert_eq!(records[1].placement().unwrap().cells[3], 3.);
    let reopened = DocumentPackage::decode(&package.encode().unwrap()).unwrap();
    assert_eq!(reopened.assembly_occurrences().unwrap(), Some(records));
}

#[test]
fn reads_go_occurrences_without_resolving_their_shared_component() {
    let yaml = b"schemaVersion: 2\ndocumentType: 2\nmodel:\n  occurrences:\n    - name: 'Part:1'\n      component: parts/model.opd\n      transform: [1, 0, 0, 10, 0, 1, 0, 20, 0, 0, 1, 30, 0, 0, 0, 1]\n      grounded: true\n    - name: 'Part:2'\n      component: parts/model.opd\n      suppressed: true\n      flexible: true\n      childTransforms: { 'Child:1': [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1] }\n    - name: 'Virtual:1'\n      virtual: true\n      virtualPartNumber: P123\n";
    let package = DocumentPackage::decode(yaml).unwrap();
    let occurrences = package.assembly_occurrences().unwrap().unwrap();
    assert_eq!(occurrences.len(), 3);
    assert_eq!(occurrences[0].component, occurrences[1].component);
    assert_eq!(occurrences[0].transform[3], 10.);
    assert_eq!(occurrences[0].placement().unwrap().cells[3], 10.);
    assert_eq!(
        occurrences[1].placement().unwrap().cells,
        [
            1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1.
        ]
    );
    assert!(occurrences[0].grounded && occurrences[1].suppressed && occurrences[1].flexible);
    assert_eq!(occurrences[1].child_transforms["Child:1"].len(), 16);
    assert!(occurrences[2].virtual_);
    assert_eq!(occurrences[2].virtual_part_number, "P123");
    let encoded = serde_yaml_ng::to_string(&occurrences).unwrap();
    let restored: Vec<AssemblyOccurrenceRecipe> = serde_yaml_ng::from_str(&encoded).unwrap();
    assert_eq!(restored, occurrences);
}

#[test]
fn declines_bad_transform_and_child_matrices_with_occurrence_name() {
    for occurrence in [
        "{name: bad, component: part.opd, transform: [1, 2]}",
        "{name: bad, component: part.opd, transform: [.nan]}",
        "{name: bad, component: part.opd, childTransforms: {child: [1]}}",
    ] {
        let yaml =
            format!("schemaVersion: 2\ndocumentType: 2\nmodel: {{occurrences: [{occurrence}]}}\n");
        let error = DocumentPackage::decode(yaml.as_bytes())
            .unwrap()
            .assembly_occurrences()
            .unwrap_err();
        let detail = error.to_string();
        assert!(detail.contains("bad") && detail.contains("16"), "{detail}");
    }
}

#[test]
fn empty_and_non_assembly_packages_remain_distinct() {
    let assembly = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 2\n").unwrap();
    let part = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    assert_eq!(assembly.assembly_occurrences().unwrap(), Some(vec![]));
    assert_eq!(part.assembly_occurrences().unwrap(), None);
}
