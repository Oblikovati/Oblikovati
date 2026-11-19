// SPDX-License-Identifier: GPL-2.0-only
//! Existing assembly parameter readers remain supported by staged part edits (#4122).
use super::*;

#[test]
fn staged_part_edit_refreshes_an_assembly_parameter_reader() {
    let mut host = FileWorkspace::new(QuietWorkspace, NativeDocumentFiles);
    let mut guids = SequenceGuids(200);
    let source = host
        .create_unbound(
            package(
                "source",
                "[{name: width, kind: user, expression: '1 cm', export: true}]",
            ),
            QuietDocument,
            "alpha",
            &mut guids,
        )
        .unwrap()
        .session_id;
    let assembly = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 2\n")
        .unwrap()
        .with_display_name("assembly");
    let target = host
        .create_unbound(assembly, QuietDocument, "alpha", &mut guids)
        .unwrap()
        .session_id;
    host.add_derived_table(target, "source", &["width".into()])
        .unwrap();
    host.run_named_prepared(
        source,
        None,
        |snapshot, _| {
            snapshot
                .parameters
                .set_expression(1, "2 cm")
                .map_err(|error| error.to_string())
        },
        |_, (), _| Ok(()),
    )
    .unwrap();
    assert_eq!(
        width(host.workspace.session(source).unwrap().snapshot()),
        2.0
    );
    assert_eq!(
        width(host.workspace.session(target).unwrap().snapshot()),
        2.0
    );
    host.commands(target).unwrap().begin("busy reader").unwrap();
    let source_generation = host.workspace.session(source).unwrap().generation();
    assert!(
        host.run_named_prepared(
            source,
            None,
            |snapshot, _| {
                snapshot
                    .parameters
                    .set_expression(1, "3 cm")
                    .map_err(|error| error.to_string())
            },
            |_, (), _| Ok(())
        )
        .is_err()
    );
    assert_eq!(
        host.workspace.session(source).unwrap().generation(),
        source_generation
    );
    assert_eq!(
        width(host.workspace.session(source).unwrap().snapshot()),
        2.0
    );
    assert_eq!(
        width(host.workspace.session(target).unwrap().snapshot()),
        2.0
    );
    host.commands(target).unwrap().abort().unwrap();
    host.navigate_named_prepared(
        source,
        None,
        oblikovati_model::history::PreparedNavigation::Undo,
    )
    .unwrap();
    assert_eq!(
        width(host.workspace.session(source).unwrap().snapshot()),
        1.0
    );
    assert_eq!(
        width(host.workspace.session(target).unwrap().snapshot()),
        1.0
    );
    host.navigate_named_prepared(
        source,
        None,
        oblikovati_model::history::PreparedNavigation::Redo,
    )
    .unwrap();
    assert_eq!(
        width(host.workspace.session(source).unwrap().snapshot()),
        2.0
    );
    assert_eq!(
        width(host.workspace.session(target).unwrap().snapshot()),
        2.0
    );
}
