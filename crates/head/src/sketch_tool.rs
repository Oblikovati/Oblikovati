// SPDX-License-Identifier: GPL-2.0-only
use crate::placement_input::SketchPlacementState;
use crate::sketch_geometric::SketchGeometricState;
use oblikovati_model::document_sketch::DocumentSketchId;

pub(crate) enum SketchToolState {
    Block(Box<crate::owner::block_placement::SketchBlockPlacement>),
    Placement(Box<SketchPlacementState>),
    Geometric(Box<SketchGeometricState>),
    Dimension(Box<crate::sketch_dimension::SketchDimensionState>),
    Measure(Box<crate::sketch_inspect::measure::SketchMeasureState>),
    Region(Box<crate::sketch_inspect::region_properties::SketchRegionState>),
}
impl SketchToolState {
    pub(crate) fn context(&self) -> (u64, DocumentSketchId, u64) {
        match self {
            Self::Block(tool) => (tool.edit.document, tool.edit.sketch, tool.generation),
            Self::Placement(tool) => (tool.document, tool.sketch, tool.generation),
            Self::Geometric(tool) => (tool.edit.document, tool.edit.sketch, tool.generation),
            Self::Dimension(tool) => (tool.edit.document, tool.edit.sketch, tool.generation),
            Self::Measure(tool) => (tool.edit.document, tool.edit.sketch, tool.generation),
            Self::Region(tool) => (tool.edit.document, tool.edit.sketch, tool.generation),
        }
    }
    pub(crate) fn label(&self) -> &'static str {
        match self {
            Self::Block(_) => "Place Block",
            Self::Placement(tool) => tool.tool.name(),
            Self::Geometric(tool) => tool.choice.label(),
            Self::Dimension(_) => "Dimension",
            Self::Measure(_) => "Measure",
            Self::Region(_) => "Region Properties",
        }
    }
    pub(crate) fn placement(&self) -> Option<&SketchPlacementState> {
        match self {
            Self::Placement(tool) => Some(tool),
            _ => None,
        }
    }
}
