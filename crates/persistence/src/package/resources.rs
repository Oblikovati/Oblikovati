// SPDX-License-Identifier: GPL-2.0-only

use super::binary_text::decode_base64;
use super::{Mapping, PackageCodecError, Value, codec_error};
use serde::{Deserialize, de::DeserializeOwned};
use std::collections::BTreeMap;

/// Resource classification follows its encoding, never its extensible type tag.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum ResourceContents {
    Utf8(String),
    Binary(Vec<u8>),
    Embedded,
}

/// A self-contained imported resource or a reference to an application resource.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct PackageResource {
    pub kind: String,
    pub origin: String,
    pub contents: ResourceContents,
}

#[derive(Default, Deserialize)]
#[serde(default)]
struct StoredResource {
    #[serde(rename = "type")]
    kind: String,
    encoding: String,
    origin: String,
    value: String,
}

pub(super) fn decode_streams(
    root: &Mapping,
) -> Result<BTreeMap<String, Vec<u8>>, PackageCodecError> {
    let encoded: BTreeMap<String, String> = section_entries(root, "data")?;
    encoded
        .into_iter()
        .map(|(name, value)| {
            decode_base64(&format!("data stream {name:?}"), &value).map(|bytes| (name, bytes))
        })
        .collect()
}

pub(super) fn decode_resources(
    root: &Mapping,
) -> Result<BTreeMap<String, PackageResource>, PackageCodecError> {
    let encoded: BTreeMap<String, StoredResource> = section_entries(root, "resources")?;
    encoded
        .into_iter()
        .map(|(name, record)| decode_resource(&name, record).map(|resource| (name, resource)))
        .collect()
}

fn section_entries<Entry: DeserializeOwned>(
    root: &Mapping,
    name: &str,
) -> Result<BTreeMap<String, Entry>, PackageCodecError> {
    match root.get(Value::String(name.to_owned())) {
        None | Some(Value::Null) => Ok(BTreeMap::new()),
        Some(section) => serde_yaml_ng::from_value(section.clone()).map_err(codec_error),
    }
}

fn decode_resource(
    name: &str,
    record: StoredResource,
) -> Result<PackageResource, PackageCodecError> {
    let contents = match record.encoding.as_str() {
        "utf8" => ResourceContents::Utf8(record.value),
        "base64" => {
            ResourceContents::Binary(decode_base64(&format!("resource {name:?}"), &record.value)?)
        }
        "embedded" if record.value.is_empty() => ResourceContents::Embedded,
        encoding => {
            return Err(PackageCodecError(format!(
                "resource {name:?} encoding {encoding:?} and value {:?}: expected utf8, base64, or embedded with no stored value",
                record.value
            )));
        }
    };
    Ok(PackageResource {
        kind: record.kind,
        origin: record.origin,
        contents,
    })
}
