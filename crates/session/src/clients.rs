// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::wire::ClientApplicationInfo;
use std::{error::Error, fmt};

/// Connected external drivers, distinct from loaded in-process add-ins.
#[derive(Default)]
pub struct ConnectedClients {
    high_water: i64,
    entries: Vec<ClientApplicationInfo>,
}

/// Failed client registration or removal; the registry is unchanged.
#[derive(Debug, PartialEq, Eq)]
pub enum ClientRegistrationError {
    EmptyName,
    Exhausted(i64),
    Unknown(i64),
}

impl ConnectedClients {
    /// Inspect connected clients in registration order; example: `clients.entries()`.
    pub fn entries(&self) -> &[ClientApplicationInfo] {
        &self.entries
    }

    /// Register a display name; duplicate names represent independent connections.
    /// Example: `let id = clients.register("cam")?;`.
    pub fn register(&mut self, name: &str) -> Result<i64, ClientRegistrationError> {
        if name.is_empty() {
            return Err(ClientRegistrationError::EmptyName);
        }
        let id = self
            .high_water
            .checked_add(1)
            .ok_or(ClientRegistrationError::Exhausted(self.high_water))?;
        self.entries.push(ClientApplicationInfo {
            id,
            name: name.into(),
        });
        self.high_water = id;
        Ok(id)
    }

    /// Remove exactly one connection without reusing its ID; example: `clients.unregister(id)?;`.
    pub fn unregister(&mut self, id: i64) -> Result<(), ClientRegistrationError> {
        let index = self
            .entries
            .iter()
            .position(|entry| entry.id == id)
            .ok_or(ClientRegistrationError::Unknown(id))?;
        self.entries.remove(index);
        Ok(())
    }

    /// Match an exact connected name, including when multiple connections share it.
    /// Example: `assert!(!ConnectedClients::default().contains_name("cam"));`.
    pub fn contains_name(&self, name: &str) -> bool {
        self.entries.iter().any(|entry| entry.name == name)
    }
}

impl fmt::Display for ClientRegistrationError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::EmptyName => write!(output, "client name \"\": expected a nonempty display name"),
            Self::Exhausted(value) => write!(
                output,
                "client ID high-water {value}: expected room for another positive i64 ID"
            ),
            Self::Unknown(id) => write!(output, "client ID {id}: expected a registered connection"),
        }
    }
}
impl Error for ClientRegistrationError {}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn exhaustion_never_wraps_or_changes_registered_clients() {
        let mut clients = ConnectedClients {
            high_water: i64::MAX - 1,
            entries: vec![],
        };
        assert_eq!(clients.register("last").unwrap(), i64::MAX);
        let saved = clients.entries().to_vec();
        let error = clients.register("overflow").unwrap_err();
        assert_eq!(error, ClientRegistrationError::Exhausted(i64::MAX));
        assert!(error.to_string().contains(&i64::MAX.to_string()));
        assert_eq!(clients.entries(), saved);
    }
}
