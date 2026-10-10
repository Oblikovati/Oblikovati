// SPDX-License-Identifier: GPL-2.0-only
use crate::{decode, encode};
use oblikovati_api::{Error, wire::*};
use oblikovati_session::clients::ConnectedClients;

/// Application connection changes are independent from document transactions.
pub const CLIENT_COMMAND_METHODS: [&str; 2] =
    [METHOD_CLIENT_APPS_REGISTER, METHOD_CLIENT_APPS_UNREGISTER];

pub(crate) fn apply(
    clients: &mut ConnectedClients,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    match method {
        METHOD_CLIENT_APPS_REGISTER => {
            let request: RegisterClientApplicationArgs = decode(method, request)?;
            let id = clients
                .register(&request.name)
                .map_err(|error| Error(error.to_string()))?;
            encode(&RegisterClientApplicationResult { id })
        }
        METHOD_CLIENT_APPS_UNREGISTER => {
            let request: UnregisterClientApplicationArgs = decode(method, request)?;
            let response = encode(&OKResult { ok: true })?;
            clients
                .unregister(request.id)
                .map_err(|error| Error(error.to_string()))?;
            Ok(response)
        }
        _ => Err(Error(format!(
            "client method {method:?}: expected one of {CLIENT_COMMAND_METHODS:?}"
        ))),
    }
}
