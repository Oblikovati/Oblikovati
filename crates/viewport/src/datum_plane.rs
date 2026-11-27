// SPDX-License-Identifier: GPL-2.0-only
//! A work-plane overlay is presentation only and is never solid topology.
use oblikovati_model::theme::Rgba;

/// One evaluated plane's display rectangle and theme colors.
#[derive(Clone, Debug, PartialEq)]
pub struct ViewportDatumPlane {
    pub corners: [[f64; 3]; 4],
    pub fill: Rgba,
    pub border: Rgba,
}

impl ViewportDatumPlane {
    /// Refuse non-finite overlay inputs; example: `plane.validate()?` before rendering.
    pub fn validate(&self) -> Result<(), String> {
        Self::validate_corners(&self.corners)?;
        self.validate_colors()?;
        Ok(())
    }
    /// Reject degenerate or unrepresentable geometry; example: `ViewportDatumPlane::validate_corners(&plane.corners)?`.
    pub fn validate_corners(corners: &[[f64; 3]; 4]) -> Result<(), String> {
        if corners
            .iter()
            .flatten()
            .any(|value| !(*value as f32).is_finite())
        {
            return Err(format!(
                "datum corners {:?}: expected finite f32 coordinates",
                corners
            ));
        }
        let points = corners.map(|point| point.map(|coordinate| coordinate as f32));
        if crate::render_normal::triangle_normal([points[0], points[1], points[2]]) == [0.; 3] {
            return Err(format!(
                "datum corners {:?}: expected a nondegenerate display rectangle",
                corners
            ));
        }
        Ok(())
    }
    fn validate_colors(&self) -> Result<(), String> {
        for color in [&self.fill, &self.border] {
            if [color.r, color.g, color.b, color.a]
                .iter()
                .any(|channel| !channel.is_finite() || !(0.0..=1.0).contains(channel))
            {
                return Err(format!(
                    "datum color {color:?}: expected finite RGBA in [0, 1]"
                ));
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rejects_invalid_positions_and_color_channels() {
        let mut plane = ViewportDatumPlane {
            corners: [[0., 0., 0.], [1., 0., 0.], [1., 1., 0.], [0., 1., 0.]],
            fill: Rgba {
                r: 0.2,
                g: 0.3,
                b: 0.4,
                a: 0.5,
            },
            border: Rgba {
                r: 1.,
                g: 1.,
                b: 1.,
                a: 1.,
            },
        };
        assert!(plane.validate().is_ok());
        plane.corners[0][0] = f64::MAX;
        assert!(plane.validate().is_err());
        plane.corners[0][0] = 0.;
        plane.fill.a = f32::NAN;
        assert!(plane.validate().is_err());
    }
}
