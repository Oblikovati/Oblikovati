// SPDX-License-Identifier: GPL-2.0-only

//! Typed owner-thread queries and transactional edits; native transport is a separate boundary.
mod datums;
mod part_owner;
pub use datums::{DATUM_COMMAND_METHODS, DATUM_QUERY_METHODS};
mod clients;
pub use clients::CLIENT_COMMAND_METHODS;
mod attachments;
mod attributes;
mod command_owner;
mod edit;
mod file_host;
mod interests;
pub use file_host::{
    DERIVED_COMMAND_METHODS, DocumentObserverFactory, FILE_COMMAND_METHODS, FileHost,
};
mod file_queries;
mod groups_edit;
mod host;
pub use file_queries::{FILE_QUERY_METHODS, FileQueryHost};
mod options;
mod settings_edit;
mod sketches;
pub use edit::EDIT_METHODS;
pub use host::SessionHost;
pub use options::SKETCH_OPTION_METHODS;
pub use sketches::{SKETCH_COMMAND_METHODS, SKETCH_QUERY_METHODS};
mod collection;
mod detail;
mod expressions;
mod parameter_choices;
mod parameter_edit;
mod parameter_tolerance;
mod parameter_update;
mod parameter_value;
mod parameters;
mod properties;
mod transactions;
mod units;
mod units_edit;
use oblikovati_api::{Error, client::Caller, wire::*};
use oblikovati_model::{parameter::ParameterCollection, units::UnitsOfMeasure};
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};
use serde::{Serialize, de::DeserializeOwned};
pub use transactions::TRANSACTION_METHODS;

/// The exact query inventory; version compatibility alone does not imply other methods exist.
pub const QUERY_METHODS: [&str; 31] = [
    METHOD_CLIENT_APPS_LIST,
    METHOD_DOCUMENTS_LIST_ATTACHMENTS,
    METHOD_DOCUMENTS_LIST_INTERESTS,
    METHOD_DOCUMENTS_HAS_INTEREST,
    METHOD_ATTRIBUTES_GET,
    METHOD_ATTRIBUTES_LIST,
    METHOD_ATTRIBUTES_LIST_SETS,
    METHOD_ATTRIBUTES_FIND,
    METHOD_DOCUMENTS_GET_UNITS,
    METHOD_DOCUMENTS_LIST_PROPERTIES,
    METHOD_DOCUMENTS_GET_PROPERTY,
    METHOD_TRANSACTION_STATE,
    METHOD_TRANSACTION_HISTORY,
    METHOD_PARAMETERS_LIST,
    METHOD_PARAMETERS_GET_DETAIL,
    METHOD_PARAMETERS_GET_SETTINGS,
    METHOD_PARAMETERS_GROUPS_LIST,
    METHOD_PARAMETERS_GET,
    METHOD_PARAMETERS_DRIVEN_BY,
    METHOD_PARAMETERS_DEPENDENTS,
    METHOD_UNITS_GET_VALUE_FROM_EXPRESSION,
    METHOD_UNITS_GET_DATABASE_UNITS_FROM_EXPRESSION,
    METHOD_UNITS_IS_EXPRESSION_VALID,
    METHOD_UNITS_COMPATIBLE_UNITS,
    METHOD_UNITS_GET_DRIVING_PARAMETERS,
    METHOD_UNITS_GET_LOCALE_CORRECTED_EXPRESSION,
    METHOD_UNITS_CONVERT,
    METHOD_UNITS_GET_TYPE_FROM_STRING,
    METHOD_UNITS_GET_STRING_FROM_TYPE,
    METHOD_UNITS_GET_STRING_FROM_VALUE,
    METHOD_UNITS_GET_PRECISE_STRING_FROM_VALUE,
];

/// A transport over immutable session state. It cannot record edits or change focus.
pub struct SessionQueryHost<'a, O, W> {
    workspace: &'a WorkspaceSession<O, W>,
}

impl<'a, O: TransactionObserver, W: WorkspaceObserver> SessionQueryHost<'a, O, W> {
    /// Borrow the session owned by the caller; example: `Client::new(SessionQueryHost::new(&workspace))`.
    pub fn new(workspace: &'a WorkspaceSession<O, W>) -> Self {
        Self { workspace }
    }

    fn active_units(&self) -> Result<&UnitsOfMeasure, Error> {
        Ok(self.active_parameters()?.units())
    }

