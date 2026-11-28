// SPDX-License-Identifier: GPL-2.0-only
//! Collect current named input entities for detached feature operations.
use super::*;

pub(in crate::prism::named::blend) struct NamedInputSources {
    pub body: Shape,
    pub entities: Vec<(Vec<u8>, Shape)>,
    pub body_keys: Vec<Vec<u8>>,
}
impl BlendNamedTopology {
    pub(in crate::prism::named::blend) fn source_topology(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<NamedInputSources, String> {
        let mut body_keys = Vec::new();
        let mut entities = Vec::new();
        for key in self.bindings.encoded_keys(history, document)? {
            let kind = ReferenceEnvelope::decode(key)
                .map_err(|cause| cause.to_string())?
                .kind;
            if kind == 4 {
                body_keys.push(key.to_vec());
                continue;
            }
            entities.push((
                key.to_vec(),
                self.bindings.bind(history, document, key, kind)?.clone(),
            ));
        }
        let first = body_keys
            .first()
            .ok_or("derived result: expected a named body")?;
        let body = self.bindings.bind(history, document, first, 4)?.clone();
        Ok(NamedInputSources {
            body,
            entities,
            body_keys,
        })
    }
}
