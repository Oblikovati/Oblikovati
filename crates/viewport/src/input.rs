// SPDX-License-Identifier: GPL-2.0-only
/// Copied physical modifiers; no native event or display pointer escapes the platform boundary.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct ViewportModifiers {
    pub shift: bool,
    pub control: bool,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ViewportButton {
    Left,
    Middle,
    Right,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ViewportKey {
    Pan,
    Zoom,
    Orbit,
    Cancel,
    Home,
    Previous,
    Save,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ViewportInput {
    Button {
        button: ViewportButton,
        pressed: bool,
        position: [i32; 2],
        modifiers: ViewportModifiers,
    },
    Motion {
        position: [i32; 2],
        modifiers: ViewportModifiers,
    },
    Wheel {
        notches: i32,
        position: [i32; 2],
    },
    Key {
        key: ViewportKey,
        pressed: bool,
    },
}
/// Camera-only requests to the kernel adapter; they cannot change document geometry.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum ViewportNavigation {
    /// Fit displayed geometry with a fractional border; example: `Fit { margin: 0.1 }`.
    Fit {
        margin: f64,
    },
    /// Restore the Y-up isometric view and fit; example: `Home { margin: 0.1 }`.
    Home {
        margin: f64,
    },
    Pan([i32; 2]),
    Orbit {
        yaw: f64,
        pitch: f64,
    },
    Zoom {
        factor: f64,
        anchor: Option<[i32; 2]>,
    },
}
impl ViewportNavigation {
    pub(crate) fn validate(self, extent: [u32; 2]) -> Result<(), String> {
        let valid = match self {
            Self::Fit { margin } | Self::Home { margin } => (0.0..1.0).contains(&margin),
            Self::Pan([_, y]) => y.checked_neg().is_some(),
            Self::Orbit { yaw, pitch } => yaw.is_finite() && pitch.is_finite(),
            Self::Zoom { factor, anchor } => {
                factor.is_finite()
                    && factor > 0.
                    && factor.recip().is_finite()
                    && valid_zoom_anchor(anchor, extent)
            }
        };
        if !valid {
            return Err(format!(
                "viewport navigation {self:?}, extent {extent:?}: expected finite camera movement and representable pixel coordinates"
            ));
        }
        Ok(())
    }
    pub(crate) fn is_stationary(self) -> bool {
        match self {
            Self::Fit { .. } | Self::Home { .. } => false,
            Self::Pan(delta) => delta == [0; 2],
            Self::Orbit { yaw, pitch } => yaw == 0. && pitch == 0.,
            Self::Zoom { factor, .. } => factor == 1.,
        }
    }
}
fn valid_zoom_anchor(anchor: Option<[i32; 2]>, extent: [u32; 2]) -> bool {
    anchor.is_none_or(|[_, y]| {
        i32::try_from(extent[1])
            .ok()
            .is_some_and(|height| height.checked_sub(y).is_some())
    })
}
#[cfg(test)]
#[path = "../tests/navigation_validation.rs"]
mod tests;
