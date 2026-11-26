// SPDX-License-Identifier: GPL-2.0-only
//! Authored sources for a Sweep, with current geometry resolved during replay.
use super::{
    FeatureReadFootprint, FeatureReadKey,
    sketch_prism::{SketchDirectedCurve, SketchPrismSelection, validation::validate_selection},
    work_geometry::{DatumAngle, DatumLength, WorkGeometryEvaluation},
};
use crate::{
    document_sketch::DocumentSketchId,
    history::{DocumentSketchPlacementError, DocumentSnapshot, ResolvedDocumentSketch},
    sketch::{OwnedSketchGeometry, SketchEntityId, SketchError},
};
use std::{collections::BTreeSet, fmt};

const VERSION: u32 = 1;
const TAPER_VERSION: u32 = 2;

/// Persisted profile boundaries and path traversal, independent of browser order.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchSweepRecipe {
    pub version: u32,
    pub profile_sketch: DocumentSketchId,
    pub profile: SketchPrismSelection,
    pub path_sketch: DocumentSketchId,
    pub path: Vec<SketchDirectedCurve>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub taper: Option<DatumAngle>,
}

/// A validated Sweep recipe without evaluated or cached geometry.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchSweepDefinition(SketchSweepRecipe);

/// The current, converged geometry of both source sketches.
#[derive(Debug)]
pub struct ResolvedSketchSweep<'a> {
    pub profile: ResolvedDocumentSketch<'a>,
    pub boundary: &'a SketchPrismSelection,
    pub path: ResolvedDocumentSketch<'a>,
    pub traversal: &'a [SketchDirectedCurve],
    pub taper: f64,
}

/// Source and placement failures have different remedies when editing a Sweep.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchSweepError {
    Definition(String),
    Sketch {
        owner: DocumentSketchId,
        cause: SketchError,
    },
    Placement {
        owner: DocumentSketchId,
        cause: Box<DocumentSketchPlacementError>,
    },
}

impl fmt::Display for SketchSweepError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Definition(message) => output.write_str(message),
            Self::Sketch { owner, cause } => write!(output, "sweep source {owner:?}: {cause}"),
            Self::Placement { owner, cause } => {
                write!(output, "sweep placement {owner:?}: {cause}")
            }
        }
    }
}

impl std::error::Error for SketchSweepError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Definition(_) => None,
            Self::Sketch { cause, .. } => Some(cause),
            Self::Placement { cause, .. } => Some(cause.as_ref()),
        }
    }
}

impl SketchSweepDefinition {
    /// Store exact curve identities and traversal. Example: `SketchSweepDefinition::new(profile_sketch, boundary, path_sketch, chain)?`.
    pub fn new(
        profile_sketch: DocumentSketchId,
        profile: SketchPrismSelection,
        path_sketch: DocumentSketchId,
        path: Vec<SketchDirectedCurve>,
    ) -> Result<Self, SketchSweepError> {
        Self::from_recipe(SketchSweepRecipe {
            version: VERSION,
            profile_sketch,
            profile,
            path_sketch,
            path,
            taper: None,
        })
    }

    /// Set or replace the taper driver. Example: `untapered.with_taper(angle)?`.
    pub fn with_taper(self, angle: DatumAngle) -> Result<Self, SketchSweepError> {
        Self::from_recipe(SketchSweepRecipe {
            version: TAPER_VERSION,
            taper: Some(angle),
            ..self.0
        })
    }

