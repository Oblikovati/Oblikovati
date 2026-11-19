// SPDX-License-Identifier: GPL-2.0-only
use super::{
    DocumentGuid, ReferenceEnvelope, ReferenceScheme, SemanticReferenceName,
    topology_reference_document,
};
use crate::history::{DocumentEditHistory, EvaluationScope};
use std::collections::{BTreeMap, btree_map::Entry};
use std::sync::Arc;

/// An immutable exact-name index for one document evaluation. It owns no geometry recovery policy.
#[derive(Clone)]
pub struct ExactReferenceBindings<Entity> {
    document: DocumentGuid,
    scope: EvaluationScope,
    entries: Arc<BTreeMap<Vec<u8>, Entity>>,
}
/// An exact index prepared for one private edit (#4122).
/// Example: `PreparedExactReferenceBindings::new(&edit, document, entries)?`.
pub struct PreparedExactReferenceBindings<Entity> {
    seal: crate::history::PreparedCandidateSeal,
    bindings: ExactReferenceBindings<Entity>,
}

impl<Entity> PreparedExactReferenceBindings<Entity> {
    /// Validate every name before the candidate can be published.
    /// Example: `PreparedExactReferenceBindings::new(&edit, document, entries)?`.
    pub fn new(
        candidate: &crate::history::PreparedPrivateEdit<'_>,
        document: DocumentGuid,
        entries: impl IntoIterator<Item = (Vec<u8>, Entity)>,
    ) -> Result<Self, String> {
        let bindings = ExactReferenceBindings::from_entries(
            document,
            candidate.source().evaluation_scope(),
            entries,
        )?;
        for (key, entity) in bindings.entries.iter() {
            let kind = ReferenceEnvelope::decode(key)
                .map_err(|error| error.to_string())?
                .kind;
            let resolved = bindings.bind(candidate.source(), document, key, kind)?;
            if !std::ptr::eq(entity, resolved) {
                return Err(format!(
                    "prepared name {key:?}: expected the captured entity"
                ));
            }
        }
        Ok(Self {
            seal: candidate.seal(),
            bindings,
        })
    }

    /// Prepare names for a dependent history under the same source candidate.
    /// Example: `PreparedExactReferenceBindings::for_dependent(&dependent, document, entries)?`.
    pub fn for_dependent(
        dependent: &crate::history::PreparedDependentRefresh<'_>,
        document: DocumentGuid,
        entries: impl IntoIterator<Item = (Vec<u8>, Entity)>,
    ) -> Result<Self, String> {
        let bindings = ExactReferenceBindings::from_entries(
            document,
            dependent.source().evaluation_scope(),
            entries,
        )?;
        for (key, entity) in bindings.entries.iter() {
            let kind = ReferenceEnvelope::decode(key)
                .map_err(|error| error.to_string())?
                .kind;
            let resolved = bindings.bind(dependent.source(), document, key, kind)?;
            if !std::ptr::eq(entity, resolved) {
                return Err(format!(
                    "prepared name {key:?}: expected the captured entity"
                ));
            }
        }
        Ok(Self {
            seal: dependent.seal(),
            bindings,
        })
    }

    /// Reject another candidate before publishing either the edit or its index.
    /// Example: `let (index, transition) = names.publish(edit)?;`.
    pub fn publish(
        self,
        candidate: crate::history::PreparedPrivateEdit<'_>,
    ) -> Result<
        (
            ExactReferenceBindings<Entity>,
            crate::history::EvaluationTransition,
        ),
        String,
    > {
        crate::history::publish_prepared(candidate, self)
    }
}

impl<Entity> crate::history::publication::private::Sealed
    for PreparedExactReferenceBindings<Entity>
{
}
impl<Entity> crate::history::PreparedPublication for PreparedExactReferenceBindings<Entity> {
    type Output = ExactReferenceBindings<Entity>;
    fn validate(&self, candidate: &crate::history::PreparedPrivateEdit<'_>) -> Result<(), String> {
        if !self.seal.matches(candidate) {
            return Err("prepared name bindings: expected their exact private candidate".into());
        }
        if self.bindings.scope.is_superseded() {
            return Err("prepared name bindings: expected the unchanged source history".into());
        }
        Ok(())
    }
    fn validate_target(&self, source: &EvaluationScope) -> Result<(), String> {
        if !self.bindings.scope.same_scope(source) {
            return Err("prepared name bindings: expected their exact target history".into());
        }
        Ok(())
    }
    fn install(mut self, token: &crate::history::PublicationToken) -> Self::Output {
        self.bindings.scope = token.scope();
        self.bindings
    }
}

impl<Entity> ExactReferenceBindings<Entity> {
    /// Copy exact entries while their original owner remains current.
    /// Example: `bindings.capture_entries(&history, document)?` before an edit.
    pub fn capture_entries(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Vec<(Vec<u8>, Entity)>, String>
    where
        Entity: Clone,
    {
        self.require_owner(history, document)?;
        let entries = self
            .entries
            .iter()
            .map(|(key, entity)| (key.clone(), entity.clone()))
            .collect();
        self.require_owner(history, document)?;
        Ok(entries)
    }
    /// Copy an index while retaining its current generation and document guard.
    /// Example: `bindings.clone_current(&history, document)?` before a transaction.
    pub fn clone_current(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Self, String>
    where
        Entity: Clone,
    {
        self.require_owner(history, document)?;
        let entries = self.entries.clone();
        self.require_owner(history, document)?;
        Ok(Self {
            document: self.document,
            scope: self.scope.clone(),
            entries,
        })
    }
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
            entries: Arc::new(indexed),
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
    let actual = match envelope.scheme {
        ReferenceScheme::Semantic => Some(SemanticReferenceName::decode(encoded)?.document()),
        ReferenceScheme::Derived => Some(topology_reference_document(encoded)?),
        ReferenceScheme::Legacy | ReferenceScheme::Current => None,
    };
    if let Some(actual) = actual
        && actual != document
    {
        return Err(format!("reference document {actual}: expected {document}"));
    }
    Ok(envelope)
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
