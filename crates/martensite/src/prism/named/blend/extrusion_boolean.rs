// SPDX-License-Identifier: GPL-2.0-only
pub(crate) use super::solid_boolean::{BooleanFailure, BooleanTool};
use super::*;
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPITransform;
use oblikovati_model::feature::sketch_prism::PrismBoolean;

impl NamedSolid {
    pub(in crate::prism::named) fn extrusion_boolean_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        tool: &NamedPrismSnapshot,
        kind: PrismBoolean,
    ) -> Result<BlendCandidate, String> {
        self.extrusion_boolean_tools(
            history,
            document,
            vec![tool.boolean_tool(history, document)?],
            kind,
        )
    }

    pub(in crate::prism::named) fn extrusion_boolean_tools(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        tools: Vec<BooleanTool>,
        kind: PrismBoolean,
    ) -> Result<BlendCandidate, String> {
        self.boolean_tools(history, document, tools, kind, &format!("extrude {kind}"))
            .map_err(|failure| failure.to_string())
    }

    pub(crate) fn boolean_tools(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        tools: Vec<BooleanTool>,
        kind: PrismBoolean,
        operation: &str,
    ) -> Result<BlendCandidate, BooleanFailure> {
        match self {
            Self::Prism(body) => body.boolean_tools(history, document, tools, kind, operation),
            Self::Derived(body) => body.boolean_tools(history, document, tools, kind, operation),
        }
    }
}

impl NamedPrismSnapshot {
    fn boolean_tools(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        tools: Vec<BooleanTool>,
        kind: PrismBoolean,
        operation: &str,
    ) -> Result<BlendCandidate, BooleanFailure> {
        let body = self.body_reference_key(history, document)?.to_vec();
        super::solid_boolean::build_boolean_candidate(
            BlendInputTopology {
                body: &self.snapshot.shape,
                source: self.blend_sources(history, document)?,
                source_bodies: vec![body.clone()],
                body_parent: body.clone(),
                primary_body: body,
                inherited_deleted: Vec::new(),
                inherited_splits: Default::default(),
                resolution: self.certification.as_ref().map(|value| value.resolution),
            },
            history,
            document,
            tools,
            kind,
            operation,
        )
    }
    fn boolean_tool(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<BooleanTool, String> {
        let resolution = self
            .certification
            .as_ref()
            .ok_or("extrude Boolean: expected a certified tool prism")?
            .resolution;
        Ok(BooleanTool {
            shape: self.snapshot.shape.clone(),
            sources: self.blend_sources(history, document)?,
            resolution: Some(resolution),
        })
    }

    pub(in crate::prism::named) fn transformed_boolean_tool(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        motion: crate::RigidMotion,
        marker: &[u8],
    ) -> Result<BooleanTool, String> {
        let mut tool = self.boolean_tool(history, document)?;
        distinguish_pattern_tool(&mut tool.sources, document, marker)?;
        let mut moved = BRepBuilderAPITransform::with_shape(
            &tool.shape,
            super::rigid_motion::compose(&[motion])?,
            false,
            false,
        )
        .map_err(|error| error.to_string())?;
        tool.shape = moved.shape().map_err(|error| error.to_string())?.clone();
        for (_, shape) in &mut tool.sources {
            *shape = moved
                .modified_shape(shape)
                .map_err(|error| error.to_string())?;
        }
        Ok(tool)
    }
}

fn distinguish_pattern_tool(
    sources: &mut [(Vec<u8>, Shape)],
    document: DocumentGuid,
    marker: &[u8],
) -> Result<(), String> {
    use oblikovati_model::identity::{DerivedReferenceName, ReferenceEnvelope};
    for (name, _) in sources {
        let envelope = ReferenceEnvelope::decode(name).map_err(|error| error.to_string())?;
        *name = DerivedReferenceName::new(
            document,
            envelope.context,
            envelope.kind,
            "pattern-tool",
            &[name.clone(), marker.to_vec()],
        )?
        .encode()?;
    }
    Ok(())
}
