// SPDX-License-Identifier: GPL-2.0-only
//! Table edit acknowledgments follow the host result, including no-ops (#4124).
use super::*;
use oblikovati_api::wire::SetModelStateTableArgs;

fn request(owner: &mut TestOwner) -> (u64, SetModelStateTableArgs) {
    let document = owner.files.workspace().active_document().unwrap();
    let view = Client::new(owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    (
        document,
        SetModelStateTableArgs {
            expected_generation: view.generation,
            rows: view.rows,
            columns: view.columns,
            properties: view.properties,
            suppressions: view.suppressions,
            parameters: view.parameters,
        },
    )
}
#[test]
fn failed_noop_and_successful_noop_report_their_own_request_ids() {
    let (mut owner, _) = owner();
    owner.create_part_document().unwrap();
    let (document, initial) = request(&mut owner);
    assert!(
        owner
            .set_model_state_table_from_gui(document, 10, &initial)
            .unwrap()
    );
    let (_, valid) = request(&mut owner);
    let mut stale = valid.clone();
    stale.expected_generation += 1;
    assert!(
        owner
            .set_model_state_table_from_gui(document, 11, &stale)
            .unwrap()
    );
    let state = owner.gui_command_state().unwrap();
    let (ack_document, ack_id, result) = state.model_state_table_ack.unwrap();
    assert_eq!((ack_document, ack_id), (document, 11));
    assert!(result.is_err());
    assert!(
        owner
            .set_model_state_table_from_gui(document, 12, &valid)
            .unwrap()
    );
    let state = owner.gui_command_state().unwrap();
    assert_eq!(state.model_state_table_ack, Some((document, 12, Ok(()))));
    let after = state.model_state_table.unwrap();
    assert_eq!(after.rows, valid.rows);
    assert_eq!(after.columns, valid.columns);
    assert_eq!(after.properties, valid.properties);
    assert_eq!(after.suppressions, valid.suppressions);
    assert_eq!(after.parameters, valid.parameters);
}
