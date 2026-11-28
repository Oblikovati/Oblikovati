// SPDX-License-Identifier: GPL-2.0-only
//! Sketch loop chaining and interior-diagonal detection for Surface Boundary Patch (#3918).
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};
use std::collections::{BTreeMap, BTreeSet};

const POINT_TOL: f64 = 1e-5;

#[derive(Clone, Copy)]
struct SketchSegment2D {
    id: SketchEntityId,
    start: [f64; 2],
    end: [f64; 2],
}

/// Detect whether `candidate` is an interior diagonal/chord strictly crossing inside
/// a closed boundary loop formed by other non-construction curves in `geometry`.
/// Example: `is_interior_diagonal_curve(sketch.geometry(), line_id)`.
pub fn is_interior_diagonal_curve(geometry: &Sketch2D, candidate: SketchEntityId) -> bool {
    let Some(line) = geometry.lines().iter().find(|l| l.id == candidate) else {
        return false;
    };
    if matches!(geometry.is_construction(candidate), Ok(true)) {
        return false;
    }
    let (Ok(p0), Ok(p1)) = (
        geometry.point(line.endpoints[0]),
        geometry.point(line.endpoints[1]),
    ) else {
        return false;
    };
    let mid = [
        0.5 * (p0.position[0] + p1.position[0]),
        0.5 * (p0.position[1] + p1.position[1]),
    ];
    let others: Vec<SketchSegment2D> = non_construction_segments(geometry)
        .into_iter()
        .filter(|seg| seg.id != candidate)
        .collect();
    if others.len() < 3 {
        return false;
    }
    collect_closed_loops(&others)
        .iter()
        .any(|poly| point_strictly_inside_polygon(mid, poly))
}

/// Walk connected non-construction boundary curves starting at `seed` when
/// Automatic Edge Chain is enabled, skipping interior diagonal chords.
/// Example: `chain_sketch_boundary_loop(sketch.geometry(), clicked_curve)`.
pub fn chain_sketch_boundary_loop(
    geometry: &Sketch2D,
    seed: SketchEntityId,
) -> Vec<SketchEntityId> {
    if matches!(geometry.is_construction(seed), Ok(true)) {
        return Vec::new();
    }
    if geometry.circles().iter().any(|c| c.id == seed)
        || geometry.ellipses().iter().any(|e| e.id() == seed)
    {
        return vec![seed];
    }
    if is_interior_diagonal_curve(geometry, seed) {
        return Vec::new();
    }
    let segments: Vec<SketchSegment2D> = non_construction_segments(geometry)
        .into_iter()
        .filter(|seg| !is_interior_diagonal_curve(geometry, seg.id))
        .collect();
    let Some(seed_idx) = segments.iter().position(|seg| seg.id == seed) else {
        return Vec::new();
    };
    if let Some(loop_ids) = find_closed_loop_from(&segments, seed_idx) {
        return loop_ids;
    }
    collect_open_chain(&segments, seed_idx)
}

fn non_construction_segments(geometry: &Sketch2D) -> Vec<SketchSegment2D> {
    let mut segments = Vec::new();
    for line in geometry.lines() {
        if matches!(geometry.is_construction(line.id), Ok(true)) {
            continue;
        }
        if let (Ok(p0), Ok(p1)) = (
            geometry.point(line.endpoints[0]),
            geometry.point(line.endpoints[1]),
        ) {
            segments.push(SketchSegment2D {
                id: line.id,
                start: p0.position,
                end: p1.position,
            });
        }
    }
    for arc in geometry.arcs() {
        if matches!(geometry.is_construction(arc.id), Ok(true)) {
            continue;
        }
        if let (Ok(p0), Ok(p1)) = (geometry.point(arc.start), geometry.point(arc.end)) {
            segments.push(SketchSegment2D {
                id: arc.id,
                start: p0.position,
                end: p1.position,
            });
        }
    }
    segments
}

fn find_closed_loop_from(
    segments: &[SketchSegment2D],
    seed_idx: usize,
) -> Option<Vec<SketchEntityId>> {
    let seed = segments[seed_idx];
    let mut visited = vec![false; segments.len()];
    visited[seed_idx] = true;
    let mut path = vec![seed.id];
    dfs_loop(segments, seed.start, seed.end, &mut visited, &mut path).then_some(path)
}

fn dfs_loop(
    segments: &[SketchSegment2D],
    target: [f64; 2],
    current: [f64; 2],
    visited: &mut [bool],
    path: &mut Vec<SketchEntityId>,
) -> bool {
    if path.len() >= 3 && dist2(current, target) <= POINT_TOL {
        return true;
    }
    for (idx, seg) in segments.iter().enumerate() {
        if visited[idx] {
            continue;
        }
        let next = if dist2(current, seg.start) <= POINT_TOL {
            seg.end
        } else if dist2(current, seg.end) <= POINT_TOL {
            seg.start
        } else {
            continue;
        };
        visited[idx] = true;
        path.push(seg.id);
        if dfs_loop(segments, target, next, visited, path) {
            return true;
        }
        path.pop();
        visited[idx] = false;
    }
    false
}

