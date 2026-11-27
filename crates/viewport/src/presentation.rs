// SPDX-License-Identifier: GPL-2.0-only
use crate::{ViewportScene, ViewportTriangle};
use oblikovati_martensite::PrismPresentation;
impl ViewportScene {
    /// Convert colored derived triangles and reject coordinates or colors that cannot be displayed.
    /// Example: `ViewportScene::from_colored_triangles([0.; 3], &triangles)?`.
    pub fn from_colored_triangles(
        background: [f64; 3],
        triangles: &[([[f64; 3]; 3], [f32; 4])],
    ) -> Result<Self, String> {
        Self::from_indexed_triangles(
            background,
            triangles
                .iter()
                .map(|&(points, color)| (points, color, None)),
        )
    }
    /// Convert triangles with optional zero-based face ordinals in ascending face order.
    /// Example: `ViewportScene::from_indexed_triangles(background, [(points, color, Some(0))])?`.
    pub fn from_indexed_triangles(
        background: [f64; 3],
        triangles: impl IntoIterator<Item = ([[f64; 3]; 3], [f32; 4], Option<usize>)>,
    ) -> Result<Self, String> {
        let triangles = triangles
            .into_iter()
            .map(|(points, color, face_index)| display_triangle(points, color, face_index))
            .collect::<Result<Vec<_>, _>>()?;
        let scene = Self {
            background,
            triangles,
        };
        scene.validate()?;
        Ok(scene)
    }
    /// Convert owned kernel mesh coordinates to display precision; construction roles are not pick keys.
    /// Example: `ViewportScene::from_prism(&mesh, [0., 0., 1.], [0., 1., 0., 1.])?`.
    pub fn from_prism(
        presentation: &PrismPresentation,
        background: [f64; 3],
        color: [f32; 4],
    ) -> Result<Self, String> {
        Self::from_indexed_triangles(
            background,
            presentation
                .faces
                .iter()
                .enumerate()
                .flat_map(|(index, face)| {
                    face.triangles
                        .iter()
                        .map(move |&points| (points, color, Some(index)))
                }),
        )
    }

    /// Combine ordered scene meshes while keeping face ordinals distinct.
    /// Example: `assembly.append_triangles(component)`.
    pub fn append_triangles(&mut self, other: Self) {
        let offset = self
            .triangles
            .iter()
            .rev()
            .find_map(|triangle| triangle.face_index)
            .map_or(0, |index| index + 1);
        self.triangles
            .extend(other.triangles.into_iter().map(|mut triangle| {
                triangle.face_index = triangle.face_index.map(|index| index + offset);
                triangle
            }));
    }
}
fn display_triangle(
    positions: [[f64; 3]; 3],
    color: [f32; 4],
    face_index: Option<usize>,
) -> Result<ViewportTriangle, String> {
    let converted = positions.map(|point| point.map(|value| value as f32));
    if converted.iter().flatten().any(|value| !value.is_finite()) {
        return Err(format!(
            "display triangle {positions:?}: expected coordinates representable as finite display floats"
        ));
    }
    Ok(ViewportTriangle {
        positions: converted,
        color,
        face_index,
    })
}
#[cfg(test)]
#[path = "../tests/presentation.rs"]
mod tests;

/// A local display decline never changes the retained analytic body.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct PartPresentationDecline {
    pub feature: oblikovati_model::feature::FeatureId,
    pub cause: String,
}
/// Derived display geometry and all local conversion declines.
pub struct PartPresentation {
    pub scene: ViewportScene,
    pub visible_bodies: usize,
    pub declines: Vec<PartPresentationDecline>,
}
impl ViewportScene {
    /// Convert successful current bodies and retain local display failures.
    /// Example: `ViewportScene::from_part_evaluation(&evaluated, mesh_options, background, color)`.
    pub fn from_part_evaluation<Failure, Body: oblikovati_martensite::BodyPresentationSource>(
        evaluation: &oblikovati_model::feature::part::PartFeatureEvaluation<Body, Failure>,
        options: oblikovati_martensite::PrismMeshOptions,
        background: [f64; 3],
        color: [f32; 4],
    ) -> Result<PartPresentation, String> {
        Self::from_part_evaluation_with_visibility(evaluation, options, background, color, |_| {
            Ok(true)
        })
    }
    /// Filter display bodies without changing analytic results or feature health.
    /// Example: `ViewportScene::from_part_evaluation_with_visibility(&evaluated, options, background, color, visible)`.
    pub fn from_part_evaluation_with_visibility<
        Failure,
        Body: oblikovati_martensite::BodyPresentationSource,
    >(
        evaluation: &oblikovati_model::feature::part::PartFeatureEvaluation<Body, Failure>,
        options: oblikovati_martensite::PrismMeshOptions,
        background: [f64; 3],
        color: [f32; 4],
        mut visible: impl FnMut(&Body) -> Result<bool, String>,
    ) -> Result<PartPresentation, String> {
        Self::from_part_evaluation_styled(evaluation, options, background, |body| {
            Ok(visible(body)?.then_some(color))
        })
    }
    /// Give each body its own colour, or leave it out with `None`.
    /// Example: `ViewportScene::from_part_evaluation_styled(&evaluated, options, background, style)`.
    pub fn from_part_evaluation_styled<
        Failure,
        Body: oblikovati_martensite::BodyPresentationSource,
    >(
        evaluation: &oblikovati_model::feature::part::PartFeatureEvaluation<Body, Failure>,
        options: oblikovati_martensite::PrismMeshOptions,
        background: [f64; 3],
        mut style: impl FnMut(&Body) -> Result<Option<[f32; 4]>, String>,
    ) -> Result<PartPresentation, String> {
        let mut result = PartPresentation {
            scene: Self {
                background,
                triangles: Vec::new(),
            },
            visible_bodies: 0,
            declines: Vec::new(),
        };
        result.scene.validate()?;
        for record in evaluation.health() {
            let Some(body) = evaluation.body(record.feature) else {
                continue;
            };
            match style(body) {
                Ok(Some(color)) => result.append_body(record.feature, body, options, color),
                Ok(None) => {}
                Err(cause) => result.declines.push(PartPresentationDecline {
                    feature: record.feature,
                    cause,
                }),
            }
        }
        Ok(result)
    }
}
impl PartPresentation {
    fn append_body<Body: oblikovati_martensite::BodyPresentationSource>(
        &mut self,
        feature: oblikovati_model::feature::FeatureId,
        body: &Body,
        options: oblikovati_martensite::PrismMeshOptions,
        color: [f32; 4],
    ) {
        let scene = body.body_face_triangles(options).and_then(|faces| {
            ViewportScene::from_indexed_triangles(
                self.scene.background,
                faces.into_iter().enumerate().flat_map(|(index, face)| {
                    face.into_iter()
                        .map(move |points| (points, color, Some(index)))
                }),
            )
        });
        match scene {
            Ok(scene) => {
                self.scene.append_triangles(scene);
                self.visible_bodies += 1;
            }
            Err(cause) => self
                .declines
                .push(PartPresentationDecline { feature, cause }),
        }
    }
}

#[cfg(test)]
#[path = "../tests/part_presentation.rs"]
mod part_tests;
