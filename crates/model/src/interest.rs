// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{DocumentInterestRecord, INTEREST_NONE, INTERESTED};
use std::{error::Error, fmt};

/// Ordered add-in discovery records; each (client ID, name) pair occurs once.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct DocumentInterests {
    records: Vec<DocumentInterestRecord>,
}

/// Invalid interest identity or unsupported strength, rejected before mutation.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct InterestError(String);

impl DocumentInterests {
    /// Inspect insertion order without allowing unvalidated mutation.
    /// Example: `let records = interests.records();`.
    pub fn records(&self) -> &[DocumentInterestRecord] {
        &self.records
    }

    /// Register owned metadata, or replace the same client/name in its original position.
    /// Zero strength defaults to interested; client payload and version are opaque.
    /// Example: `interests.put(record)?;`.
    pub fn put(&mut self, mut record: DocumentInterestRecord) -> Result<(), InterestError> {
        validate(&record)?;
        if record.interest_type.0 == 0 {
            record.interest_type = INTERESTED;
        }
        if let Some(index) = self.position(&record.client_id, &record.name) {
            self.records[index] = record;
            return Ok(());
        }
        self.records.push(record);
        Ok(())
    }

    /// Probe by exact client ID or interest name, including withdrawn interests.
    /// Example: `assert!(!DocumentInterests::default().has_interest("cam"));`.
    pub fn has_interest(&self, client_or_name: &str) -> bool {
        self.records
            .iter()
            .any(|record| record.client_id == client_or_name || record.name == client_or_name)
    }

    /// Remove one identity while retaining the order of all remaining records.
    /// Example: `let removed = interests.remove("cam", "toolpaths");`.
    pub fn remove(&mut self, client: &str, name: &str) -> bool {
        let Some(index) = self.position(client, name) else {
            return false;
        };
        self.records.remove(index);
        true
    }

    fn position(&self, client: &str, name: &str) -> Option<usize> {
        self.records
            .iter()
            .position(|record| record.client_id == client && record.name == name)
    }
}

fn validate(record: &DocumentInterestRecord) -> Result<(), InterestError> {
    if record.client_id.is_empty() || record.name.is_empty() {
        return Err(InterestError(format!(
            "interest ({:?}, {:?}): expected nonempty client ID and name",
            record.client_id, record.name
        )));
    }
    if record.interest_type.0 != 0 && ![INTEREST_NONE, INTERESTED].contains(&record.interest_type) {
        return Err(InterestError(format!(
            "interest strength {}: expected 0, {} or {}",
            record.interest_type.0, INTEREST_NONE.0, INTERESTED.0
        )));
    }
    Ok(())
}

impl fmt::Display for InterestError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.fmt(formatter)
    }
}
impl Error for InterestError {}
