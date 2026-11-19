// SPDX-License-Identifier: GPL-2.0-only
//! Shell contract inputs are checked before creating a private candidate (#4106).
use super::*;
use oblikovati_model::{
    feature::work_geometry::DatumLength,
    identity::{DocumentGuid, SemanticReferenceName},
};

fn key(kind: u32, identity: u64) -> Vec<u8> {
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    SemanticReferenceName::new(document, 1, kind, "fixture", "source", &[identity])
        .unwrap()
        .encode()
        .unwrap()
}

#[test]
fn shell_inputs_keep_driving_parameter_and_direction() {
    let snapshot = crate::prism_test_fixture::selected_prism_document();
    let input = Shell {
        thickness: "width".into(),
        direction: " OUTSIDE ".into(),
        ..Default::default()
    };
    let recipe = prepare_shell(&snapshot, key(4, 1), vec![key(1, 1)], &input)
        .unwrap()
        .definition
        .recipe();
    assert_eq!(recipe.thickness, DatumLength::Parameter(1));
    assert_eq!(recipe.direction, ShellDirection::Outside);
}

#[test]
fn shell_rejects_empty_openings_and_invalid_wall_inputs() {
    let snapshot = crate::prism_test_fixture::selected_prism_document();
    assert!(opening_keys(&Shell::default()).is_err());
    for (thickness, direction) in [("", "inside"), ("0 mm", "inside"), ("1 mm", "sideways")] {
        let input = Shell {
            thickness: thickness.into(),
            direction: direction.into(),
            ..Default::default()
        };
        assert!(prepare_shell(&snapshot, key(4, 1), vec![key(1, 1)], &input).is_err());
    }
}

struct SilentShellObserver;
impl oblikovati_session::TransactionObserver for SilentShellObserver {
    fn before(
        &mut self,
        _: &oblikovati_api::wire::TransactionEventPayload,
        _: &DocumentSnapshot,
    ) -> Result<(), String> {
        Ok(())
    }
    fn after(&mut self, _: &oblikovati_api::wire::TransactionEventPayload, _: &DocumentSnapshot) {}
    fn changed(&mut self, _: u64, _: u64) {}
}

#[test]
fn shell_candidate_preview_does_not_change_history() {
    use oblikovati_model::{history::DocumentEditHistory, identity::RecipeIdAllocator};
    let snapshot = crate::prism_test_fixture::selected_prism_document();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    let history = DocumentEditHistory::new(snapshot, ids).unwrap();
    let owner = oblikovati_session::DocumentSession::new(1, history, SilentShellObserver);
    let input = Shell {
        thickness: "1 mm".into(),
        ..Default::default()
    };
    let prepared = prepare_shell(owner.snapshot(), key(4, 1), vec![key(1, 1)], &input).unwrap();
    let before = owner.generation();
    let kind = owner
        .preview_projected(
            |candidate, ids| prepared.create(candidate, ids),
            |candidate, id| {
                Ok::<_, Error>(
                    candidate
                        .work_geometry
                        .get(id)
                        .unwrap()
                        .definition()
                        .kind()
                        .to_owned(),
                )
            },
        )
        .unwrap();
    assert_eq!(kind, "shell");
    assert_eq!(owner.generation(), before);
    assert_eq!(owner.snapshot().work_geometry.features().len(), 1);
}
