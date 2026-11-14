// SPDX-License-Identifier: GPL-2.0-only

pub(super) struct FitStep {
    pub upper_mm: f64,
    pub widths_um: [u16; 7],
    pub shaft_deviations_um: [i16; 3],
}

// ISO 286-2:2010, tables 1, 18, 20, 21. #3573 replaces the Go formula's
// incorrect rounding with integer micrometres; no platform-dependent pow call.
pub(super) const STEPS: [FitStep; 12] = [
    FitStep {
        upper_mm: 6.0,
        widths_um: [5, 8, 12, 18, 30, 48, 75],
        shaft_deviations_um: [-30, -10, -4],
    },
    FitStep {
        upper_mm: 10.0,
        widths_um: [6, 9, 15, 22, 36, 58, 90],
        shaft_deviations_um: [-40, -13, -5],
    },
    FitStep {
        upper_mm: 18.0,
        widths_um: [8, 11, 18, 27, 43, 70, 110],
        shaft_deviations_um: [-50, -16, -6],
    },
    FitStep {
        upper_mm: 30.0,
        widths_um: [9, 13, 21, 33, 52, 84, 130],
        shaft_deviations_um: [-65, -20, -7],
    },
    FitStep {
        upper_mm: 50.0,
        widths_um: [11, 16, 25, 39, 62, 100, 160],
        shaft_deviations_um: [-80, -25, -9],
    },
    FitStep {
        upper_mm: 80.0,
        widths_um: [13, 19, 30, 46, 74, 120, 190],
        shaft_deviations_um: [-100, -30, -10],
    },
    FitStep {
        upper_mm: 120.0,
        widths_um: [15, 22, 35, 54, 87, 140, 220],
        shaft_deviations_um: [-120, -36, -12],
    },
    FitStep {
        upper_mm: 180.0,
        widths_um: [18, 25, 40, 63, 100, 160, 250],
        shaft_deviations_um: [-145, -43, -14],
    },
    FitStep {
        upper_mm: 250.0,
        widths_um: [20, 29, 46, 72, 115, 185, 290],
        shaft_deviations_um: [-170, -50, -15],
    },
    FitStep {
        upper_mm: 315.0,
        widths_um: [23, 32, 52, 81, 130, 210, 320],
        shaft_deviations_um: [-190, -56, -17],
    },
    FitStep {
        upper_mm: 400.0,
        widths_um: [25, 36, 57, 89, 140, 230, 360],
        shaft_deviations_um: [-210, -62, -18],
    },
    FitStep {
        upper_mm: 500.0,
        widths_um: [27, 40, 63, 97, 155, 250, 400],
        shaft_deviations_um: [-230, -68, -20],
    },
];
