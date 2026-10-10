// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{history::DocumentEditHistory, parameter::ParameterHealth};
#[derive(Default)]
struct HealthObserver {
    observed: Vec<ParameterHealth<u64>>,
    generations: Vec<(u64, u64)>,
}
impl HealthObserver {
    fn observe(&mut self, snapshot: &DocumentSnapshot) {
        self.observed
            .push(snapshot.parameters.get(1).unwrap().health.clone());
    }
}
impl TransactionObserver for HealthObserver {
    fn before(
        &mut self,
        _: &TransactionEventPayload,
        snapshot: &DocumentSnapshot,
    ) -> Result<(), String> {
        self.observe(snapshot);
        Ok(())
    }
    fn after(&mut self, _: &TransactionEventPayload, snapshot: &DocumentSnapshot) {
        self.observe(snapshot);
    }
    fn changed(&mut self, document: u64, generation: u64) {
        self.generations.push((document, generation));
    }
}
fn health_session() -> (DocumentSession<HealthObserver>, DocumentSketchId) {
    let ids = RecipeIdAllocator::restore(Some(1), &[1]).unwrap();
    let history = DocumentEditHistory::new(recipe(), ids).unwrap();
    let mut session = DocumentSession::new(17, history, HealthObserver::default());
    let sketch = session.run("Sketch", add_sketch).unwrap();
    session
        .recompute_sketches(SolveOptions::default(), || false)
        .unwrap();
    (session, sketch)
}
#[test]
fn failed_parameter_health_reaches_session_observations_and_sketch_consumer_declines() {
    let (mut session, sketch) = health_session();
    session
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "missing + 1 cm")
        })
        .unwrap();
    let report = session
        .recompute_sketches(SolveOptions::default(), || false)
        .unwrap();
    assert!(report.reports[0].result.is_err());
    assert_failed_sketch(&session, sketch);
    assert!(
        session
            .observer()
            .observed
            .iter()
            .any(|health| matches!(health, ParameterHealth::Failed(_)))
    );
    assert!(!session.observer().generations.is_empty());
}

fn assert_failed_sketch(session: &DocumentSession<HealthObserver>, sketch: DocumentSketchId) {
    let snapshot = session.snapshot();
    let placed = snapshot.sketches.get(sketch).unwrap();
    let error = placed
        .require_fully_constrained(&snapshot.parameters)
        .unwrap_err();
    assert!(matches!(error.cause, SketchReadinessCause::Failed(_)));
    assert_eq!(error.sketch, sketch);
}