    /// Compare exact persisted sources. Example: `old.same_recipe(&candidate)` during replay planning.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self == other
            && match (self.0.taper, other.0.taper) {
                (Some(first), Some(second)) => first.same_recipe(second),
                _ => true,
            }
    }

    /// Validate saved sources before use. Example: `SketchSweepDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: SketchSweepRecipe) -> Result<Self, SketchSweepError> {
        let expected = if recipe.taper.is_some() {
            TAPER_VERSION
        } else {
            VERSION
        };
        if recipe.version != expected || recipe.profile_sketch.0 == 0 || recipe.path_sketch.0 == 0 {
            return Err(SketchSweepError::Definition(format!(
                "sweep version {}, profile {:?}, path {:?}: expected version {expected} and nonzero sketch IDs",
                recipe.version, recipe.profile_sketch, recipe.path_sketch
            )));
        }
        validate_selection(&recipe.profile)
            .map_err(|cause| SketchSweepError::Definition(format!("sweep profile: {cause}")))?;
        require_path(&recipe.path)?;
        if let Some(taper) = recipe.taper {
            taper.validate_source().map_err(taper_error)?;
            if let DatumLength::Constant(angle) = taper.source() {
                require_taper_range(angle)?;
            }
        }
        Ok(Self(recipe))
    }

    /// Borrow the versioned source recipe. Example: `serde_json::to_string(sweep.recipe())?`.
    pub fn recipe(&self) -> &SketchSweepRecipe {
        &self.0
    }

    pub(super) fn source_reads(&self) -> FeatureReadFootprint {
        FeatureReadFootprint::new(
            [self.0.profile_sketch, self.0.path_sketch]
                .into_iter()
                .map(|owner| FeatureReadKey::Sketch(owner.0))
                .chain(
                    self.0
                        .taper
                        .and_then(DatumAngle::parameter_id)
                        .map(FeatureReadKey::Parameter),
                ),
            false,
        )
    }

    /// Read both sketches and their upstream dependencies. Example: `sweep.reads(&snapshot)?`.
    pub fn reads(
        &self,
        snapshot: &DocumentSnapshot,
    ) -> Result<FeatureReadFootprint, SketchSweepError> {
        let mut keys = self.source_reads().keys().clone();
        let mut unattributed = false;
        for owner in [self.0.profile_sketch, self.0.path_sketch] {
            let sketch = snapshot
                .sketches
                .get(owner)
                .map_err(|cause| SketchSweepError::Sketch { owner, cause })?;
            keys.extend(sketch.read_footprint().keys());
            unattributed |= sketch.read_footprint().has_unattributed_reads();
        }
        Ok(FeatureReadFootprint::new(keys, unattributed))
    }

    /// Resolve source geometry against the current Part datums. Example: `sweep.resolve(&snapshot, &datums)?`.
    pub fn resolve<'a>(
        &'a self,
        snapshot: &'a DocumentSnapshot,
        datums: &WorkGeometryEvaluation,
    ) -> Result<ResolvedSketchSweep<'a>, SketchSweepError> {
        self.resolve_sources(snapshot, |owner| {
            snapshot.require_converged_resolved_sketch_with_datums(owner, datums)
        })
    }

    pub(super) fn resolve_for_replay<'a>(
        &'a self,
        snapshot: &'a DocumentSnapshot,
        datums: &WorkGeometryEvaluation,
    ) -> Result<ResolvedSketchSweep<'a>, SketchSweepError> {
        self.resolve_sources(snapshot, |owner| {
            snapshot.require_converged_sketch_with_datums(owner, datums)
        })
    }

    fn resolve_sources<'a>(
        &'a self,
        snapshot: &DocumentSnapshot,
        resolve: impl Fn(
            DocumentSketchId,
        ) -> Result<ResolvedDocumentSketch<'a>, DocumentSketchPlacementError>,
    ) -> Result<ResolvedSketchSweep<'a>, SketchSweepError> {
        let source = |owner| {
            resolve(owner).map_err(|cause| SketchSweepError::Placement {
                owner,
                cause: Box::new(cause),
            })
        };
        let profile = source(self.0.profile_sketch)?;
        let path = source(self.0.path_sketch)?;
        validate_curves(&profile, profile_curves(&self.0.profile))?;
        validate_curves(&path, self.0.path.iter())?;
        let taper = self.0.taper.map_or(Ok(0.), |angle| {
            angle.resolve(&snapshot.parameters).map_err(taper_error)
        })?;
        require_taper_range(taper)?;
        Ok(ResolvedSketchSweep {
            profile,
            boundary: &self.0.profile,
            path,
            traversal: &self.0.path,
            taper,
        })
    }
}

fn require_taper_range(radians: f64) -> Result<(), SketchSweepError> {
    if radians.abs() >= std::f64::consts::FRAC_PI_2 {
        return Err(taper_error(format!(
            "{} degrees: expected an angle strictly between -90 and 90 degrees",
            radians.to_degrees()
        )));
    }
    Ok(())
}

fn taper_error(cause: impl fmt::Display) -> SketchSweepError {
    SketchSweepError::Definition(format!("sweep taper: {cause}"))
}

fn require_path(path: &[SketchDirectedCurve]) -> Result<(), SketchSweepError> {
    let mut seen = BTreeSet::new();
    if path.is_empty()
        || path
            .iter()
            .any(|member| member.curve.0 == 0 || !seen.insert(member.curve))
    {
        return Err(SketchSweepError::Definition(format!(
            "sweep path {path:?}: expected a nonempty traversal with unique nonzero curve IDs"
        )));
    }
    Ok(())
}

fn profile_curves(profile: &SketchPrismSelection) -> impl Iterator<Item = &SketchDirectedCurve> {
    profile.outer.iter().chain(profile.holes.iter().flatten())
}

fn validate_curves<'a>(
    source: &ResolvedDocumentSketch<'_>,
    curves: impl Iterator<Item = &'a SketchDirectedCurve>,
) -> Result<(), SketchSweepError> {
    let owner = source.sketch.id();
    let index = source.sketch.geometry().read_index();
    for curve in curves {
        require_curve(owner, &index, curve.curve)?;
    }
    Ok(())
}

fn require_curve(
    owner: DocumentSketchId,
    index: &crate::sketch::SketchReadIndex<'_>,
    curve: SketchEntityId,
) -> Result<(), SketchSweepError> {
    let geometry = index
        .entity(curve)
        .map_err(|cause| SketchSweepError::Sketch { owner, cause })?;
    let construction = index
        .is_construction(curve)
        .map_err(|cause| SketchSweepError::Sketch { owner, cause })?;
    if construction
        || matches!(
            geometry,
            OwnedSketchGeometry::Point(_) | OwnedSketchGeometry::Text(_)
        )
    {
        return Err(SketchSweepError::Definition(format!(
            "sweep sketch {owner:?} entity {curve:?}: expected a non-construction curve"
        )));
    }
    Ok(())
}
