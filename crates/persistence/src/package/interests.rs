// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentPackage, PackageCodecError, Value, codec_error};
use oblikovati_api::types::{DocumentInterestRecord, INTEREST_NONE, INTERESTED};
use oblikovati_model::interest::DocumentInterests;
use serde::{Deserialize, Serialize};
use std::collections::BTreeSet;

// YAML uses enum spellings; the API record uses numeric enum values.
#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields, rename_all = "camelCase")]
struct StoredInterest {
    client_id: String,
    name: String,
    #[serde(default)]
    interest_type: String,
    #[serde(default)]
    data_version: i64,
    #[serde(default)]
    client_data: String,
}

impl DocumentPackage {
    /// Restore validated add-in discovery records in their stored order.
    /// Example: `let interests = package.restore_interests()?;`.
    pub fn restore_interests(&self) -> Result<DocumentInterests, PackageCodecError> {
        let records: Vec<StoredInterest> = match self.root.get("interests") {
            None | Some(Value::Null) => Vec::new(),
            Some(section) => serde_yaml_ng::from_value(section.clone()).map_err(codec_error)?,
        };
        let mut interests = DocumentInterests::default();
        let mut identities = BTreeSet::new();
        for record in records {
            let identity = (record.client_id.clone(), record.name.clone());
            if !identities.insert(identity.clone()) {
                return Err(PackageCodecError(format!(
                    "interest {identity:?}: expected a unique client/name pair"
                )));
            }
            interests.put(record.restore()?).map_err(interest_error)?;
        }
        Ok(interests)
    }

    /// Replace interest records without changing other sections or the model digest.
    /// Example: `let updated = package.with_interests(&interests)?;`.
    pub fn with_interests(&self, interests: &DocumentInterests) -> Result<Self, PackageCodecError> {
        self.restore_interests()?;
        let mut candidate = self.clone();
        candidate.root.remove("interests");
        if !interests.records().is_empty() {
            let records: Vec<StoredInterest> = interests
                .records()
                .iter()
                .map(StoredInterest::capture)
                .collect();
            candidate.root.insert(
                "interests".into(),
                serde_yaml_ng::to_value(records).map_err(codec_error)?,
            );
        }
        Ok(candidate)
    }
}

impl StoredInterest {
    fn restore(self) -> Result<DocumentInterestRecord, PackageCodecError> {
        let strength = match self.interest_type.as_str() {
            "" | "interested" => INTERESTED,
            "notInterested" => INTEREST_NONE,
            other => {
                return Err(PackageCodecError(format!(
                    "interest strength {other:?}: expected interested or notInterested"
                )));
            }
        };
        Ok(DocumentInterestRecord {
            client_id: self.client_id,
            name: self.name,
            interest_type: strength,
            data_version: self.data_version,
            client_data: self.client_data,
        })
    }

    fn capture(record: &DocumentInterestRecord) -> Self {
        Self {
            client_id: record.client_id.clone(),
            name: record.name.clone(),
            interest_type: if record.interest_type == INTERESTED {
                "interested"
            } else {
                "notInterested"
            }
            .into(),
            data_version: record.data_version,
            client_data: record.client_data.clone(),
        }
    }
}

fn interest_error(error: impl std::fmt::Display) -> PackageCodecError {
    PackageCodecError(format!("document interests: {error}"))
}
