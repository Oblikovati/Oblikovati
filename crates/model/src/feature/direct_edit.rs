// SPDX-License-Identifier: GPL-2.0-only
//! Direct Edit feature model, ordered sub-operations, and versioned recipes (#3918).
use super::{FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity};

/// Active manipulation mode in the floating Direct Edit toolbar.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum DirectEditMode {
    #[default]
    Move,
    Size,
    Scale,
    Rotate,
    Delete,
}

impl DirectEditMode {
    /// Human-readable toolbar button caption and operation-name prefix.
    /// Example: `DirectEditMode::Scale.label() == "Scale"`.
    pub fn label(self) -> &'static str {
        match self {
            Self::Move => "Move",
            Self::Size => "Size",
            Self::Scale => "Scale",
            Self::Rotate => "Rotate",
            Self::Delete => "Delete",
        }
    }
}

/// Entity selection filter offered by the Direct Edit toolbar.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum DirectEditSelectionFilter {
    #[default]
    Faces,
    Solids,
}

impl DirectEditSelectionFilter {
    /// Display caption for the selection filter menu.
    /// Example: `DirectEditSelectionFilter::Solids.label() == "Solids"`.
    pub fn label(self) -> &'static str {
        match self {
            Self::Faces => "Faces",
            Self::Solids => "Solids",
        }
    }
}

/// Coordinate frame choice for Direct Edit Rotate operations.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum DirectEditCoordinateMode {
    #[default]
    Local,
    World,
}

impl DirectEditCoordinateMode {
    /// Display caption for the Rotate coordinate menu.
    /// Example: `DirectEditCoordinateMode::World.label() == "World"`.
    pub fn label(self) -> &'static str {
        match self {
            Self::Local => "Local",
            Self::World => "World",
        }
    }
}

/// Scaling mode for Direct Edit Scale operations.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum DirectEditScaleMode {
    #[default]
    Uniform,
    NonUniform,
}

impl DirectEditScaleMode {
    /// Display caption for the Scale mode selector.
    /// Example: `DirectEditScaleMode::NonUniform.label() == "Non Uniform"`.
    pub fn label(self) -> &'static str {
        match self {
            Self::Uniform => "Uniform",
            Self::NonUniform => "Non Uniform",
        }
    }
}

/// One typed Direct Edit sub-operation payload.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub enum DirectEditOperationKind {
    Move {
        filter: DirectEditSelectionFilter,
        faces: Vec<Vec<u8>>,
        translation: [f64; 3],
    },
    Size {
        faces: Vec<Vec<u8>>,
        offset: f64,
    },
    Scale {
        mode: DirectEditScaleMode,
        uniform: f64,
        factors: [f64; 3],
    },
    Rotate {
        filter: DirectEditSelectionFilter,
        faces: Vec<Vec<u8>>,
        coordinate: DirectEditCoordinateMode,
        origin: [f64; 3],
        axis: [f64; 3],
        angle_deg: f64,
    },
    Delete {
        faces: Vec<Vec<u8>>,
    },
}

impl DirectEditOperationKind {
    /// Return the toolbar mode corresponding to this sub-operation.
    /// Example: `op.mode() == DirectEditMode::Scale`.
    pub fn mode(&self) -> DirectEditMode {
        match self {
            Self::Move { .. } => DirectEditMode::Move,
            Self::Size { .. } => DirectEditMode::Size,
            Self::Scale { .. } => DirectEditMode::Scale,
            Self::Rotate { .. } => DirectEditMode::Rotate,
            Self::Delete { .. } => DirectEditMode::Delete,
        }
    }
}

/// One named sub-operation stored inside a `DirectEditDefinition` (e.g. `Scale1`, `Move1`).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct DirectEditOperation {
    pub name: String,
    pub kind: DirectEditOperationKind,
}

/// Versioned Direct Edit recipe stored without evaluated geometry caches.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct DirectEditRecipe {
    pub version: u32,
    pub body: Vec<u8>,
    pub operations: Vec<DirectEditOperation>,
}

