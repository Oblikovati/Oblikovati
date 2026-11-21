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
        let triangles = triangles
            .iter()
            .map(|(points, color)| display_triangle(*points, *color))
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
        let mut triangles = Vec::new();
        for face in &presentation.faces {
            for positions in &face.triangles {
                triangles.push(display_triangle(*positions, color)?);
            }
        }
        let scene = Self {
            background,
            triangles,
        };
        scene.validate()?;
        Ok(scene)
    }
}
fn display_triangle(positions: [[f64; 3]; 3], color: [f32; 4]) -> Result<ViewportTriangle, String> {
    let converted = positions.map(|point| point.map(|value| value as f32));
    if converted.iter().flatten().any(|value| !value.is_finite()) {
        return Err(format!(
            "display triangle {positions:?}: expected coordinates representable as finite display floats"
        ));
    }
    Ok(ViewportTriangle {
        positions: converted,
        color,
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
        let scene = body
            .body_triangles(options)
            .and_then(|triangles| body_scene(triangles, self.scene.background, color));
        match scene {
            Ok(mut scene) => {
                self.scene.triangles.append(&mut scene.triangles);
                self.visible_bodies += 1;
            }
            Err(cause) => self
                .declines
                .push(PartPresentationDecline { feature, cause }),
        }
    }
}

fn body_scene(
    triangles: Vec<[[f64; 3]; 3]>,
    background: [f64; 3],
    color: [f32; 4],
) -> Result<ViewportScene, String> {
    let triangles = triangles
        .into_iter()
        .map(|triangle| display_triangle(triangle, color))
        .collect::<Result<_, _>>()?;
    let scene = ViewportScene {
        background,
        triangles,
    };
    scene.validate()?;
    Ok(scene)
}

#[cfg(test)]
#[path = "../tests/part_presentation.rs"]
mod part_tests;
