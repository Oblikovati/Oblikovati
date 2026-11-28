// SPDX-License-Identifier: GPL-2.0-only
//! Solid-body Boolean combination with isolated tool copies and exact lineage (#3918).
use super::extrusion_boolean::{BooleanFailure, BooleanTool};
use super::*;
use oblikovati_model::feature::combine::CombineOperation;

impl NamedSolid {
    pub(in crate::prism::named) fn combine_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        tools: &[&NamedSolid],
        operation: CombineOperation,
    ) -> Result<BlendCandidate, BooleanFailure> {
        if tools.is_empty() {
            return Err(BooleanFailure::Refused(
                "combine tools []: expected at least one solid tool body".into(),
            ));
        }
        let mut boolean_tools = Vec::with_capacity(tools.len());
        for tool in tools {
            boolean_tools.push(tool.combine_boolean_tool(history, document)?);
        }
        self.boolean_tools(
            history,
            document,
            boolean_tools,
            operation.prism_boolean(),
            &format!("combine {operation}"),
        )
    }

    fn combine_boolean_tool(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<BooleanTool, String> {
        let raw = match self {
            Self::Prism(body) => body.combine_boolean_tool(history, document)?,
            Self::Derived(body) => body.sweep_boolean_tool(history, document)?,
        };
        let mut copier = crate::checked_shape::copy_shape(&raw.shape, "combine tool")
            .map_err(|cause| cause.to_string())?;
        let sources = copy_named_sources(&mut copier, raw.sources)?;
        let shape = copier.shape().map_err(|cause| cause.to_string())?.clone();
        Ok(BooleanTool {
            shape,
            sources,
            resolution: raw.resolution,
        })
    }
}

impl NamedPrismSnapshot {
    fn combine_boolean_tool(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<BooleanTool, String> {
        let resolution = self
            .certification
            .as_ref()
            .ok_or("combine tool: expected a certified tool prism")?
            .resolution;
        Ok(BooleanTool {
            shape: self.snapshot.shape.clone(),
            sources: self.blend_sources(history, document)?,
            resolution: Some(resolution),
        })
    }
}
