// SPDX-License-Identifier: GPL-2.0-only
use crate::{ViewportScene, ViewportTriangle};
use oblikovati_martensite::PrismPresentation;
impl ViewportScene {
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
            "prism triangle {positions:?}: expected coordinates representable as finite display floats"
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
    pub fn from_part_evaluation<Failure, Body: oblikovati_martensite::PrismPresentationSource>(
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
    /// Example: `ViewportScene::from_part_evaluation_with_visibility(&evaluated, options, background, color, |_| Ok(false))`.
    pub fn from_part_evaluation_with_visibility<
        Failure,
        Body: oblikovati_martensite::PrismPresentationSource,
    >(
        evaluation: &oblikovati_model::feature::part::PartFeatureEvaluation<Body, Failure>,
        options: oblikovati_martensite::PrismMeshOptions,
        background: [f64; 3],
        color: [f32; 4],
        mut visible: impl FnMut(&Body) -> Result<bool, String>,
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
            if visible(body)? {
                result.append_body(record.feature, body, options, color);
            }
        }
        Ok(result)
    }
}
impl PartPresentation {
    fn append_body<Body: oblikovati_martensite::PrismPresentationSource>(
        &mut self,
        feature: oblikovati_model::feature::FeatureId,
        body: &Body,
        options: oblikovati_martensite::PrismMeshOptions,
        color: [f32; 4],
    ) {
        let scene = body
            .presentation(options)
            .and_then(|mesh| ViewportScene::from_prism(&mesh, self.scene.background, color));
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

#[cfg(test)]
#[path = "../tests/part_presentation.rs"]
mod part_tests;
