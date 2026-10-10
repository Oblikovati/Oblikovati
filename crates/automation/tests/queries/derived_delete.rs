// SPDX-License-Identifier: GPL-2.0-only

use super::parameters::{parameter_workspace, parameter_workspace_observer};
use super::*;
use oblikovati_automation::SessionHost;
use oblikovati_model::quantity::Quantity;
use std::collections::BTreeMap;

fn linked_parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .add_derived_table(
            7,
            "source.opd",
            &["width".into()],
            &BTreeMap::from([("width".into(), Quantity::new(2.0, Dimension::Length))]),
            &BTreeMap::from([("width".into(), 1)]),
        )
        .unwrap();
    parameters
        .insert_expression(2, "double_width", USER_PARAM, "width * 2")
        .unwrap();
    parameters
}

#[test]
fn derived_deletion_and_reader_health_are_undoable_through_the_client() {
    let (mut workspace, id) = parameter_workspace(linked_parameters());
    let original = workspace.session(id).unwrap().snapshot().clone();
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        client
            .parameters()
            .delete_derived_table(&DerivedParameterTableDeleteArgs { id: 7 })
            .unwrap();
        assert!(
            client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "width".into()
                })
                .is_err()
        );
        assert!(
            !client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "double_width".into()
                })
                .unwrap()
                .health
                .is_empty()
        );
        let state = client.transactions().state().unwrap();
        assert_eq!(state.next_undo, "Edit Derived Parameters");
        client.transactions().undo().unwrap();
        assert_eq!(
            client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "width".into()
                })
                .unwrap()
                .kind,
            "derived"
        );
    }
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .same_recipe(&original.parameters)
    );
    assert_eq!(
        workspace.session(id).unwrap().snapshot().properties,
        original.properties
    );
    workspace.commands(id).unwrap().redo().unwrap();
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .derived_tables()
            .is_empty()
    );
    assert!(workspace.needs_save(id).unwrap());
}

#[test]
fn missing_owned_and_vetoed_table_deletion_preserves_the_entire_recipe() {
    for (owned, veto, target) in [(false, false, 99), (true, false, 7), (false, true, 7)] {
        let mut parameters = linked_parameters();
        if owned {
            parameters.mark_derived_table_owned(7).unwrap();
        }
        let (mut workspace, id) = parameter_workspace_observer(
            parameters,
            QueryObserver {
                veto,
                ..Default::default()
            },
        );
        let original = workspace.session(id).unwrap().snapshot().clone();
        assert!(
            Client::new(SessionHost::new(&mut workspace))
                .parameters()
                .delete_derived_table(&DerivedParameterTableDeleteArgs { id: target })
                .is_err()
        );
        assert!(
            workspace
                .session(id)
                .unwrap()
                .snapshot()
                .parameters
                .same_recipe(&original.parameters)
        );
        assert_eq!(
            workspace.session(id).unwrap().snapshot().properties,
            original.properties
        );
        assert!(workspace.session(id).unwrap().undo_labels().is_empty());
        assert!(!workspace.needs_save(id).unwrap());
    }
}
