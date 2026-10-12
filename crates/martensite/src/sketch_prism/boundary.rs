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
) -> Result<PrismProbeLoop, String> {
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
) -> Result<DirectedEndpoints, String> {
    let line = require_boundary_line(geometry, directed.line)?;
    let [a, b] = line.endpoints;
    let [start, end] = if directed.reversed { [b, a] } else { [a, b] };
    let position = geometry
        .point(start)
        .map_err(|error| error.to_string())?
        .position;
    Ok(DirectedEndpoints {
        line: line.id,
        start,
        end,
        position,
    })
}
fn require_shared_closed_boundary(edges: &[DirectedEndpoints]) -> Result<(), String> {
    for (edge, next) in edges.iter().zip(edges.iter().cycle().skip(1)) {
        if edge.end != next.start {
            return Err(format!(
                "sketch prism lines {:?}->{:?}, endpoints {:?}->{:?}: expected shared authored endpoint identity in a closed directed boundary; distinct coincident points require kernel profile arrangement",
                edge.line, next.line, edge.end, next.start
            ));
        }
    }
    Ok(())
}

fn require_boundary_line(
    geometry: &Sketch2D,
    id: SketchEntityId,
) -> Result<oblikovati_model::sketch::SketchLine2D, String> {
    let line = geometry.line(id).map_err(|error| error.to_string())?;
    if geometry
        .is_construction(line.id)
        .map_err(|error| error.to_string())?
    {
        return Err(format!(
            "sketch prism line {:?}: expected a non-construction boundary curve",
            line.id
        ));
    }
    Ok(*line)
}
