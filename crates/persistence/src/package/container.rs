// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl DocumentPackage {
    pub(crate) fn empty_container() -> Self {
        Self {
            manifest: PackageManifest {
                schema_version: 0,
                document_type: 0,
                sub_type: String::new(),
                display_name: String::new(),
            },
            root: Mapping::new(),
            streams: BTreeMap::new(),
            resources: BTreeMap::new(),
        }
    }

    pub(crate) fn decode_container(encoded: &[u8]) -> Result<Self, PackageCodecError> {
        let root: Mapping = serde_yaml_ng::from_slice(encoded).map_err(codec_error)?;
        let mut projection = root.clone();
        projection.entry("schemaVersion".into()).or_insert(0.into());
        projection.entry("documentType".into()).or_insert(0.into());
        let manifest: PackageManifest =
            serde_yaml_ng::from_value(Value::Mapping(projection)).map_err(codec_error)?;
        // Go skips migration only for a zero manifest; this exception is confined
        // to DataIO (#3681), never used by document registration or recipe loading.
        if manifest.schema_version == 0
            && manifest.document_type == 0
            && manifest.display_name.is_empty()
            && manifest.sub_type.is_empty()
        {
            return Self::from_root(root, manifest);
        }
        Self::decode(encoded)
    }
}