/// Validated Direct Edit feature retaining target body lineage and ordered sub-operations.
#[derive(Clone, Debug, PartialEq)]
pub struct DirectEditDefinition {
    body: Vec<u8>,
    body_source: FeatureId,
    operations: Vec<DirectEditOperation>,
}

/// Resolved Direct Edit operation sequence passed to the kernel evaluator.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedDirectEdit {
    pub operations: Vec<DirectEditOperationKind>,
}

impl DirectEditDefinition {
    /// Construct and validate a Direct Edit definition; example: `DirectEditDefinition::new(body, ops)?`.
    pub fn new(body: Vec<u8>, operations: Vec<DirectEditOperation>) -> Result<Self, String> {
        let (_, body_source) = reference_identity(&body, 4, "direct edit body")?;
        if operations.is_empty() {
            return Err("direct edit operations []: expected at least one sub-operation".into());
        }
        for op in &operations {
            validate_operation(op)?;
        }
        Ok(Self {
            body,
            body_source,
            operations,
        })
    }

    /// Restore a validated recipe; example: `DirectEditDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: DirectEditRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "direct edit recipe version {}: expected version 1",
                recipe.version
            ));
        }
        Self::new(recipe.body, recipe.operations)
    }

    /// Capture authored inputs for persistence; example: `definition.recipe()`.
    pub fn recipe(&self) -> DirectEditRecipe {
        DirectEditRecipe {
            version: 1,
            body: self.body.clone(),
            operations: self.operations.clone(),
        }
    }

    /// Target solid reference key; example: `definition.body()`.
    pub fn body(&self) -> &[u8] {
        &self.body
    }

    /// Producing feature of the target solid; example: `definition.body_source()`.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }

    /// Ordered sub-operations owned by this Direct Edit feature (e.g. `Scale1`).
    /// Example: `definition.operations()`.
    pub fn operations(&self) -> &[DirectEditOperation] {
        &self.operations
    }

    /// Attribute body dependency reads; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        FeatureReadFootprint::new(
            std::iter::once(FeatureReadKey::Feature(self.body_source)),
            false,
        )
    }

    /// Resolve authored sub-operations for kernel replay; example: `definition.resolve()`.
    pub fn resolve(&self) -> ResolvedDirectEdit {
        ResolvedDirectEdit {
            operations: self.operations.iter().map(|op| op.kind.clone()).collect(),
        }
    }

    /// Exact structural comparison for history diffing; example: `before.same_recipe(&after)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self.body == other.body
            && self.operations.len() == other.operations.len()
            && self
                .operations
                .iter()
                .zip(&other.operations)
                .all(|(a, b)| a.name == b.name && same_operation_kind(&a.kind, &b.kind))
    }

    /// Compute the next sequential sub-operation name for a given mode (e.g. `Scale1`, `Scale2`).
    /// Example: `DirectEditDefinition::next_operation_name(&ops, DirectEditMode::Scale)`.
    pub fn next_operation_name(existing: &[DirectEditOperation], mode: DirectEditMode) -> String {
        let prefix = mode.label();
        let mut n = 1usize;
        loop {
            let candidate = format!("{prefix}{n}");
            if !existing.iter().any(|op| op.name == candidate) {
                return candidate;
            }
            n += 1;
        }
    }
}

/// Validate a positive finite scale factor; example: `validate_scale_factor(2.0, "Uniform")?`.
pub fn validate_scale_factor(value: f64, label: &str) -> Result<(), String> {
    if !value.is_finite() || value <= 0.0 {
        return Err(format!(
            "direct edit scale {label} {value:?}: expected a finite positive factor"
        ));
    }
    Ok(())
}

