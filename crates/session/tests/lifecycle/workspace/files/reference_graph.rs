// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;
type ReferenceHost = FileWorkspace<ContentObserver, RecordingWorkspace, WorkspaceFiles>;

fn assembly(host: &mut ReferenceHost, source: &mut SequenceGuids, names: &[&str]) -> u64 {
    let package = DocumentPackage::empty(oblikovati_api::types::DOCUMENT_ASSEMBLY, "Assembly");
    let id = host
        .create_unbound(package, ContentObserver, "alpha", source)
        .unwrap()
        .session_id;
    host.commands(id)
        .unwrap()
        .run("Add references", |snapshot, _| -> Result<(), ()> {
            snapshot.assembly = Some(
                names
                    .iter()
                    .map(|name| AssemblyOccurrenceRecipe {
                        name: format!("{name}:1"),
                        component: (*name).into(),
                        ..Default::default()
                    })
                    .collect(),
            );
            Ok(())
        })
        .unwrap();
    id
}

#[test]
fn shared_models_are_retained_until_the_last_owner_closes() {
    let files = WorkspaceFiles::default();
    files
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(1));
    files
        .contents
        .borrow_mut()
        .insert(path("b.obk"), fixture(2));
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), files);
    let mut source = SequenceGuids(1000);
    let first = host
        .open(
            &path("a.obk"),
            ContentObserver,
            "alpha",
            &mut source,
            OpenActivation::Background,
        )
        .unwrap()
        .session_id();
    let second = host
        .open(
            &path("b.obk"),
            ContentObserver,
            "alpha",
            &mut source,
            OpenActivation::Background,
        )
        .unwrap()
        .session_id();
    host.set_visible(first, false).unwrap();
    host.set_visible(second, false).unwrap();
    let owner = assembly(&mut host, &mut source, &["a", "b"]);
    let other = assembly(&mut host, &mut source, &["a"]);
    assert!(
        host.attach_document_reference(owner, "".into(), first, false)
            .is_err()
    );
    host.attach_document_reference(owner, "a".into(), first, true)
        .unwrap();
    host.attach_document_reference(owner, "b".into(), second, true)
        .unwrap();
    host.attach_document_reference(other, "a".into(), first, false)
        .unwrap();
    assert_eq!(host.document_reference(owner, "b"), Some(second));
    assert!(host.close_discarding_changes(first).is_err());
    let order = host.document_close_order().unwrap();
    assert!(order.iter().position(|id| *id == owner) < order.iter().position(|id| *id == second));
    host.close_discarding_changes(owner).unwrap();
    assert!(host.workspace().documents().contains(&first));
    assert!(!host.workspace().documents().contains(&second));
    host.close_discarding_changes(other).unwrap();
    assert!(!host.workspace().documents().contains(&first));
}

#[test]
fn reference_cycles_are_refused_without_replacing_existing_edges() {
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), WorkspaceFiles::default());
    let mut source = SequenceGuids(1000);
    let first = assembly(&mut host, &mut source, &["second"]);
    let second = assembly(&mut host, &mut source, &["first"]);
    host.attach_document_reference(first, "second".into(), second, false)
        .unwrap();
    let error = host
        .attach_document_reference(second, "first".into(), first, false)
        .unwrap_err();
    assert!(error.to_string().contains("acyclic"));
    assert_eq!(host.document_reference(first, "second"), Some(second));
    assert_eq!(host.document_reference(second, "first"), None);
    assert_eq!(host.document_close_order().unwrap(), vec![first, second]);
    host.commands(first)
        .unwrap()
        .run("Remove reference", |snapshot, _| -> Result<(), ()> {
            snapshot.assembly = Some(vec![]);
            Ok(())
        })
        .unwrap();
    assert!(
        host.attach_document_reference(second, "first".into(), first, false)
            .is_err()
    );
    host.commands(first).unwrap().undo().unwrap();
    assert_eq!(host.document_reference(first, "second"), Some(second));
    assert_eq!(host.document_reference(second, "first"), None);
}

#[test]
fn nested_assembly_close_order_puts_every_owner_before_its_models() {
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), WorkspaceFiles::default());
    let mut source = SequenceGuids(1000);
    let leaf = assembly(&mut host, &mut source, &[]);
    let nested = assembly(&mut host, &mut source, &["leaf"]);
    let root = assembly(&mut host, &mut source, &["nested"]);
    host.attach_document_reference(nested, "leaf".into(), leaf, false)
        .unwrap();
    host.attach_document_reference(root, "nested".into(), nested, false)
        .unwrap();
    assert_eq!(
        host.document_close_order().unwrap(),
        vec![root, nested, leaf]
    );
}

#[test]
fn references_refuse_non_component_document_types() {
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), WorkspaceFiles::default());
    let mut source = SequenceGuids(1000);
    let drawing = host
        .create_unbound(
            DocumentPackage::empty(oblikovati_api::types::DOCUMENT_DRAWING, "Drawing"),
            ContentObserver,
            "alpha",
            &mut source,
        )
        .unwrap()
        .session_id;
    let owner = assembly(&mut host, &mut source, &["drawing"]);
    let error = host
        .attach_document_reference(owner, "drawing".into(), drawing, false)
        .unwrap_err();
    assert!(error.to_string().contains("expected a part or assembly"));
    assert_eq!(host.document_reference(owner, "drawing"), None);
}
