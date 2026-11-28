// SPDX-License-Identifier: GPL-2.0-only
//! A work-axis overlay is presentation only and is never solid topology (#3918).
use oblikovati_model::theme::Rgba;

/// One evaluated work axis's finite 3D display segment and theme styling.
#[derive(Clone, Debug, PartialEq)]
pub struct ViewportDatumAxis {
    pub endpoints: [[f64; 3]; 2],
    pub color: Rgba,
    pub width: f32,
    pub selected: bool,
}

impl ViewportDatumAxis {
    /// Refuse non-finite or degenerate axis overlay inputs; example: `axis.validate()?`.
    pub fn validate(&self) -> Result<(), String> {
        if self
            .endpoints
            .iter()
            .flatten()
            .any(|value| !(*value as f32).is_finite())
        {
            return Err(format!(
                "datum axis endpoints {:?}: expected finite f32 coordinates",
                self.endpoints
            ));
        }
        let delta = [
            self.endpoints[1][0] - self.endpoints[0][0],
            self.endpoints[1][1] - self.endpoints[0][1],
            self.endpoints[1][2] - self.endpoints[0][2],
        ];
        let length = delta[0].hypot(delta[1]).hypot(delta[2]);
        if !length.is_finite() || length <= 0.0 {
            return Err(format!(
                "datum axis endpoints {:?}: expected distinct endpoints",
                self.endpoints
            ));
        }
        if !self.width.is_finite() || self.width <= 0.0 {
            return Err(format!(
                "datum axis width {}: expected a finite positive width",
                self.width
            ));
        }
        let color = &self.color;
        if [color.r, color.g, color.b, color.a]
            .iter()
            .any(|channel| !channel.is_finite() || !(0.0..=1.0).contains(channel))
        {
            return Err(format!(
                "datum axis color {color:?}: expected finite RGBA in [0, 1]"
            ));
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn validates_axis_endpoints_width_and_color() {
        let mut axis = ViewportDatumAxis {
            endpoints: [[0.0, 0.0, -5.0], [0.0, 0.0, 5.0]],
            color: Rgba {
                r: 0.9,
                g: 0.7,
                b: 0.2,
                a: 1.0,
            },
            width: 1.5,
            selected: false,
        };
        assert!(axis.validate().is_ok());
        axis.endpoints[1] = axis.endpoints[0];
        assert!(axis.validate().is_err());
        axis.endpoints[1] = [0.0, 0.0, 5.0];
        axis.width = 0.0;
        assert!(axis.validate().is_err());
    }
}
