// SPDX-License-Identifier: GPL-2.0-only
//! Extended authored options for Hole v2 recipes (types, seats, drill points, termination surfaces).
use super::DatumLength;
use crate::parameter::ParameterCollection;

/// Hole feature classification matching the four Hole Properties type icons.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum HoleKind {
    #[default]
    Simple,
    Clearance,
    Tapped,
    TaperTapped,
}

/// Canonical Part origin plane selectable as a Hole `To Surface` terminator.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum HoleOriginPlane {
    Xy,
    Xz,
    Yz,
}

impl HoleOriginPlane {
    /// Return the origin point and unit normal for this canonical plane.
    /// Example: `let (origin, normal) = HoleOriginPlane::Xy.frame();`.
    pub fn frame(self) -> ([f64; 3], [f64; 3]) {
        match self {
            Self::Xy => ([0., 0., 0.], [0., 0., 1.]),
            Self::Xz => ([0., 0., 0.], [0., 1., 0.]),
            Self::Yz => ([0., 0., 0.], [1., 0., 0.]),
        }
    }

    /// Return the browser display name for this origin plane.
    /// Example: `assert_eq!(HoleOriginPlane::Xy.label(), "XY Plane");`.
    pub fn label(self) -> &'static str {
        match self {
            Self::Xy => "XY Plane",
            Self::Xz => "XZ Plane",
            Self::Yz => "YZ Plane",
        }
    }

    /// Parse a canonical browser plane label into an origin plane.
    /// Example: `HoleOriginPlane::from_label("XY Plane")`.
    pub fn from_label(label: &str) -> Option<Self> {
        let normalized = label.trim().to_ascii_lowercase();
        if normalized.starts_with("xy") {
            Some(Self::Xy)
        } else if normalized.starts_with("xz") {
            Some(Self::Xz)
        } else if normalized.starts_with("yz") {
            Some(Self::Yz)
        } else {
            None
        }
    }
}

/// Named surface or origin plane terminating a `To Surface` Hole.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(tag = "kind", content = "value", deny_unknown_fields)]
pub enum HoleTerminationSurface {
    OriginPlane(HoleOriginPlane),
    Face(Vec<u8>),
}

/// Authored seat profile at the entry side of a Hole.
#[derive(Clone, Copy, Debug, Default, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(tag = "kind", content = "value", deny_unknown_fields)]
pub enum HoleSeat {
    #[default]
    None,
    Counterbore {
        #[serde(with = "super::serde_inputs::length")]
        diameter: DatumLength,
        #[serde(with = "super::serde_inputs::length")]
        depth: DatumLength,
    },
    Spotface {
        #[serde(with = "super::serde_inputs::length")]
        diameter: DatumLength,
        #[serde(with = "super::serde_inputs::length")]
        depth: DatumLength,
    },
    Countersink {
        #[serde(with = "super::serde_inputs::length")]
        diameter: DatumLength,
        angle_deg: f64,
    },
}

/// Resolved numeric seat dimensions ready for kernel construction.
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub enum ResolvedHoleSeat {
    #[default]
    None,
    Counterbore {
        diameter: f64,
        depth: f64,
    },
    Spotface {
        diameter: f64,
        depth: f64,
    },
    Countersink {
        diameter: f64,
        angle_deg: f64,
    },
}

/// Bottom termination tip profile for finite-depth Holes.
#[derive(Clone, Copy, Debug, Default, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(tag = "kind", content = "value", deny_unknown_fields)]
pub enum HoleDrillPoint {
    #[default]
    Flat,
    Angle(f64),
}

/// Clearance fit selection for fastener-based clearance holes.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum HoleFastenerFit {
    Close,
    #[default]
    Normal,
    Loose,
}

impl HoleFastenerFit {
    /// Human-readable fit name for the Clearance Hole UI.
    pub fn label(self) -> &'static str {
        match self {
            Self::Close => "Close",
            Self::Normal => "Normal",
            Self::Loose => "Loose",
        }
    }
}

/// Authored fastener clearance selection stored on Clearance holes.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct HoleFastenerSpec {
    pub standard: String,
    pub fastener_type: String,
    pub size: String,
    pub fit: HoleFastenerFit,
}

/// Authored thread metadata stored on Tapped and Taper Tapped holes.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct HoleThreadSpec {
    pub profile: String,
    pub designation: String,
    pub full_depth: bool,
    pub left_hand: bool,
}

/// Optional v2 Hole settings; omitted from serialized v1 recipes when default.
#[derive(Clone, Debug, Default, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct HoleOptions {
    #[serde(default)]
    pub kind: HoleKind,
    #[serde(default)]
    pub seat: HoleSeat,
    #[serde(default)]
    pub drill_point: HoleDrillPoint,
    #[serde(default)]
    pub direction_flipped: bool,
    #[serde(default)]
    pub extend_start: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub fastener: Option<HoleFastenerSpec>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub thread: Option<HoleThreadSpec>,
}

impl HoleOptions {
    /// Report whether every option equals the v1 straight-bore default.
    /// Example: `assert!(HoleOptions::default().is_default());`.
    pub fn is_default(&self) -> bool {
        self.kind == HoleKind::Simple
            && self.seat == HoleSeat::None
            && self.drill_point == HoleDrillPoint::Flat
            && !self.direction_flipped
            && !self.extend_start
            && self.fastener.is_none()
            && self.thread.is_none()
    }

