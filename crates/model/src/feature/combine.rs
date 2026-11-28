// SPDX-License-Identifier: GPL-2.0-only
//! Authored Part Modify Combine feature recipe, validation, and resolution (#3918).
use super::{
    FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity,
    sketch_prism::PrismBoolean,
};
use std::{collections::BTreeSet, fmt};

/// Boolean operation selected for a Part Modify Combine feature.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum CombineOperation {
    #[default]
    Join,
    Cut,
    Intersect,
}

impl CombineOperation {
    /// Map to the shared Martensite solid Boolean kind; example: `CombineOperation::Cut.prism_boolean()`.
    pub fn prism_boolean(self) -> PrismBoolean {
        match self {
            Self::Join => PrismBoolean::Join,
            Self::Cut => PrismBoolean::Cut,
            Self::Intersect => PrismBoolean::Intersect,
        }
    }
}

impl fmt::Display for CombineOperation {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Join => f.write_str("Join"),
            Self::Cut => f.write_str("Cut"),
            Self::Intersect => f.write_str("Intersect"),
        }
    }
}

/// Persisted Part Modify Combine recipe (Part program version 59, source version 49).
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct CombineRecipe {
    pub version: u32,
    pub base_body: Vec<u8>,
    pub tool_bodies: Vec<Vec<u8>>,
    #[serde(default)]
    pub operation: CombineOperation,
    #[serde(default)]
    pub keep_toolbodies: bool,
}

/// Validated authored Combine definition.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct CombineDefinition {
    recipe: CombineRecipe,
    base_source: FeatureId,
    tool_sources: Vec<FeatureId>,
}

/// Resolved Combine inputs ready for Martensite solid Boolean evaluation.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ResolvedCombine {
    pub base_body: Vec<u8>,
    pub tool_bodies: Vec<Vec<u8>>,
    pub operation: CombineOperation,
    pub keep_toolbodies: bool,
}

impl CombineDefinition {
    /// Validate and construct a Combine definition from its serialized recipe.
    /// Example: `CombineDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: CombineRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "combine recipe version {}: expected 1",
                recipe.version
            ));
        }
        let (document, base_source) = reference_identity(&recipe.base_body, 4, "combine base")?;
        if recipe.tool_bodies.is_empty() {
            return Err("combine tool bodies []: expected at least one solid tool body".into());
        }
        let mut seen = BTreeSet::new();
        seen.insert(recipe.base_body.as_slice());
        let mut tool_sources = Vec::with_capacity(recipe.tool_bodies.len());
        for tool in &recipe.tool_bodies {
            let (tool_document, tool_source) = reference_identity(tool, 4, "combine tool")?;
            if tool_document != document {
                return Err(format!(
                    "combine tool document {tool_document:?}: expected base document {document:?}"
                ));
            }
            if !seen.insert(tool.as_slice()) {
                return Err(format!(
                    "combine tool {tool:?}: expected distinct tool bodies separate from base"
                ));
            }
            tool_sources.push(tool_source);
        }
        Ok(Self {
            recipe,
            base_source,
            tool_sources,
        })
    }

    /// Copy the serializable recipe; example: `definition.recipe()`.
    pub fn recipe(&self) -> CombineRecipe {
        self.recipe.clone()
    }

    /// Borrow the authored base body key; example: `definition.base_body()`.
    pub fn base_body(&self) -> &[u8] {
        &self.recipe.base_body
    }

    /// Borrow the authored tool body keys; example: `definition.tool_bodies()`.
    pub fn tool_bodies(&self) -> &[Vec<u8>] {
        &self.recipe.tool_bodies
    }

    /// Producer feature of the base body; example: `definition.base_source()`.
    pub fn base_source(&self) -> FeatureId {
        self.base_source
    }

    /// Producer features of each tool body in authored order; example: `definition.tool_sources()`.
    pub fn tool_sources(&self) -> &[FeatureId] {
        &self.tool_sources
    }

    /// Selected Boolean operation; example: `definition.operation()`.
    pub fn operation(&self) -> CombineOperation {
        self.recipe.operation
    }

    /// Whether tool bodies remain in the part after combination; example: `definition.keep_toolbodies()`.
    pub fn keep_toolbodies(&self) -> bool {
        self.recipe.keep_toolbodies
    }

    /// Attribute all read dependencies; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let mut keys = BTreeSet::new();
        keys.insert(FeatureReadKey::Feature(self.base_source));
        for source in &self.tool_sources {
            keys.insert(FeatureReadKey::Feature(*source));
        }
        FeatureReadFootprint::new(keys, false)
    }

    /// Resolve authored Combine fields for kernel evaluation; example: `definition.resolve()`.
    pub fn resolve(&self) -> ResolvedCombine {
        ResolvedCombine {
            base_body: self.recipe.base_body.clone(),
            tool_bodies: self.recipe.tool_bodies.clone(),
            operation: self.recipe.operation,
            keep_toolbodies: self.recipe.keep_toolbodies,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::identity::{DocumentGuid, SemanticReferenceName};

    fn body_key(doc: DocumentGuid, feature: u64) -> Vec<u8> {
        SemanticReferenceName::new(doc, feature, 4, "prism", "body", &[feature])
            .unwrap()
            .encode()
            .unwrap()
    }

    #[test]
    fn combine_definition_validates_distinct_bodies_and_records_reads() {
        let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let base = body_key(doc, 1);
        let tool_a = body_key(doc, 2);
        let tool_b = body_key(doc, 3);
        let recipe = CombineRecipe {
            version: 1,
            base_body: base.clone(),
            tool_bodies: vec![tool_a.clone(), tool_b.clone()],
            operation: CombineOperation::Cut,
            keep_toolbodies: false,
        };
        let def = CombineDefinition::from_recipe(recipe).unwrap();
        assert_eq!(def.base_source(), FeatureId(1));
        assert_eq!(def.tool_sources(), &[FeatureId(2), FeatureId(3)]);
        assert_eq!(def.operation(), CombineOperation::Cut);
        assert!(!def.keep_toolbodies());
        let reads = def.reads();
        assert!(
            reads
                .keys()
                .contains(&FeatureReadKey::Feature(FeatureId(1)))
        );
        assert!(
            reads
                .keys()
                .contains(&FeatureReadKey::Feature(FeatureId(2)))
        );
        assert!(
            reads
                .keys()
                .contains(&FeatureReadKey::Feature(FeatureId(3)))
        );

        // Reject empty tools
        let empty = CombineRecipe {
            version: 1,
            base_body: base.clone(),
            tool_bodies: vec![],
            operation: CombineOperation::Join,
            keep_toolbodies: true,
        };
        assert!(CombineDefinition::from_recipe(empty).is_err());

        // Reject base reused as tool
        let self_tool = CombineRecipe {
            version: 1,
            base_body: base.clone(),
            tool_bodies: vec![base.clone()],
            operation: CombineOperation::Join,
            keep_toolbodies: false,
        };
        assert!(CombineDefinition::from_recipe(self_tool).is_err());

        // Reject duplicate tools
        let dup_tool = CombineRecipe {
            version: 1,
            base_body: base,
            tool_bodies: vec![tool_a.clone(), tool_a],
            operation: CombineOperation::Intersect,
            keep_toolbodies: false,
        };
        assert!(CombineDefinition::from_recipe(dup_tool).is_err());
    }
}
