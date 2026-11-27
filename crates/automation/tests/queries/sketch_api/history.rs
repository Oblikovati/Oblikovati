// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn grouped_api_edit(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    let client = Client::new(FileHost::new(files, source, "test"));
    client
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Resize circle".into(),
        })
        .unwrap();
    client.sketch().solve(&SketchArgs::default()).unwrap();
    client
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "4 cm".into(),
            ..Default::default()
        })
        .unwrap();
    client.transactions().end().unwrap();
}
fn reopened_document(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
) -> u64 {
    reopen_with_observer(files, document, source, QueryObserver::default())
}
#[test]
fn typed_solve_joins_transactions_and_preserves_identity_through_save_reopen() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    files.save(document, "test", &mut source).unwrap();
    grouped_api_edit(&mut files, &mut source);
    assert_radius(&files, document, sketch, 4.0);
    assert_eq!(
        owner_revision(&files, document).1.last().unwrap(),
        "Resize circle"
    );
    files.commands(document).unwrap().undo().unwrap();
    assert_radius(&files, document, sketch, 1.0);
    assert!(!files.workspace().needs_save(document).unwrap());
    files.commands(document).unwrap().redo().unwrap();
    let reopened = reopened_document(&mut files, document, &mut source);
    assert_radius(&files, reopened, sketch, 4.0);
    assert_eq!(status_call(&files, 0).unwrap().status, "well");
    assert!(!files.workspace().needs_save(reopened).unwrap());
}

fn assert_radius(
    files: &DiscoveryWorkspace,
    document: u64,
    sketch: DocumentSketchId,
    expected: f64,
) {
    let actual = circle_radius(files.workspace(), document, sketch);
    assert!(
        (actual - expected).abs() < 1e-8,
        "radius {actual}: expected {expected}"
    );
}
