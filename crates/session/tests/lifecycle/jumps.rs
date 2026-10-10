// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[derive(Default)]
struct VetoSecondUndo {
    attempts: usize,
    changes: usize,
}
impl TransactionObserver for VetoSecondUndo {
    fn before(
        &mut self,
        event: &TransactionEventPayload,
        _: &DocumentSnapshot,
    ) -> Result<(), String> {
        if event.type_ == EVENT_TRANSACTION_UNDONE {
            self.attempts += 1;
            if self.attempts == 2 {
                return Err("second undo is locked".into());
            }
        }
        Ok(())
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {}
    fn changed(&mut self, _: u64, _: u64) {
        self.changes += 1;
    }
}

#[test]
fn cursor_jump_stops_at_last_accepted_step_on_veto() {
    let history =
        DocumentEditHistory::new(recipe(), RecipeIdAllocator::restore(Some(1), &[1]).unwrap())
            .unwrap();
    let mut session = DocumentSession::new(17, history, VetoSecondUndo::default());
    for (label, source) in [("first", "2 cm"), ("second", "3 cm"), ("third", "4 cm")] {
        session
            .run(label, |snapshot, _| {
                snapshot.parameters.set_expression(1, source)
            })
            .unwrap();
    }
    assert_eq!(
        session.jump_to(0),
        Err(SessionError::Veto("second undo is locked".into()))
    );
    assert_eq!(session.undo_labels(), ["first", "second"]);
    assert_eq!(session.redo_labels(), ["third"]);
    assert_eq!(width(session.snapshot()), 3.0);
    assert_eq!(session.observer().attempts, 2);
    assert_eq!(session.observer().changes, 4);
    session.jump_to(0).unwrap();
    assert_eq!(width(session.snapshot()), 1.0);
    session.jump_to(3).unwrap();
    assert_eq!(width(session.snapshot()), 4.0);
}

#[test]
fn invalid_or_open_group_jumps_leave_cursor_generation_and_events_unchanged() {
    let mut session = session(None);
    commit_width(&mut session);
    let generation = session.generation();
    let count = session.observer().observations.len();
    for position in [-1, 2, i64::MAX] {
        let error = session.jump_to(position).unwrap_err();
        assert_eq!(
            error,
            SessionError::HistoryPosition {
                supplied: position,
                total: 1
            }
        );
        assert!(error.to_string().contains(&position.to_string()));
    }
    session.jump_to(1).unwrap();
    assert_eq!(session.generation(), generation);
    assert_eq!(session.observer().observations.len(), count);
    session.begin("preview").unwrap();
    let error = session.jump_to(1).unwrap_err();
    assert_eq!(error, SessionError::NavigationGroupOpen);
    assert!(
        error
            .to_string()
            .contains("expected all transactions closed")
    );
    session.abort().unwrap();
}
