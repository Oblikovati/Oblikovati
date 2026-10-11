// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{
    client::{Caller, Client},
    types::*,
    wire::*,
};
use oblikovati_automation::{QUERY_METHODS, SessionQueryHost};
use oblikovati_model::{
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::*,
    parameter::ParameterCollection,
    property::DocumentProperties,
    quantity::Dimension,
    units::UnitsOfMeasure,
};
use oblikovati_session::{TransactionObserver, workspace::*};
use std::path::Path;

#[derive(Default)]
struct QueryObserver {
    interest_notices: Vec<(u64, String, Vec<String>)>,
    notifications: usize,
    veto: bool,
}
impl TransactionObserver for QueryObserver {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        self.notifications += 1;
        if self.veto {
            return Err("settings are locked".into());
        }
        Ok(())
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {
        self.notifications += 1;
    }
    fn changed(&mut self, _: u64, _: u64) {
        self.notifications += 1;
    }
}
impl WorkspaceObserver for QueryObserver {
    fn interest_client_present(&self, _: &str) -> bool {
        false
    }
    fn unavailable_interest_clients(&mut self, id: u64, name: &str, clients: &[String]) {
        self.notifications += 1;
        self.interest_notices
            .push((id, name.into(), clients.to_vec()));
    }

    fn created(&mut self, _: u64, _: IdentityRegistration) {
        self.notifications += 1;
    }
    fn opened(&mut self, _: u64, _: IdentityRegistration) {
        self.notifications += 1;
    }
    fn activated(&mut self, _: u64) {
        self.notifications += 1;
    }
    fn closed(&mut self, _: u64, _: Option<u64>) {
        self.notifications += 1;
    }
    fn saved(&mut self, _: u64) {
        self.notifications += 1;
    }
    fn path_changed(&mut self, _: u64, _: &Path, _: &Path) {
        self.notifications += 1;
    }
    fn before_open(&mut self, _: &Path) -> Result<(), String> {
        self.notifications += 1;
        Ok(())
    }
    fn before_save(&mut self, _: u64, _: &DocumentSnapshot) -> Result<(), String> {
        self.notifications += 1;
        Ok(())
    }
    fn before_close(&mut self, _: u64, _: &DocumentSnapshot) -> Result<(), String> {
        self.notifications += 1;
        Ok(())
    }
}
struct QueryGuids(u64);
impl GuidSource for QueryGuids {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        self.0 += 1;
        Ok(format!("00000000-0000-4000-8000-{:012x}", self.0)
            .parse()
            .unwrap())
    }
}
type Workspace = WorkspaceSession<QueryObserver, QueryObserver>;
fn register(workspace: &mut Workspace, units: UnitsOfMeasure, source: &mut QueryGuids) -> u64 {
    let snapshot = DocumentSnapshot {
        work_geometry: Default::default(),
        sketches: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
        parameters: ParameterCollection::new(units),
        properties: DocumentProperties::default(),
    };
    let history =
        DocumentEditHistory::new(snapshot, RecipeIdAllocator::restore(Some(0), &[]).unwrap())
            .unwrap();
    workspace
        .register(
            history,
            FileIdentity::new("test", source).unwrap(),
            QueryObserver::default(),
            source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id
}

#[test]
fn canonical_client_reads_unit_settings_and_formats_without_mutating_the_session() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let id = register(
        &mut workspace,
        UnitsOfMeasure::default(),
        &mut QueryGuids(0),
    );
    let notices = workspace.observer().notifications;
    let client = Client::new(SessionQueryHost::new(&workspace));
    let info = client.documents().get_units().unwrap();
    assert_eq!(
        (
            info.length_unit.as_str(),
            info.angle_unit.as_str(),
            info.mass_unit.as_str(),
            info.time_unit.as_str()
        ),
        ("mm", "deg", "kg", "s")
    );
    assert_eq!(
        (info.length_display_precision, info.angle_display_precision),
        (3, 2)
    );
    assert_eq!(info.length_display_format, "decimal");
    assert_eq!(info.working_scale_cm, 1.0);
    assert_eq!(
        client
            .units()
            .get_string_from_type(&UnitsTypeArgs {
                units_type: "length".into()
            })
            .unwrap()
            .value,
        "mm"
    );
    let request = StringFromValueArgs {
        value: 1.0,
        units_type: "length".into(),
    };
    assert_eq!(
        client
            .units()
            .get_string_from_value(&request)
            .unwrap()
            .value,
        "10.000 mm"
    );
    assert_eq!(
        client
            .units()
            .get_precise_string_from_value(&request)
            .unwrap()
            .value,
        "10 mm"
    );
    assert_eq!(workspace.observer().notifications, notices);
    let session = workspace.session(id).unwrap();
    assert_eq!(session.generation(), 0);
    assert_eq!(session.observer().notifications, 0);
    assert!(session.undo_labels().is_empty());
    assert!(!workspace.needs_save(id).unwrap());
}

