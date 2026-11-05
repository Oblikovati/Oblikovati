// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchError, require_position};
pub(crate) fn require_chain_vertices(points: &[[f64; 2]], closed: bool) -> Result<(), SketchError> {
    let minimum = if closed { 3 } else { 2 };
    if points.len() < minimum {
        return Err(SketchError::Definition(format!(
            "polyline vertices {points:?}, closed {closed}: expected at least {minimum} finite positions"
        )));
    }
    for point in points {
        require_position(*point)?;
    }
    Ok(())
}
pub(crate) fn line_chain_indices(
    count: usize,
    closed: bool,
) -> impl ExactSizeIterator<Item = [usize; 2]> {
    let segments = if closed {
        count
    } else {
        count.saturating_sub(1)
    };
    (0..segments).map(move |i| [i, (i + 1) % count])
}
