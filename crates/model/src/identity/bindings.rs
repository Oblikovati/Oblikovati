// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentGuid, ReferenceEnvelope, ReferenceScheme, SemanticReferenceName};
use crate::history::{DocumentEditHistory, EvaluationScope};
use std::collections::{BTreeMap, btree_map::Entry};

/// An immutable exact-name index for one document evaluation. It owns no geometry recovery policy.
pub struct ExactReferenceBindings<Entity> {
    document: DocumentGuid,
    scope: EvaluationScope,
    entries: BTreeMap<Vec<u8>, Entity>,
}
impl<Entity> ExactReferenceBindings<Entity> {
    /// Admit names and entities together; duplicate names refuse the entire index.
    /// Example: `ExactReferenceBindings::from_entries(document, job.scope().clone(), entries)?`.
    pub fn from_entries(
        document: DocumentGuid,
        scope: EvaluationScope,
        entries: impl IntoIterator<Item = (Vec<u8>, Entity)>,
    ) -> Result<Self, String> {
        require_live_scope(&scope)?;
        let mut indexed = BTreeMap::new();
        for (encoded, entity) in entries {
            insert_exact_entry(&mut indexed, document, &encoded, entity)?;
        }
        require_live_scope(&scope)?;
        Ok(Self {
            document,
            scope,
            entries: indexed,
        })
    }
    /// Resolve only within the current owning history and requested kind; example: `bindings.bind(&history, document, key, 1)?`.
    /// The returned borrow prevents editing this history while the entity is in use.
    /// ```compile_fail
    /// use oblikovati_model::{history::DocumentEditHistory, identity::{DocumentGuid, ExactReferenceBindings}};
    /// fn edit_while_borrowed(index: &ExactReferenceBindings<()>, history: &mut DocumentEditHistory, document: DocumentGuid, key: &[u8]) {
    ///     let entity = index.bind(history, document, key, 1).unwrap();
    ///     history.refresh(|_| Ok::<(), String>(())).unwrap();
    ///     assert_eq!(*entity, ());
    /// }
    /// ```
    pub fn bind<'a>(
        &'a self,
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
        encoded: &[u8],
        expected_kind: u32,
    ) -> Result<&'a Entity, String> {
        self.require_owner(history, document)?;
        let envelope = decoded_binding_key(encoded, document)?;
        if envelope.kind != expected_kind {
            return Err(format!(
                "reference kind {}: expected {expected_kind}",
                envelope.kind
            ));
        }
        let canonical = envelope
            .encode_current()
            .map_err(|cause| cause.to_string())?;
        self.entries
            .get(&canonical)
            .ok_or_else(|| missing_reference(&envelope))
    }
    /// Read canonical names in byte order, with the same freshness guard as binding.
    /// Example: `bindings.encoded_keys(&history, document)?` supplies saved reference bytes.
    pub fn encoded_keys<'a>(
        &'a self,
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<impl Iterator<Item = &'a [u8]>, String> {
        self.require_owner(history, document)?;
        Ok(self.entries.keys().map(Vec::as_slice))
    }
    fn require_owner(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<(), String> {
        if self.document != document {
            return Err(format!(
                "reference document {document}: expected {}",
                self.document
            ));
        }
        if !self.scope.is_current(history) {
            return Err(format!(
                "reference evaluation generation {}: expected its current owning history",
                self.scope.generation()
            ));
        }
        Ok(())
    }
}
fn require_live_scope(scope: &EvaluationScope) -> Result<(), String> {
    if scope.is_superseded() {
        return Err(format!(
            "reference evaluation generation {}: expected a live unchanged owner",
            scope.generation()
        ));
    }
    Ok(())
}
fn decoded_binding_key(
    encoded: &[u8],
    document: DocumentGuid,
) -> Result<ReferenceEnvelope<'_>, String> {
    let envelope = ReferenceEnvelope::decode(encoded).map_err(|cause| cause.to_string())?;
    // #3563: these persisted values are Go model/identity kinds, not kernel/topo kinds.
    if !matches!(envelope.kind, 1..=4 | 10 | 11 | 20 | 21) {
        return Err(format!(
            "reference kind {}: expected face/edge/vertex/body (1..4), feature (10), parameter (11), document (20), or external (21)",
            envelope.kind
        ));
    }
    if envelope.scheme == ReferenceScheme::Semantic {
        require_semantic_document(encoded, document)?;
    }
    Ok(envelope)
}
fn require_semantic_document(encoded: &[u8], expected: DocumentGuid) -> Result<(), String> {
    let actual = SemanticReferenceName::decode(encoded)?.document();
    if actual != expected {
        return Err(format!(
            "semantic reference document {actual}: expected {expected}"
        ));
    }
    Ok(())
}
fn insert_exact_entry<Entity>(
    entries: &mut BTreeMap<Vec<u8>, Entity>,
    document: DocumentGuid,
    encoded: &[u8],
    entity: Entity,
) -> Result<(), String> {
    let envelope = decoded_binding_key(encoded, document)?;
    let canonical = envelope
        .encode_current()
        .map_err(|cause| cause.to_string())?;
    match entries.entry(canonical) {
        Entry::Vacant(slot) => {
            slot.insert(entity);
            Ok(())
        }
        Entry::Occupied(_) => Err(format!(
            "reference context {}, kind {}, payload {:?}: expected one entity per exact name",
            envelope.context, envelope.kind, envelope.payload
        )),
    }
}
fn missing_reference(envelope: &ReferenceEnvelope<'_>) -> String {
    format!(
        "reference context {}, kind {}, payload {:?}: expected an exact entity in the current evaluation",
        envelope.context, envelope.kind, envelope.payload
    )
}
