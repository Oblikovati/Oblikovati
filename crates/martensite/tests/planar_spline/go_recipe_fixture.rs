// SPDX-License-Identifier: GPL-2.0-only
use serde::Deserialize;

#[derive(Deserialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub struct GoSplineOracle {
    pub method: u8,
    pub closed: bool,
    pub degree: i32,
    pub poles: Vec<[f64; 2]>,
    pub knots: Vec<f64>,
    pub stations: Vec<f64>,
    pub samples: Vec<[f64; 5]>,
}

pub const FIT_POINTS: [[f64; 2]; 5] = [[0., 0.], [0.2, 1.], [2., 1.5], [3., -0.5], [1., -2.]];