fn collect_open_chain(segments: &[SketchSegment2D], seed_idx: usize) -> Vec<SketchEntityId> {
    let mut used = BTreeSet::new();
    let mut chain = vec![segments[seed_idx]];
    used.insert(seed_idx);
    loop {
        let mut added = false;
        let head = chain[0].start;
        let tail = chain[chain.len() - 1].end;
        for (idx, &seg) in segments.iter().enumerate() {
            if used.contains(&idx) {
                continue;
            }
            if dist2(tail, seg.start) <= POINT_TOL {
                chain.push(seg);
            } else if dist2(tail, seg.end) <= POINT_TOL {
                chain.push(SketchSegment2D {
                    id: seg.id,
                    start: seg.end,
                    end: seg.start,
                });
            } else if dist2(head, seg.end) <= POINT_TOL {
                chain.insert(0, seg);
            } else if dist2(head, seg.start) <= POINT_TOL {
                chain.insert(
                    0,
                    SketchSegment2D {
                        id: seg.id,
                        start: seg.end,
                        end: seg.start,
                    },
                );
            } else {
                continue;
            }
            used.insert(idx);
            added = true;
            break;
        }
        if !added {
            break;
        }
    }
    chain.into_iter().map(|s| s.id).collect()
}

fn collect_closed_loops(segments: &[SketchSegment2D]) -> Vec<Vec<[f64; 2]>> {
    let mut loops = Vec::new();
    let mut seen_sets: BTreeSet<Vec<u64>> = BTreeSet::new();
    for start_idx in 0..segments.len() {
        let seed = segments[start_idx];
        let mut visited = vec![false; segments.len()];
        visited[start_idx] = true;
        let mut pts = vec![seed.start, seed.end];
        let mut ids = vec![seed.id.0];
        let mut found = BTreeMap::new();
        dfs_all_loops(
            segments,
            seed.start,
            seed.end,
            &mut visited,
            &mut pts,
            &mut ids,
            &mut found,
        );
        for (key, poly) in found {
            if seen_sets.insert(key) {
                loops.push(poly);
            }
        }
    }
    loops
}

fn dfs_all_loops(
    segments: &[SketchSegment2D],
    target: [f64; 2],
    current: [f64; 2],
    visited: &mut [bool],
    pts: &mut Vec<[f64; 2]>,
    ids: &mut Vec<u64>,
    out: &mut BTreeMap<Vec<u64>, Vec<[f64; 2]>>,
) {
    if ids.len() >= 3 && dist2(current, target) <= POINT_TOL {
        let mut key = ids.clone();
        key.sort_unstable();
        out.entry(key)
            .or_insert_with(|| pts[..pts.len() - 1].to_vec());
        return;
    }
    if ids.len() >= 16 {
        return;
    }
    for (idx, seg) in segments.iter().enumerate() {
        if visited[idx] {
            continue;
        }
        let next = if dist2(current, seg.start) <= POINT_TOL {
            seg.end
        } else if dist2(current, seg.end) <= POINT_TOL {
            seg.start
        } else {
            continue;
        };
        visited[idx] = true;
        pts.push(next);
        ids.push(seg.id.0);
        dfs_all_loops(segments, target, next, visited, pts, ids, out);
        ids.pop();
        pts.pop();
        visited[idx] = false;
    }
}

fn point_strictly_inside_polygon(pt: [f64; 2], poly: &[[f64; 2]]) -> bool {
    if poly.len() < 3 {
        return false;
    }
    let mut inside = false;
    for i in 0..poly.len() {
        let a = poly[i];
        let b = poly[(i + 1) % poly.len()];
        if point_to_segment_dist2(pt, a, b) <= POINT_TOL {
            return false;
        }
        let crosses = (a[1] > pt[1]) != (b[1] > pt[1]);
        if crosses {
            let x_int = a[0] + (pt[1] - a[1]) * (b[0] - a[0]) / (b[1] - a[1]);
            if pt[0] < x_int {
                inside = !inside;
            }
        }
    }
    inside
}

fn point_to_segment_dist2(p: [f64; 2], a: [f64; 2], b: [f64; 2]) -> f64 {
    let dx = b[0] - a[0];
    let dy = b[1] - a[1];
    let len_sq = dx * dx + dy * dy;
    if len_sq <= 1e-18 {
        return dist2(p, a);
    }
    let t = (((p[0] - a[0]) * dx + (p[1] - a[1]) * dy) / len_sq).clamp(0.0, 1.0);
    dist2(p, [a[0] + t * dx, a[1] + t * dy])
}

fn dist2(a: [f64; 2], b: [f64; 2]) -> f64 {
    (a[0] - b[0]).hypot(a[1] - b[1])
}