fn validate_operation(op: &DirectEditOperation) -> Result<(), String> {
    if op.name.trim().is_empty() || op.name.contains('\0') {
        return Err(format!(
            "direct edit operation name {:?}: expected a non-empty name without NUL",
            op.name
        ));
    }
    match &op.kind {
        DirectEditOperationKind::Move { translation, .. } => {
            require_finite3(*translation, "move translation")
        }
        DirectEditOperationKind::Size { offset, .. } => {
            if !offset.is_finite() {
                return Err(format!(
                    "direct edit size offset {offset:?}: expected a finite offset"
                ));
            }
            Ok(())
        }
        DirectEditOperationKind::Scale {
            mode,
            uniform,
            factors,
        } => {
            validate_scale_factor(*uniform, "uniform")?;
            if *mode == DirectEditScaleMode::NonUniform {
                validate_scale_factor(factors[0], "X")?;
                validate_scale_factor(factors[1], "Y")?;
                validate_scale_factor(factors[2], "Z")?;
            }
            Ok(())
        }
        DirectEditOperationKind::Rotate {
            origin,
            axis,
            angle_deg,
            ..
        } => {
            require_finite3(*origin, "rotate origin")?;
            require_finite3(*axis, "rotate axis")?;
            if axis.iter().all(|c| *c == 0.0) {
                return Err(format!(
                    "direct edit rotate axis {axis:?}: expected a nonzero axis"
                ));
            }
            if !angle_deg.is_finite() {
                return Err(format!(
                    "direct edit rotate angle {angle_deg:?}: expected a finite angle"
                ));
            }
            Ok(())
        }
        DirectEditOperationKind::Delete { .. } => Ok(()),
    }
}

fn require_finite3(values: [f64; 3], label: &str) -> Result<(), String> {
    if values.iter().all(|value| value.is_finite()) {
        return Ok(());
    }
    Err(format!(
        "direct edit {label} {values:?}: expected three finite coordinates"
    ))
}

fn same_f64(a: f64, b: f64) -> bool {
    a.to_bits() == b.to_bits()
}

fn same_f64_3(a: [f64; 3], b: [f64; 3]) -> bool {
    same_f64(a[0], b[0]) && same_f64(a[1], b[1]) && same_f64(a[2], b[2])
}

fn same_operation_kind(a: &DirectEditOperationKind, b: &DirectEditOperationKind) -> bool {
    match (a, b) {
        (
            DirectEditOperationKind::Move {
                filter: f1,
                faces: fa1,
                translation: t1,
            },
            DirectEditOperationKind::Move {
                filter: f2,
                faces: fa2,
                translation: t2,
            },
        ) => f1 == f2 && fa1 == fa2 && same_f64_3(*t1, *t2),
        (
            DirectEditOperationKind::Size {
                faces: fa1,
                offset: o1,
            },
            DirectEditOperationKind::Size {
                faces: fa2,
                offset: o2,
            },
        ) => fa1 == fa2 && same_f64(*o1, *o2),
        (
            DirectEditOperationKind::Scale {
                mode: m1,
                uniform: u1,
                factors: f1,
            },
            DirectEditOperationKind::Scale {
                mode: m2,
                uniform: u2,
                factors: f2,
            },
        ) => m1 == m2 && same_f64(*u1, *u2) && same_f64_3(*f1, *f2),
        (
            DirectEditOperationKind::Rotate {
                filter: f1,
                faces: fa1,
                coordinate: c1,
                origin: o1,
                axis: a1,
                angle_deg: d1,
            },
            DirectEditOperationKind::Rotate {
                filter: f2,
                faces: fa2,
                coordinate: c2,
                origin: o2,
                axis: a2,
                angle_deg: d2,
            },
        ) => {
            f1 == f2
                && fa1 == fa2
                && c1 == c2
                && same_f64_3(*o1, *o2)
                && same_f64_3(*a1, *a2)
                && same_f64(*d1, *d2)
        }
        (
            DirectEditOperationKind::Delete { faces: fa1 },
            DirectEditOperationKind::Delete { faces: fa2 },
        ) => fa1 == fa2,
        _ => false,
    }
}
