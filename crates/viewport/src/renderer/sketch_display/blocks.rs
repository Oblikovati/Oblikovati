// SPDX-License-Identifier: GPL-2.0-only
//! Draw block definitions through their instance placement after curve sampling.
use super::*;
use crate::ViewportSketchBlock;
use oblikovati_martensite::planar_curve::sketch::SketchDisplayReport;
use oblikovati_model::document_sketch::SketchBlockInstance;

impl SketchDisplayOverlay {
    pub(super) fn append_block(
        &mut self,
        viewer: &V3dViewer,
        sketch: &ViewportSketchDisplay,
        block: &ViewportSketchBlock,
        deflection: f64,
    ) {
        let report = match sample_block(block, deflection) {
            Ok(report) => report,
            Err(cause) => {
                self.declines
                    .push(display_decline(sketch, block.instance.id(), cause));
                return;
            }
        };
        self.append_block_report(viewer, sketch, block, report);
    }
    fn append_block_report(
        &mut self,
        viewer: &V3dViewer,
        sketch: &ViewportSketchDisplay,
        block: &ViewportSketchBlock,
        report: SketchDisplayReport,
    ) {
        let definition = block_display(sketch, block);
        for decline in report.declines {
            self.declines.push(display_decline(
                sketch,
                block.instance.id(),
                format!("block child {:?}: {}", decline.entity, decline.cause),
            ));
        }
        for point in report.center_markers {
            let entity = center_marker(point);
            self.append_transformed_block_entity(viewer, &definition, &block.instance, entity);
        }
        for entity in report.entities {
            self.append_transformed_block_entity(viewer, &definition, &block.instance, entity);
        }
    }
    fn append_transformed_block_entity(
        &mut self,
        viewer: &V3dViewer,
        sketch: &ViewportSketchDisplay,
        instance: &SketchBlockInstance,
        mut entity: SketchDisplayEntity,
    ) {
        match transform_primitive(&entity.primitive, instance) {
            Ok(primitive) => {
                entity.primitive = primitive;
                self.append_entity_for(viewer, sketch, &entity, instance.id(), instance.color());
            }
            Err(cause) => self.declines.push(display_decline(
                sketch,
                instance.id(),
                format!("block child {:?}: {cause}", entity.entity),
            )),
        }
    }
}
fn sample_block(
    block: &ViewportSketchBlock,
    deflection: f64,
) -> Result<SketchDisplayReport, String> {
    sample_sketch_display(
        &block.geometry,
        SketchSamplingOptionsV1 {
            deflection: deflection / block.instance.scale().abs(),
            maximum_curve_points: 4096,
        },
    )
}
pub(in crate::renderer) fn block_display(
    sketch: &ViewportSketchDisplay,
    block: &ViewportSketchBlock,
) -> ViewportSketchDisplay {
    ViewportSketchDisplay {
        sketch: sketch.sketch,
        plane: sketch.plane,
        geometry: block.geometry.clone(),
        blocks: Vec::new(),
        line_color: block
            .definition_color
            .as_ref()
            .map(super::api_color)
            .or_else(|| sketch.line_color.clone()),
    }
}
fn center_marker(point: oblikovati_model::sketch::SketchPoint2D) -> SketchDisplayEntity {
    SketchDisplayEntity {
        entity: point.id,
        construction: false,
        centerline: false,
        primitive: SketchDisplayPrimitive::Point(point.position),
    }
}
pub(in crate::renderer) fn transform_primitive(
    primitive: &SketchDisplayPrimitive,
    instance: &SketchBlockInstance,
) -> Result<SketchDisplayPrimitive, String> {
    match primitive {
        SketchDisplayPrimitive::Point(point) => Ok(SketchDisplayPrimitive::Point(transform_point(
            *point, instance,
        )?)),
        SketchDisplayPrimitive::Curve(points) => Ok(SketchDisplayPrimitive::Curve(
            points
                .iter()
                .map(|point| transform_point(*point, instance))
                .collect::<Result<Vec<_>, _>>()?,
        )),
    }
}
fn transform_point([x, y]: [f64; 2], instance: &SketchBlockInstance) -> Result<[f64; 2], String> {
    instance
        .to_sketch([x, y])
        .map_err(|cause| cause.to_string())
}
#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::sketch::SketchEntityId;
    #[test]
    fn placement_rotates_scales_and_translates_and_refuses_overflow() {
        let instance = SketchBlockInstance::new(
            SketchEntityId(1),
            "Rib",
            [3., 4.],
            std::f64::consts::FRAC_PI_2,
            2.,
        )
        .unwrap();
        let result = transform_point([1., 0.], &instance).unwrap();
        assert!((result[0] - 3.).abs() < 1e-12 && (result[1] - 6.).abs() < 1e-12);
        let huge =
            SketchBlockInstance::new(SketchEntityId(2), "Rib", [0., 0.], 0., f64::MAX).unwrap();
        assert!(
            transform_point([2., 0.], &huge)
                .unwrap_err()
                .contains("finite")
        );
    }
}