#[test]
fn physical_conversion_and_category_queries_do_not_require_a_document() {
    let workspace = Workspace::new(QueryObserver::default());
    let client = Client::new(SessionQueryHost::new(&workspace));
    assert_eq!(
        client
            .units()
            .convert(&ConvertUnitsArgs {
                value: 1.0,
                from: "in".into(),
                to: "mm".into()
            })
            .unwrap()
            .value,
        25.4
    );
    assert_eq!(
        client
            .units()
            .get_type_from_string(&UnitStringArgs {
                unit_string: "mm".into()
            })
            .unwrap()
            .units_type,
        "length"
    );
    assert!(
        client
            .documents()
            .get_units()
            .unwrap_err()
            .to_string()
            .contains("active document")
    );
    assert!(
        client
            .units()
            .convert(&ConvertUnitsArgs {
                value: 1.0,
                from: "in".into(),
                to: "s".into()
            })
            .is_err()
    );
    assert!(
        client
            .units()
            .get_type_from_string(&UnitStringArgs {
                unit_string: "not-a-unit".into()
            })
            .is_err()
    );
    assert_eq!(workspace.observer().notifications, 0);
}

#[test]
fn query_context_follows_explicit_activation_and_keeps_working_scale() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let mut source = QueryGuids(0);
    let first = register(&mut workspace, UnitsOfMeasure::default(), &mut source);
    let mut inches = UnitsOfMeasure::default().with_working_scale(2.54).unwrap();
    inches.set_preferred(Dimension::Length, "in").unwrap();
    let second = register(&mut workspace, inches, &mut source);
    let client = Client::new(SessionQueryHost::new(&workspace));
    assert_eq!(
        client.documents().get_units().unwrap().working_scale_cm,
        2.54
    );
    assert_eq!(
        client
            .units()
            .get_precise_string_from_value(&StringFromValueArgs {
                value: 1.0,
                units_type: "length".into()
            })
            .unwrap()
            .value,
        "1 in"
    );
    workspace.activate(first).unwrap();
    assert_eq!(
        Client::new(SessionQueryHost::new(&workspace))
            .documents()
            .get_units()
            .unwrap()
            .length_unit,
        "mm"
    );
    assert_eq!(workspace.session(second).unwrap().generation(), 0);
}

#[test]
fn display_format_names_are_taken_from_validated_document_state() {
    for (format, expected) in [
        (DISPLAY_FORMAT_FRACTIONAL, "fractional"),
        (DISPLAY_FORMAT_ARCHITECTURAL, "architectural"),
    ] {
        let mut units = UnitsOfMeasure::default();
        let mut display = units.display_options();
        display.length_format = format;
        units.set_display_options(display).unwrap();
        let mut workspace = Workspace::new(QueryObserver::default());
        register(&mut workspace, units, &mut QueryGuids(0));
        let client = Client::new(SessionQueryHost::new(&workspace));
        assert_eq!(
            client
                .documents()
                .get_units()
                .unwrap()
                .length_display_format,
            expected
        );
    }
}

#[test]
fn unregistered_methods_and_malformed_arguments_are_refused_before_model_work() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let id = register(
        &mut workspace,
        UnitsOfMeasure::default(),
        &mut QueryGuids(0),
    );
    let host = SessionQueryHost::new(&workspace);
    assert_eq!(
        QUERY_METHODS
            .iter()
            .collect::<std::collections::BTreeSet<_>>()
            .len(),
        31
    );
    assert!(
        host.call(METHOD_PARAMETERS_SET, b"{}")
            .unwrap_err()
            .to_string()
            .contains("read-only")
    );
    for request in [
        b"".as_slice(),
        b"[]",
        b"{",
        b"null",
        br#"{"value":"bad","from":"cm","to":"mm"}"#,
    ] {
        assert!(host.call(METHOD_UNITS_CONVERT, request).is_err());
    }
    let client = Client::new(host);
    assert!(
        client
            .units()
            .get_string_from_type(&UnitsTypeArgs {
                units_type: "bad".into()
            })
            .is_err()
    );
    assert!(
        client
            .units()
            .get_string_from_value(&StringFromValueArgs {
                value: 1.0,
                units_type: "text".into()
            })
            .is_err()
    );
    assert!(
        client
            .units()
            .get_precise_string_from_value(&StringFromValueArgs {
                value: 1.0,
                units_type: "bad".into()
            })
            .is_err()
    );
    assert_eq!(workspace.session(id).unwrap().generation(), 0);
}

#[path = "queries/expressions.rs"]
mod expressions;

#[path = "queries/parameters.rs"]
mod parameters;

#[path = "queries/collection.rs"]
mod collection;

#[path = "queries/detail.rs"]
mod detail;

#[path = "queries/settings_edit.rs"]
mod settings_edit;

#[path = "queries/groups_edit.rs"]
mod groups_edit;

#[path = "queries/parameter_edit.rs"]
mod parameter_edit;

#[path = "queries/parameter_value.rs"]
mod parameter_value;

#[path = "queries/parameter_choices.rs"]
mod parameter_choices;

#[path = "queries/parameter_tolerance.rs"]
mod parameter_tolerance;

#[path = "queries/parameter_update.rs"]
mod parameter_update;

#[path = "queries/transactions.rs"]
mod transactions;

#[path = "queries/transaction_jumps.rs"]
mod transaction_jumps;

#[path = "queries/properties.rs"]
mod properties;

#[path = "queries/file_discovery.rs"]
mod file_discovery;

#[path = "queries/derived_delete.rs"]
mod derived_delete;

#[path = "queries/sketch_updates.rs"]
mod sketch_updates;
