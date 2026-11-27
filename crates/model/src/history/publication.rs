// SPDX-License-Identifier: GPL-2.0-only
//! Publish only components validated for the same private edit (#4122).

use super::{EvaluationScope, EvaluationTransition, PreparedPrivateEdit};

pub(crate) mod private {
    pub trait Sealed {}
}

/// An unforgeable publication capability passed only after source history accepts an edit.
#[doc(hidden)]
pub struct PublicationToken(EvaluationScope);
impl PublicationToken {
    pub(crate) fn for_scope(scope: EvaluationScope) -> Self {
        Self(scope)
    }
    pub(crate) fn scope(&self) -> EvaluationScope {
        self.0.clone()
    }
}

/// A source-checked part of one prepared publication.
/// Example: pair prepared name bindings with a prepared dependent refresh.
pub trait PreparedPublication: private::Sealed {
    /// The installed value, returned after all components publish.
    type Output;
    /// Check the private edit without changing any owner.
    fn validate(&self, edit: &PreparedPrivateEdit<'_>) -> Result<(), String>;
    /// Confirm that name bindings target the owning document history.
    fn validate_target(&self, _source: &EvaluationScope) -> Result<(), String> {
        Ok(())
    }
    /// Install a checked component; the token cannot be constructed by callers.
    #[doc(hidden)]
    fn install(self, token: &PublicationToken) -> Self::Output;
}

/// Validate every component before publishing a source edit and its consequences.
/// Example: `publish_prepared(edit, (names, dependent))?`.
pub fn publish_prepared<P: PreparedPublication>(
    edit: PreparedPrivateEdit<'_>,
    publication: P,
) -> Result<(P::Output, EvaluationTransition), String> {
    publication.validate(&edit)?;
    publication.validate_target(&edit.source().evaluation_scope())?;
    let transition = edit.publish();
    let token = PublicationToken(transition.target_scope());
    Ok((publication.install(&token), transition))
}

impl<A: PreparedPublication, B: PreparedPublication> private::Sealed for (A, B) {}
impl<A: PreparedPublication, B: PreparedPublication> PreparedPublication for (A, B) {
    type Output = (A::Output, B::Output);
    fn validate(&self, edit: &PreparedPrivateEdit<'_>) -> Result<(), String> {
        self.0.validate(edit)?;
        self.1.validate(edit)
    }
    fn validate_target(&self, source: &EvaluationScope) -> Result<(), String> {
        self.0.validate_target(source)?;
        self.1.validate_target(source)
    }
    fn install(self, token: &PublicationToken) -> Self::Output {
        (self.0.install(token), self.1.install(token))
    }
}

impl<P: PreparedPublication> private::Sealed for Vec<P> {}
impl<P: PreparedPublication> PreparedPublication for Vec<P> {
    type Output = Vec<P::Output>;
    fn validate(&self, edit: &PreparedPrivateEdit<'_>) -> Result<(), String> {
        for part in self {
            part.validate(edit)?;
        }
        Ok(())
    }
    fn validate_target(&self, source: &EvaluationScope) -> Result<(), String> {
        for part in self {
            part.validate_target(source)?;
        }
        Ok(())
    }
    fn install(self, token: &PublicationToken) -> Self::Output {
        self.into_iter().map(|part| part.install(token)).collect()
    }
}

impl<P: PreparedPublication> private::Sealed for Option<P> {}
impl<P: PreparedPublication> PreparedPublication for Option<P> {
    type Output = Option<P::Output>;
    fn validate(&self, edit: &PreparedPrivateEdit<'_>) -> Result<(), String> {
        match self {
            Some(part) => part.validate(edit),
            None => Ok(()),
        }
    }
    fn validate_target(&self, source: &EvaluationScope) -> Result<(), String> {
        match self {
            Some(part) => part.validate_target(source),
            None => Ok(()),
        }
    }
    fn install(self, token: &PublicationToken) -> Self::Output {
        self.map(|part| part.install(token))
    }
}

impl private::Sealed for () {}
impl PreparedPublication for () {
    type Output = ();
    fn validate(&self, _: &PreparedPrivateEdit<'_>) -> Result<(), String> {
        Ok(())
    }
    fn install(self, _: &PublicationToken) -> Self::Output {}
}

/// A value captured for one exact private edit.
/// Example: `PreparedValue::new(&edit, body_metadata)`.
pub struct PreparedValue<T> {
    seal: super::PreparedCandidateSeal,
    value: T,
}
impl<T> PreparedValue<T> {
    /// Seal a value to its candidate; example: `PreparedValue::new(&edit, metadata)`.
    pub fn new(edit: &PreparedPrivateEdit<'_>, value: T) -> Self {
        Self {
            seal: edit.seal(),
            value,
        }
    }
}
impl<T> private::Sealed for PreparedValue<T> {}
impl<T> PreparedPublication for PreparedValue<T> {
    type Output = T;
    fn validate(&self, edit: &PreparedPrivateEdit<'_>) -> Result<(), String> {
        if !self.seal.matches(edit) {
            return Err("prepared value: expected its exact private candidate".into());
        }
        Ok(())
    }
    fn install(self, _: &PublicationToken) -> T {
        self.value
    }
}

/// Select one prepared result without losing its concrete publication type.
/// Example: `PreparedChoice::First(prism)` for a prism result.
pub enum PreparedChoice<A, B> {
    First(A),
    Second(B),
}
impl<A: PreparedPublication, B: PreparedPublication> private::Sealed for PreparedChoice<A, B> {}
impl<A: PreparedPublication, B: PreparedPublication> PreparedPublication for PreparedChoice<A, B> {
    type Output = PreparedChoice<A::Output, B::Output>;
    fn validate(&self, edit: &PreparedPrivateEdit<'_>) -> Result<(), String> {
        match self {
            Self::First(value) => value.validate(edit),
            Self::Second(value) => value.validate(edit),
        }
    }
    fn validate_target(&self, source: &EvaluationScope) -> Result<(), String> {
        match self {
            Self::First(value) => value.validate_target(source),
            Self::Second(value) => value.validate_target(source),
        }
    }
    fn install(self, token: &PublicationToken) -> Self::Output {
        match self {
            Self::First(value) => PreparedChoice::First(value.install(token)),
            Self::Second(value) => PreparedChoice::Second(value.install(token)),
        }
    }
}

/// Assemble already validated publication parts without further fallible work.
/// Example: `PreparedMapped::new(parts, |parts, scope| assemble(parts, scope))`.
pub struct PreparedMapped<P, F> {
    parts: P,
    assemble: F,
}
impl<P, F> PreparedMapped<P, F> {
    /// Retain the parts and their infallible assembly function.
    /// Example: `PreparedMapped::new(parts, assemble)`.
    pub fn new(parts: P, assemble: F) -> Self {
        Self { parts, assemble }
    }
}
impl<P: PreparedPublication, F> private::Sealed for PreparedMapped<P, F> {}
impl<P: PreparedPublication, F, U> PreparedPublication for PreparedMapped<P, F>
where
    F: FnOnce(P::Output, EvaluationScope) -> U,
{
    type Output = U;
    fn validate(&self, edit: &PreparedPrivateEdit<'_>) -> Result<(), String> {
        self.parts.validate(edit)
    }
    fn validate_target(&self, source: &EvaluationScope) -> Result<(), String> {
        self.parts.validate_target(source)
    }
    fn install(self, token: &PublicationToken) -> U {
        (self.assemble)(self.parts.install(token), token.scope())
    }
}
