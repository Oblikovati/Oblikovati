// SPDX-License-Identifier: GPL-2.0-only

//! Typed query dispatch; edits and native add-in transport are separate host responsibilities.
mod units;
use oblikovati_api::{Error, client::Caller, wire::*};
use oblikovati_model::units::UnitsOfMeasure;
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};
use serde::{Serialize, de::DeserializeOwned};

/// The exact query inventory; version compatibility alone does not imply other methods exist.
pub const QUERY_METHODS: [&str; 6] = [
    METHOD_DOCUMENTS_GET_UNITS,
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
        let id = self
            .workspace
            .active_document()
            .ok_or_else(|| Error("unit query: expected an active document".into()))?;
        let session = self
            .workspace
            .session(id)
            .map_err(|error| Error(error.to_string()))?;
        Ok(session.snapshot().parameters.units())
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver> Caller for SessionQueryHost<'_, O, W> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        match method {
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
    serde_json::from_slice::<Option<T>>(request).map(Option::unwrap_or_default).map_err(|error| {
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
