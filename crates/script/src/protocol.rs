// SPDX-License-Identifier: GPL-2.0-only
use crate::{ScriptProgram, ScriptReport};
use serde::{Deserialize, Serialize};
use std::io::{BufRead, Read, Write};
pub(crate) const MAX_FRAME: usize = 8 << 20;
#[derive(Serialize, Deserialize)]
#[serde(tag = "kind", rename_all = "camelCase")]
pub(crate) enum ParentFrame {
    Start { program: ScriptProgram },
    Reply { outcome: Result<Vec<u8>, String> },
}
#[derive(Serialize, Deserialize)]
#[serde(tag = "kind", rename_all = "camelCase")]
pub(crate) enum WorkerFrame {
    Call {
        method: String,
        arguments: Vec<u8>,
    },
    Print {
        line: String,
    },
    Finished {
        outcome: Result<ScriptReport, String>,
    },
}
pub(crate) fn write_frame<T: Serialize>(writer: &mut impl Write, frame: &T) -> Result<(), String> {
    let encoded = serde_json::to_vec(frame)
        .map_err(|error| format!("worker frame: expected serializable JSON: {error}"))?;
    if encoded.len() >= MAX_FRAME {
        return Err(format!(
            "worker frame {} bytes: expected less than {MAX_FRAME}",
            encoded.len()
        ));
    }
    writer
        .write_all(&encoded)
        .and_then(|()| writer.write_all(b"\n"))
        .and_then(|()| writer.flush())
        .map_err(|error| format!("worker pipe: expected writable frame: {error}"))
}
pub(crate) fn read_frame<T: serde::de::DeserializeOwned>(
    reader: &mut impl BufRead,
) -> Result<Option<T>, String> {
    let mut encoded = Vec::new();
    let length = reader
        .by_ref()
        .take((MAX_FRAME + 1) as u64)
        .read_until(b'\n', &mut encoded)
        .map_err(|error| format!("worker pipe: expected a readable frame: {error}"))?;
    if length == 0 {
        return Ok(None);
    }
    if length > MAX_FRAME || encoded.last() != Some(&b'\n') {
        return Err(format!(
            "worker frame {length} bytes: expected newline within {MAX_FRAME}"
        ));
    }
    decode_frame(&encoded).map(Some)
}
fn decode_frame<T: serde::de::DeserializeOwned>(encoded: &[u8]) -> Result<T, String> {
    serde_json::from_slice(encoded).map_err(|error| {
        format!(
            "worker frame {:?}: expected canonical protocol JSON: {error}",
            String::from_utf8_lossy(encoded)
        )
    })
}
