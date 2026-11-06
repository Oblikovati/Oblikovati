// SPDX-License-Identifier: GPL-2.0-only
use crate::process::BuildResult;
use regex::Regex;
// tree-sitter-go 0.25.0 predates Go 1.27 generic methods (API client.go, #1650).
// Erase declaration type parameters only; preserving bytes keeps original-node spans valid.
pub(super) fn normalized(source: &str) -> BuildResult<String> {
    let headers =
        Regex::new(r"(?m)^func\s+\([^)]*\)\s+[\p{L}_][\p{L}\p{N}_]*\s*(\[)").map_err(|error| {
            format!("Go method header pattern: expected valid grammar adapter: {error}")
        })?;
    let mask = code_mask(source)?;
    let mut result = source.as_bytes().to_vec();
    for captures in headers.captures_iter(&mask) {
        let start = captures.get(1).expect("method bracket capture").start();
        let end = parameter_end(source.as_bytes(), start)?;
        erase_preserving_lines(&mut result[start..end]);
    }
    String::from_utf8(result)
        .map_err(|error| format!("Go normalization: expected unchanged UTF-8 offsets: {error}"))
}
fn parameter_end(source: &[u8], start: usize) -> BuildResult<usize> {
    let mut index = start;
    let mut depth: isize = 0;
    while index < source.len() {
        let (next, delta) = parameter_step(source, index)?;
        depth += delta;
        index = next;
        if depth == 0 {
            return Ok(index);
        }
    }
    Err(format!(
        "Go method type parameters at byte {start}: expected closing bracket"
    ))
}
fn parameter_step(source: &[u8], index: usize) -> BuildResult<(usize, isize)> {
    match source[index] {
        b'\'' | b'"' | b'`' => Ok((quoted_end(source, index)?, 0)),
        b'/' if matches!(source.get(index + 1), Some(b'/' | b'*')) => {
            Ok((comment_end(source, index)?, 0))
        }
        b'[' => Ok((index + 1, 1)),
        b']' => Ok((index + 1, -1)),
        _ => Ok((index + 1, 0)),
    }
}
fn quoted_end(source: &[u8], start: usize) -> BuildResult<usize> {
    let quote = source[start];
    let mut index = start + 1;
    while index < source.len() {
        if source[index] == quote {
            return Ok(index + 1);
        }
        if source[index] == b'\\' && quote != b'`' {
            index += 1;
        }
        index += 1;
    }
    Err(format!(
        "Go type-parameter quote at byte {start}: expected closing {}",
        char::from(quote)
    ))
}
fn comment_end(source: &[u8], start: usize) -> BuildResult<usize> {
    if source[start + 1] == b'/' {
        return Ok(source[start..]
            .iter()
            .position(|byte| *byte == b'\n')
            .map(|offset| start + offset)
            .unwrap_or(source.len()));
    }
    source[start + 2..]
        .windows(2)
        .position(|pair| pair == b"*/")
        .map(|offset| start + offset + 4)
        .ok_or_else(|| format!("Go type-parameter comment at byte {start}: expected closing */"))
}

fn code_mask(source: &str) -> BuildResult<String> {
    let bytes = source.as_bytes();
    let mut masked = bytes.to_vec();
    let mut index = 0;
    while index < bytes.len() {
        let end = match bytes[index] {
            b'\'' | b'"' | b'`' => quoted_end(bytes, index)?,
            b'/' if matches!(bytes.get(index + 1), Some(b'/' | b'*')) => comment_end(bytes, index)?,
            _ => {
                index += 1;
                continue;
            }
        };
        erase_preserving_lines(&mut masked[index..end]);
        index = end;
    }
    String::from_utf8(masked)
        .map_err(|error| format!("Go code mask: expected preserved UTF-8: {error}"))
}

fn erase_preserving_lines(source: &mut [u8]) {
    for byte in source {
        if *byte != b'\n' && *byte != b'\r' {
            *byte = b' ';
        }
    }
}
