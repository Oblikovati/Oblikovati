// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn evaluation_refresh_preserves_redo_and_advances_generation_only_when_changed() {
    let mut history = history();
    let token = history.begin("authored").unwrap();
    set_width(&mut history, &token, "2 cm");
    history.commit(&token, false).unwrap();
    history.mark_saved().unwrap();
    history.undo().unwrap();
    let entries = history.entries();
    let generation = history.generation();
    assert!(
        history
            .refresh(|snapshot| snapshot.parameters.set_expression(1, "3 cm"))
            .unwrap()
    );
    assert_eq!(history.entries(), entries);
    assert_eq!(history.generation(), generation + 1);
    assert_eq!(history.high_water(), 1);
    assert!(
        !history
            .refresh(|snapshot| snapshot.parameters.set_expression(1, "3 cm"))
            .unwrap()
    );
    assert_eq!(history.generation(), generation + 1);
    history.redo().unwrap();
    assert_eq!(width(&history), 2.0);
}

#[test]
fn failed_refresh_and_unreserved_identity_are_atomic_and_open_groups_are_refused() {
    let mut history = history();
    let generation = history.generation();
    assert!(
        history
            .refresh(|snapshot| {
                snapshot.parameters.set_expression(1, "3 cm").unwrap();
                Err::<(), _>("failed evaluation")
            })
            .is_err()
    );
    assert!(matches!(
        history.refresh(|snapshot| snapshot.parameters.insert_expression(
            2,
            "unreserved",
            USER_PARAM,
            "1"
        )),
        Err(EditError::History(HistoryError::UnreservedIdentity(2)))
    ));
    assert_eq!(width(&history), 1.0);
    assert_eq!(history.generation(), generation);
    assert_eq!(history.high_water(), 1);
    let token = history.begin("open").unwrap();
    assert!(history.refresh(|_| Ok::<(), String>(())).is_err());
    history.abort(&token).unwrap();
}