    fn active_parameters(&self) -> Result<&ParameterCollection<u64>, Error> {
        let id = self
            .workspace
            .active_document()
            .ok_or_else(|| Error("unit query: expected an active document".into()))?;
        let session = self
            .workspace
            .session(id)
            .map_err(|error| Error(error.to_string()))?;
        Ok(&session.snapshot().parameters)
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver> Caller for SessionQueryHost<'_, O, W> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        match method {
            METHOD_CLIENT_APPS_LIST => encode(&ListClientApplicationsResult {
                clients: Some(self.workspace.connected_clients().entries().to_vec()),
            }),
            METHOD_DOCUMENTS_LIST_ATTACHMENTS => encode(&attachments::list(
                self.workspace,
                decode(method, request)?,
                None,
            )?),
            METHOD_DOCUMENTS_LIST_INTERESTS => {
                encode(&interests::list(self.workspace, decode(method, request)?)?)
            }
            METHOD_DOCUMENTS_HAS_INTEREST => {
                encode(&interests::has(self.workspace, decode(method, request)?)?)
            }
            METHOD_ATTRIBUTES_GET
            | METHOD_ATTRIBUTES_LIST
            | METHOD_ATTRIBUTES_LIST_SETS
            | METHOD_ATTRIBUTES_FIND => attributes::dispatch(self.workspace, method, request),
            METHOD_DOCUMENTS_LIST_PROPERTIES => {
                encode(&properties::list(self.workspace, decode(method, request)?)?)
            }
            METHOD_DOCUMENTS_GET_PROPERTY => {
                encode(&properties::get(self.workspace, decode(method, request)?)?)
            }
            METHOD_TRANSACTION_HISTORY => encode(&transactions::history(
                self.workspace,
                decode(method, request)?,
            )?),
            METHOD_TRANSACTION_STATE => encode(&transactions::state(self.workspace)?),
            METHOD_PARAMETERS_GET_DETAIL => encode(&detail::read(
                self.active_parameters()?,
                decode(method, request)?,
            )?),
            METHOD_PARAMETERS_GET_SETTINGS => {
                encode(&collection::settings(self.active_parameters()?)?)
            }
            METHOD_PARAMETERS_GROUPS_LIST => {
                encode(&collection::groups(self.active_parameters()?)?)
            }
            METHOD_PARAMETERS_LIST => encode(&parameters::list(self.active_parameters()?)?),
            METHOD_PARAMETERS_GET => encode(&parameters::get(
                self.active_parameters()?,
                decode(method, request)?,
            )?),
            METHOD_PARAMETERS_DRIVEN_BY => encode(&parameters::neighbors(
                self.active_parameters()?,
                decode(method, request)?,
                true,
            )?),
            METHOD_PARAMETERS_DEPENDENTS => encode(&parameters::neighbors(
                self.active_parameters()?,
                decode(method, request)?,
                false,
            )?),
            METHOD_UNITS_GET_VALUE_FROM_EXPRESSION => encode(&expressions::value(
                self.active_parameters()?,
                decode(method, request)?,
            )?),
            METHOD_UNITS_GET_DATABASE_UNITS_FROM_EXPRESSION => encode(&expressions::database(
                self.active_parameters()?,
                decode(method, request)?,
            )?),
            METHOD_UNITS_IS_EXPRESSION_VALID => encode(&expressions::valid(
                self.active_parameters()?,
                decode(method, request)?,
            )?),
            METHOD_UNITS_COMPATIBLE_UNITS => encode(&expressions::compatible(
                self.active_parameters()?,
                decode(method, request)?,
            )?),
            METHOD_UNITS_GET_DRIVING_PARAMETERS => {
                encode(&expressions::references(decode(method, request)?)?)
            }
            METHOD_UNITS_GET_LOCALE_CORRECTED_EXPRESSION => {
                encode(&expressions::canonical(decode(method, request)?)?)
            }
            METHOD_DOCUMENTS_GET_UNITS => encode(&units::document_units(self.active_units()?)),
            METHOD_UNITS_CONVERT => encode(&units::convert(decode(method, request)?)?),
            METHOD_UNITS_GET_TYPE_FROM_STRING => {
                encode(&units::category(decode(method, request)?)?)
            }
            METHOD_UNITS_GET_STRING_FROM_TYPE => encode(&units::preferred(
                self.active_units()?,
                decode(method, request)?,
            )?),
            METHOD_UNITS_GET_STRING_FROM_VALUE => encode(&units::formatted(
                self.active_units()?,
                decode(method, request)?,
                false,
            )?),
            METHOD_UNITS_GET_PRECISE_STRING_FROM_VALUE => encode(&units::formatted(
                self.active_units()?,
                decode(method, request)?,
                true,
            )?),
            _ => Err(Error(format!(
                "method {method:?}: expected a supported read-only method from {QUERY_METHODS:?}"
            ))),
        }
    }
}

fn decode<T: DeserializeOwned + Default>(method: &str, request: &[u8]) -> Result<T, Error> {
    serde_json::from_slice::<serde_json::Value>(request).and_then(|value| {
        if !value.is_object() && !value.is_null() {
            return Err(serde::de::Error::custom("expected a JSON object or null"));
        }
        serde_json::from_value::<Option<T>>(value).map(Option::unwrap_or_default)
    }).map_err(|error| {
        Error(format!("method {method:?} request {:?}: expected its canonical JSON argument object: {error}", String::from_utf8_lossy(request)))
    })
}

fn encode<T: Serialize>(response: &T) -> Result<Vec<u8>, Error> {
    serde_json::to_vec(response).map_err(|error| {
        Error(format!(
            "query response: expected canonical serializable JSON: {error}"
        ))
    })
}

mod derived_edit;
