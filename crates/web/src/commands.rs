// SPDX-License-Identifier: GPL-2.0-only
//! Typed JSON command and UI snapshot contracts between the browser shell and worker runtime.

use oblikovati_viewport::{ViewportBackground, ViewportCamera};
use serde::{Deserialize, Serialize};

/// Visual theme selection for the browser workbench and viewport background.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum WebTheme {
    #[default]
    Dark,
    Light,
}

impl WebTheme {
    /// Resolve the viewport background matching this web workbench theme.
    /// Example: `let bg = WebTheme::Dark.viewport_background();`.
    pub fn viewport_background(self) -> ViewportBackground {
        match self {
            Self::Dark => ViewportBackground::VerticalGradient {
                top: [26, 33, 44],
                bottom: [13, 17, 23],
            },
            Self::Light => ViewportBackground::VerticalGradient {
                top: [232, 238, 246],
                bottom: [204, 214, 228],
            },
        }
    }

    /// Resolve the fallback linear RGB background color carried in `ViewportScene`.
    /// Example: `let rgb = WebTheme::Light.scene_background();`.
    pub fn scene_background(self) -> [f64; 3] {
        match self {
            Self::Dark => [0.06, 0.08, 0.11],
            Self::Light => [0.85, 0.88, 0.92],
        }
    }
}

/// Standard camera orientations exposed by the browser toolbar and ViewCube actions.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum WebViewOrientation {
    Iso,
    Front,
    Back,
    Left,
    Right,
    Top,
    Bottom,
}

/// Commands dispatched from the main-thread browser shell to the Orchestrator Worker.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "camelCase")]
pub enum WebCommand {
    NewPart {
        name: Option<String>,
    },
    LoadSamplePart,
    ActivateDocument {
        id: u64,
    },
    CloseDocument {
        id: u64,
    },
    SetParameter {
        name: String,
        expression: String,
    },
    Undo,
    Redo,
    Orient {
        view: WebViewOrientation,
    },
    Fit,
    Home,
    SetProjection {
        perspective: bool,
    },
    SetTheme {
        theme: WebTheme,
    },
    SetNormalDebug {
        enabled: bool,
    },
    Orbit {
        yaw: f64,
        pitch: f64,
    },
    Pan {
        dx: i32,
        dy: i32,
    },
    Zoom {
        factor: f64,
        anchor: Option<[i32; 2]>,
    },
}

/// Summary of an open document in the browser workspace.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WebDocumentSummary {
    pub id: u64,
    pub name: String,
    pub active: bool,
    pub dirty: bool,
}

/// Summary of a feature node shown in the browser Model Browser.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WebFeatureSummary {
    pub id: u64,
    pub name: String,
    pub kind: String,
    pub status: String,
}

/// Summary of a document parameter editable from the browser sidebar.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WebParameterSummary {
    pub id: u64,
    pub name: String,
    pub expression: String,
    pub value: Option<f64>,
    pub unit: String,
}

/// Serializable camera state mirrored into the browser HUD.
#[derive(Clone, Copy, Debug, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WebCameraSnapshot {
    pub eye: [f64; 3],
    pub target: [f64; 3],
    pub up: [f64; 3],
    pub scale: f64,
    pub perspective: bool,
}

impl From<ViewportCamera> for WebCameraSnapshot {
    fn from(camera: ViewportCamera) -> Self {
        Self {
            eye: camera.eye,
            target: camera.target,
            up: camera.up,
            scale: camera.scale,
            perspective: camera.perspective,
        }
    }
}

/// Immutable snapshot of the web application state pushed to the browser UI.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WebStateSnapshot {
    pub active_document: Option<u64>,
    pub documents: Vec<WebDocumentSummary>,
    pub features: Vec<WebFeatureSummary>,
    pub parameters: Vec<WebParameterSummary>,
    pub visible_bodies: usize,
    pub triangle_count: usize,
    pub can_undo: bool,
    pub can_redo: bool,
    pub theme: WebTheme,
    pub perspective: bool,
    pub normal_debug: bool,
    pub camera: WebCameraSnapshot,
    pub device_name: String,
    pub active_workers: usize,
    pub validation_errors: usize,
    pub status_message: String,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn commands_and_themes_round_trip_through_json() {
        let cmd = WebCommand::SetParameter {
            name: "depth".into(),
            expression: "5 cm".into(),
        };
        let encoded = serde_json::to_string(&cmd).unwrap();
        let decoded: WebCommand = serde_json::from_str(&encoded).unwrap();
        assert_eq!(decoded, cmd);

        assert_ne!(
            WebTheme::Dark.viewport_background(),
            WebTheme::Light.viewport_background()
        );
        assert_ne!(
            WebTheme::Dark.scene_background(),
            WebTheme::Light.scene_background()
        );
        let cam = WebCameraSnapshot::from(ViewportCamera::default());
        assert_eq!(cam.scale, 20.0);
    }
}
