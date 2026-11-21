// SPDX-License-Identifier: GPL-2.0-only
//! Disposable geometry for the curvature inspection overlay.
use super::{SketchDisplayOverlay, SketchStructure, display_decline, sketch_structure};
use crate::{ViewportSketchCurvature, ViewportSketchDisplay};
use martensite_tkservice::aspect::AspectTypeOfLine;
use martensite_tkv3d::v3d::V3dViewer;
use oblikovati_martensite::planar_curve::{
    PlanarCurvatureStation, sample_planar_curvature, sketch::resolve_sketch_curve,
};
use oblikovati_model::theme::Rgba;

impl SketchDisplayOverlay {
    pub(super) fn append_curvature(&mut self, viewer: &V3dViewer, sketch: &ViewportSketchDisplay) {
        for request in &sketch.curvature {
            match build_comb(viewer, sketch, *request, &self.curvature_color) {
                Ok(Some(structure)) => self.structures.push(structure),
                Ok(None) => (),
                Err(cause) => self
                    .declines
                    .push(display_decline(sketch, request.entity, cause)),
            }
        }
    }
}

fn build_comb(
    viewer: &V3dViewer,
    sketch: &ViewportSketchDisplay,
    request: ViewportSketchCurvature,
    color: &Rgba,
) -> Result<Option<SketchStructure>, String> {
    let definition = resolve_sketch_curve(&sketch.geometry, request.entity)
        .map_err(|error| error.to_string())?;
    let stations = sample_planar_curvature(&definition, request.intervals)
        .map_err(|error| error.to_string())?;
    let paths = comb_paths(&stations, request.envelope_ratio)?;
    if paths.is_empty() {
        return Ok(None);
    }
    sketch_structure::curve_paths(
        viewer,
        sketch.plane,
        &paths,
        color,
        AspectTypeOfLine::Solid,
        true,
    )
    .map(Some)
}

fn comb_paths(
    stations: &[PlanarCurvatureStation],
    envelope_ratio: f64,
) -> Result<Vec<Vec<[f64; 2]>>, String> {
    if !(0. ..=1.).contains(&envelope_ratio) {
        return Err(format!(
            "curvature envelope ratio {envelope_ratio}: expected 0..=1"
        ));
    }
    let peak = stations
        .iter()
        .map(|point| point.curvature[0].hypot(point.curvature[1]))
        .fold(0., f64::max);
    if peak == 0. || envelope_ratio == 0. {
        return Ok(Vec::new());
    }
    let reach = curve_span(stations) * envelope_ratio;
    let mut envelope = Vec::with_capacity(stations.len());
    let mut paths = Vec::with_capacity(stations.len() + 1);
    for station in stations {
        let tip = std::array::from_fn(|axis| {
            station.position[axis] - (station.curvature[axis] / peak) * reach
        });
        envelope.push(tip);
        paths.push(vec![station.position, tip]);
    }
    paths.push(envelope);
    Ok(paths)
}

fn curve_span(stations: &[PlanarCurvatureStation]) -> f64 {
    let mut minimum = [f64::INFINITY; 2];
    let mut maximum = [f64::NEG_INFINITY; 2];
    for point in stations {
        for (axis, coordinate) in point.position.into_iter().enumerate() {
            minimum[axis] = minimum[axis].min(coordinate);
            maximum[axis] = maximum[axis].max(coordinate);
        }
    }
    (maximum[0] - minimum[0]).max(maximum[1] - minimum[1])
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn display_scale_changes_outward_teeth_and_preserves_their_curvature_ratio() {
        let stations = [
            PlanarCurvatureStation {
                position: [2., 0.],
                curvature: [-2., 0.],
            },
            PlanarCurvatureStation {
                position: [0., 1.],
                curvature: [0., -0.25],
            },
        ];
        assert_eq!(
            comb_paths(&stations, 0.25).unwrap(),
            [
                vec![[2., 0.], [2.5, 0.]],
                vec![[0., 1.], [0., 1.0625]],
                vec![[2.5, 0.], [0., 1.0625]],
            ]
        );
        assert_eq!(comb_paths(&stations, 0.5).unwrap()[0], [[2., 0.], [3., 0.]]);
        assert!(comb_paths(&stations, 0.).unwrap().is_empty());
        assert!(comb_paths(&stations, 1.5).is_err());
    }
}
