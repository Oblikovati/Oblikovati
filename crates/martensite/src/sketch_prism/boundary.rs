// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::PrismProbeLoop;
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};
struct DirectedEndpoints {
    line: SketchEntityId,
    start: SketchEntityId,
    end: SketchEntityId,
    position: [f64; 2],
}
pub(super) fn selected_loop(
    geometry: &Sketch2D,
    lines: &[SketchDirectedLine],
) -> Result<PrismProbeLoop, EvaluationFailure> {
    let resolved = lines
        .iter()
        .map(|directed| resolve_endpoints(geometry, *directed))
        .collect::<Result<Vec<_>, _>>()?;
    require_shared_closed_boundary(&resolved)?;
    Ok(PrismProbeLoop {
        edge_keys: resolved.iter().map(|edge| edge.line.0).collect(),
        vertices: resolved.iter().map(|edge| edge.position).collect(),
    })
}
fn resolve_endpoints(
    geometry: &Sketch2D,
    directed: SketchDirectedLine,
) -> Result<DirectedEndpoints, EvaluationFailure> {
    let line = require_boundary_line(geometry, directed.line)?;
    let [a, b] = line.endpoints;
    let [start, end] = if directed.reversed { [b, a] } else { [a, b] };
    let position = geometry
        .point(start)
        .map_err(|error| Phase::Input.sketch(Entity::SketchEntity(start.0), error))?
        .position;
    Ok(DirectedEndpoints {
        line: line.id,
        start,
        end,
        position,
    })
}
fn require_shared_closed_boundary(edges: &[DirectedEndpoints]) -> Result<(), EvaluationFailure> {
    for (edge, next) in edges.iter().zip(edges.iter().cycle().skip(1)) {
        if edge.end != next.start {
            return Err(Phase::Input.refused(Kind::InvalidInput, Entity::Edges(vec![edge.line.0, next.line.0]), format!(
                "sketch prism lines {:?}->{:?}, endpoints {:?}->{:?}: expected shared authored endpoint identity in a closed directed boundary; distinct coincident points require kernel profile arrangement",
                edge.line, next.line, edge.end, next.start
            )));
        }
    }
    Ok(())
}

fn require_boundary_line(
    geometry: &Sketch2D,
    id: SketchEntityId,
) -> Result<oblikovati_model::sketch::SketchLine2D, EvaluationFailure> {
    let line = geometry
        .line(id)
        .map_err(|error| Phase::Input.sketch(Entity::SketchEntity(id.0), error))?;
    if geometry
        .is_construction(line.id)
        .map_err(|error| Phase::Input.sketch(Entity::SketchEntity(id.0), error))?
    {
        return Err(Phase::Input.refused(
            Kind::InvalidInput,
            Entity::SketchEntity(id.0),
            format!(
                "sketch prism line {:?}: expected a non-construction boundary curve",
                line.id
            ),
        ));
    }
    Ok(*line)
}