    pub(super) fn same_recipe(&self, other: &Self) -> bool {
        self.kind == other.kind
            && same_seat(self.seat, other.seat)
            && same_drill_point(self.drill_point, other.drill_point)
            && self.direction_flipped == other.direction_flipped
            && self.extend_start == other.extend_start
            && self.fastener == other.fastener
            && self.thread == other.thread
    }

    pub(super) fn seat_lengths(&self) -> Vec<DatumLength> {
        match self.seat {
            HoleSeat::None => Vec::new(),
            HoleSeat::Counterbore { diameter, depth } | HoleSeat::Spotface { diameter, depth } => {
                vec![diameter, depth]
            }
            HoleSeat::Countersink { diameter, .. } => vec![diameter],
        }
    }

    pub(super) fn validate(&self, bore_diameter: DatumLength) -> Result<(), String> {
        if self.kind == HoleKind::TaperTapped && self.seat != HoleSeat::None {
            return Err(
                "taper-tapped hole seat: expected no seat on a taper pipe-thread hole".into(),
            );
        }
        validate_seat(self.seat, bore_diameter)?;
        validate_drill_point(self.drill_point)?;
        Ok(())
    }

    pub(super) fn resolve_seat(
        &self,
        parameters: &ParameterCollection<u64>,
        bore_diameter: f64,
    ) -> Result<ResolvedHoleSeat, String> {
        match self.seat {
            HoleSeat::None => Ok(ResolvedHoleSeat::None),
            HoleSeat::Counterbore { diameter, depth } => {
                let d = super::positive_length(diameter, parameters, "seat diameter")?;
                let h = super::positive_length(depth, parameters, "seat depth")?;
                require_seat_wider(d, bore_diameter)?;
                Ok(ResolvedHoleSeat::Counterbore {
                    diameter: d,
                    depth: h,
                })
            }
            HoleSeat::Spotface { diameter, depth } => {
                let d = super::positive_length(diameter, parameters, "spotface diameter")?;
                let h = super::positive_length(depth, parameters, "spotface depth")?;
                require_seat_wider(d, bore_diameter)?;
                Ok(ResolvedHoleSeat::Spotface {
                    diameter: d,
                    depth: h,
                })
            }
            HoleSeat::Countersink {
                diameter,
                angle_deg,
            } => {
                let d = super::positive_length(diameter, parameters, "countersink diameter")?;
                require_seat_wider(d, bore_diameter)?;
                validate_angle(angle_deg, "countersink angle")?;
                Ok(ResolvedHoleSeat::Countersink {
                    diameter: d,
                    angle_deg,
                })
            }
        }
    }
}

fn validate_seat(seat: HoleSeat, bore: DatumLength) -> Result<(), String> {
    let seat_diameter = match seat {
        HoleSeat::None => return Ok(()),
        HoleSeat::Counterbore { diameter, depth } | HoleSeat::Spotface { diameter, depth } => {
            super::validate_positive_source(diameter, "seat diameter")?;
            super::validate_positive_source(depth, "seat depth")?;
            diameter
        }
        HoleSeat::Countersink {
            diameter,
            angle_deg,
        } => {
            super::validate_positive_source(diameter, "countersink diameter")?;
            validate_angle(angle_deg, "countersink angle")?;
            diameter
        }
    };
    if let (DatumLength::Constant(sd), DatumLength::Constant(bd)) = (seat_diameter, bore) {
        require_seat_wider(sd, bd)?;
    }
    Ok(())
}

fn require_seat_wider(seat_diameter: f64, bore_diameter: f64) -> Result<(), String> {
    if seat_diameter <= bore_diameter {
        return Err(format!(
            "hole seat diameter {seat_diameter}: expected strictly greater than hole diameter {bore_diameter}"
        ));
    }
    Ok(())
}

fn validate_drill_point(point: HoleDrillPoint) -> Result<(), String> {
    if let HoleDrillPoint::Angle(deg) = point {
        validate_angle(deg, "drill point angle")?;
    }
    Ok(())
}

fn validate_angle(angle_deg: f64, label: &str) -> Result<(), String> {
    if !angle_deg.is_finite() || angle_deg <= 0. || angle_deg >= 180. {
        return Err(format!(
            "hole {label} {angle_deg}: expected finite angle in (0, 180) degrees"
        ));
    }
    Ok(())
}

fn same_seat(first: HoleSeat, second: HoleSeat) -> bool {
    match (first, second) {
        (HoleSeat::None, HoleSeat::None) => true,
        (
            HoleSeat::Counterbore {
                diameter: d1,
                depth: h1,
            },
            HoleSeat::Counterbore {
                diameter: d2,
                depth: h2,
            },
        )
        | (
            HoleSeat::Spotface {
                diameter: d1,
                depth: h1,
            },
            HoleSeat::Spotface {
                diameter: d2,
                depth: h2,
            },
        ) => d1.same_recipe(d2) && h1.same_recipe(h2),
        (
            HoleSeat::Countersink {
                diameter: d1,
                angle_deg: a1,
            },
            HoleSeat::Countersink {
                diameter: d2,
                angle_deg: a2,
            },
        ) => d1.same_recipe(d2) && a1.to_bits() == a2.to_bits(),
        _ => false,
    }
}

fn same_drill_point(first: HoleDrillPoint, second: HoleDrillPoint) -> bool {
    match (first, second) {
        (HoleDrillPoint::Flat, HoleDrillPoint::Flat) => true,
        (HoleDrillPoint::Angle(a1), HoleDrillPoint::Angle(a2)) => a1.to_bits() == a2.to_bits(),
        _ => false,
    }
}
