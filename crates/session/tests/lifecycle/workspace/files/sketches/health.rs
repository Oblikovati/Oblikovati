// SPDX-License-Identifier: GPL-2.0-only
use super::recompute::saved_driven_sketch;
use super::*;
use oblikovati_model::document_sketch::SketchReadinessCause;
fn decline(host: &Host, id: u64, sketch: DocumentSketchId) -> Option<SketchReadinessCause> {
    let snapshot = host.workspace().session(id).unwrap().snapshot();
    let placed = snapshot.sketches.get(sketch).unwrap();
    placed
        .require_fully_constrained(&snapshot.parameters)
        .err()
        .map(|error| error.cause)
}
fn assert_fresh(host: &Host, id: u64, sketch: DocumentSketchId) {
    assert_eq!(decline(host, id, sketch), None);
}
fn assert_stale(host: &Host, id: u64, sketch: DocumentSketchId) {
    assert_eq!(
        decline(host, id, sketch),
        Some(SketchReadinessCause::Stale(vec![1]))
    );
}
fn recompute(host: &mut Host, id: u64) {
    let report = host
        .commands(id)
        .unwrap()
        .recompute_sketches(SolveOptions::default(), || false)
        .unwrap();
    assert!(!report.cancelled);
}
fn set_width(host: &mut Host, id: u64, expression: &str) {
    host.commands(id)
        .unwrap()
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, expression)
        })
        .unwrap();
}
#[test]
fn undo_redo_and_recompute_preserve_health_against_each_snapshots_parameter_targets() {
    let (mut host, _, id, sketch) = saved_driven_sketch();
    recompute(&mut host, id);
    assert_fresh(&host, id, sketch);
    set_width(&mut host, id, "6 cm");
    assert_stale(&host, id, sketch);
    host.commands(id).unwrap().undo().unwrap();
    assert_fresh(&host, id, sketch);
    host.commands(id).unwrap().redo().unwrap();
    assert_stale(&host, id, sketch);
    assert_recompute_navigation(&mut host, id, sketch);
}
fn assert_recompute_navigation(host: &mut Host, id: u64, sketch: DocumentSketchId) {
    recompute(host, id);
    assert_fresh(host, id, sketch);
    host.commands(id).unwrap().undo().unwrap();
    assert_stale(host, id, sketch);
    host.commands(id).unwrap().redo().unwrap();
    assert_fresh(host, id, sketch);
}
#[test]
fn reopening_requires_verification_but_health_only_recompute_keeps_the_saved_recipe_clean() {
    let (mut host, mut source, id, sketch) = saved_driven_sketch();
    recompute(&mut host, id);
    assert!(!host.workspace().needs_save(id).unwrap());
    host.close_discarding_changes(id).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_eq!(
        decline(&host, reopened, sketch),
        Some(SketchReadinessCause::Unevaluated)
    );
    recompute(&mut host, reopened);
    assert_fresh(&host, reopened, sketch);
    assert!(!host.workspace().needs_save(reopened).unwrap());
    let history = host.workspace().session(reopened).unwrap().undo_labels();
    assert!(history.is_empty());
}
#[test]
fn cancelled_recompute_keeps_stale_health_and_prior_coordinates() {
    let (mut host, _, id, sketch) = saved_driven_sketch();
    recompute(&mut host, id);
    set_width(&mut host, id, "6 cm");
    let report = host
        .commands(id)
        .unwrap()
        .recompute_sketches(SolveOptions::default(), || true)
        .unwrap();
    assert!(report.cancelled);
    assert_stale(&host, id, sketch);
    assert!((point_height(&host, id, sketch) - 1.0).abs() < 1e-8);
}
#[path = "health/failure.rs"]
mod failure;
