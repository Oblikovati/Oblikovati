// SPDX-License-Identifier: GPL-2.0-only
//! What Region Properties holds and how accepted loops read as a result.
use crate::HeadSketchEdit;
use oblikovati_martensite::sketch_region::SketchRegionMeasure;
use oblikovati_model::units::UnitsOfMeasure;

/// The running dialog on one sketch.
pub(crate) struct SketchRegionState {
    pub edit: HeadSketchEdit,
    pub generation: u64,
    pub loops: Vec<SketchRegionMeasure>,
    pub results: Vec<String>,
}
impl SketchRegionState {
    /// Clicking an accepted loop again releases it; any change drops an older result.
    pub(crate) fn toggle_loop(&mut self, region: SketchRegionMeasure) {
        self.results.clear();
        match self.loops.iter().position(|known| *known == region) {
            Some(index) => {
                self.loops.remove(index);
            }
            None => self.loops.push(region),
        }
    }
    pub(crate) fn loop_names(&self) -> Vec<String> {
        (1..=self.loops.len())
            .map(|number| format!("Sketch Loop{number}"))
            .collect()
    }
}
/// Totals over the accepted loops: area and length add, the centroid weighs by area.
pub(crate) fn region_result_rows(
    loops: &[SketchRegionMeasure],
    units: &UnitsOfMeasure,
) -> Vec<String> {
    // Working lengths are centimetres at scale one: fits use the same factor.
    let millimetres = 10. * units.working_scale();
    let area: f64 = loops.iter().map(|region| region.area).sum();
    let perimeter: f64 = loops.iter().map(|region| region.perimeter).sum();
    let moment = |axis: usize| {
        loops
            .iter()
            .map(|region| region.area * region.centroid[axis])
            .sum::<f64>()
    };
    vec![
        format!("Area {:.3} mm^2", area * millimetres * millimetres),
        format!("Perimeter {:.3} mm", perimeter * millimetres),
        format!(
            "Centroid X {:.3} Y {:.3} mm",
            moment(0) / area * millimetres,
            moment(1) / area * millimetres
        ),
    ]
}

#[cfg(test)]
#[path = "../../../tests/sketch/region-properties/rows.rs"]
mod tests;
