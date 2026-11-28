#!/usr/bin/env python3
"""Run headless Chromium WebGPU end-to-end verification against the full Oblikovati desktop UI."""
import argparse
import base64
import functools
import gzip
import http.server
import json
import pathlib
import sys
import threading
import time

WEB_ROOT = pathlib.Path(__file__).resolve().parent
PKG = WEB_ROOT / "pkg"


def analyze_screenshot(page, png_bytes):
    b64 = base64.b64encode(png_bytes).decode("ascii")
    return page.evaluate(
        """async (b64) => {
        const img = new Image();
        img.src = "data:image/png;base64," + b64;
        await img.decode();
        const canvas = document.createElement("canvas");
        canvas.width = img.width;
        canvas.height = img.height;
        const ctx = canvas.getContext("2d");
        ctx.drawImage(img, 0, 0);
        const rgba = ctx.getImageData(0, 0, canvas.width, canvas.height).data;
        let nonBlack = 0;
        const colors = new Set();
        for (let i = 0; i < rgba.length; i += 4) {
            const r = rgba[i], g = rgba[i + 1], b = rgba[i + 2];
            if (r > 5 || g > 5 || b > 5) nonBlack++;
            if (colors.size < 256) colors.add(((r >> 3) << 10) | ((g >> 3) << 5) | (b >> 3));
        }
        return { nonBlack, distinctColorBuckets: colors.size };
    }""",
        b64,
    )


class IsolatedHandler(http.server.SimpleHTTPRequestHandler):
    extensions_map = {
        **http.server.SimpleHTTPRequestHandler.extensions_map,
        ".wasm": "application/wasm",
        ".js": "application/javascript",
    }

    def end_headers(self):
        self.send_header("Cross-Origin-Opener-Policy", "same-origin")
        self.send_header("Cross-Origin-Embedder-Policy", "require-corp")
        self.send_header("Cross-Origin-Resource-Policy", "same-origin")
        super().end_headers()

    def log_message(self, format, *args):
        # Intentionally silent so stderr only reports E2E progress and failures.
        return


def main():
    from playwright.sync_api import sync_playwright

    parser = argparse.ArgumentParser()
    parser.add_argument("--screenshot-dir", type=pathlib.Path, default=None)
    args = parser.parse_args()

    wasm_file = PKG / "oblikovati_web_bg.wasm"
    if not wasm_file.is_file():
        raise FileNotFoundError(f"missing {wasm_file}; run `cargo xtask wasm-build` first")
    raw_bytes = wasm_file.read_bytes()
    gz_bytes = gzip.compress(raw_bytes, compresslevel=9)

    handler = functools.partial(IsolatedHandler, directory=str(WEB_ROOT))
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), handler)
    port = server.server_address[1]
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()

    console_logs = []
    try:
        with sync_playwright() as p:
            browser = p.chromium.launch(
                headless=True,
                args=[
                    "--enable-unsafe-webgpu",
                    "--use-angle=vulkan",
                    "--enable-features=Vulkan",
                    "--ignore-gpu-blocklist",
                    "--no-sandbox",
                ],
            )
            page = browser.new_page(viewport={"width": 1280, "height": 720})
            page.on("console", lambda msg: console_logs.append(f"{msg.type}: {msg.text}"))
            page.on("pageerror", lambda err: console_logs.append(f"pageerror: {err}"))

            page.goto(f"http://127.0.0.1:{port}/?validation=1", wait_until="domcontentloaded")
            page.wait_for_function("() => Boolean(window.__OBLIKOVATI_WEB__)", timeout=30000)

            home_metrics = page.evaluate(
                """async () => {
                const cap = await window.__OBLIKOVATI_WEB__.capture();
                const rgba = new Uint8Array(cap.rgba);
                let nonBlack = 0;
                const colors = new Set();
                for (let i = 0; i < rgba.length; i += 4) {
                    const r = rgba[i], g = rgba[i + 1], b = rgba[i + 2];
                    if (r > 5 || g > 5 || b > 5) nonBlack++;
                    if (colors.size < 256) colors.add(((r >> 3) << 10) | ((g >> 3) << 5) | (b >> 3));
                }
                return {
                    crossOriginIsolated: window.crossOriginIsolated,
                    overlayHidden: document.getElementById("boot-overlay").classList.contains("hidden"),
                    status: cap.status,
                    validationErrors: cap.validationErrors,
                    nonBlack,
                    distinctColorBuckets: colors.size,
                };
            }"""
            )

            home_png = page.screenshot()
            home_visible = analyze_screenshot(page, home_png)
            if args.screenshot_dir:
                args.screenshot_dir.mkdir(parents=True, exist_ok=True)
                (args.screenshot_dir / "home_canvas.png").write_bytes(home_png)

            # Click the primary "New Part" button on the Home screen and verify the Part UI opens.
            page.mouse.click(100, 230)
            page.wait_for_function(
                """async () => {
                const res = await window.__OBLIKOVATI_WEB__.status();
                return res.status.openDocuments >= 1;
            }""",
                timeout=15000,
            )
            time.sleep(0.3)
            part_metrics = page.evaluate(
                """async () => {
                const cap = await window.__OBLIKOVATI_WEB__.capture();
                return {
                    status: cap.status,
                    validationErrors: cap.validationErrors,
                };
            }"""
            )
            part_png = page.screenshot()
            part_visible = analyze_screenshot(page, part_png)
            if args.screenshot_dir:
                (args.screenshot_dir / "part_canvas.png").write_bytes(part_png)

            browser.close()
    finally:
        server.shutdown()

    report = {
        "wasmRawBytes": len(raw_bytes),
        "wasmGzipBytes": len(gz_bytes),
        "crossOriginIsolated": home_metrics["crossOriginIsolated"],
        "overlayHidden": home_metrics["overlayHidden"],
        "deviceName": home_metrics["status"]["deviceName"],
        "activeWorkers": home_metrics["status"]["activeWorkers"],
        "homeNonBlackPixels": home_metrics["nonBlack"],
        "homeDistinctColorBuckets": home_metrics["distinctColorBuckets"],
        "homeVisibleCanvasNonBlackPixels": home_visible["nonBlack"],
        "homeVisibleCanvasColorBuckets": home_visible["distinctColorBuckets"],
        "partOpenDocuments": part_metrics["status"]["openDocuments"],
        "partVisibleCanvasNonBlackPixels": part_visible["nonBlack"],
        "partVisibleCanvasColorBuckets": part_visible["distinctColorBuckets"],
        "validationErrors": len(home_metrics["validationErrors"])
        + len(part_metrics["validationErrors"]),
        "consoleLogs": console_logs,
    }
    print(json.dumps(report, indent=2))

    assert report["crossOriginIsolated"], "expected crossOriginIsolated=true"
    assert report["overlayHidden"], "expected boot overlay hidden after initialization"
    assert report["homeNonBlackPixels"] > 100000, "expected rendered Oblikovati desktop UI"
    assert report["homeDistinctColorBuckets"] >= 32, "expected rich desktop UI theme and icons"
    assert (
        report["homeVisibleCanvasNonBlackPixels"] > 100000
    ), "expected visible browser canvas to display Home desktop UI"
    assert (
        report["homeVisibleCanvasColorBuckets"] >= 32
    ), "expected visible browser canvas to preserve Home UI colors"
    assert report["partOpenDocuments"] >= 1, "expected New Part click to open a Part document"
    assert (
        report["partVisibleCanvasNonBlackPixels"] > 100000
    ), "expected visible browser canvas to display Part desktop UI"
    assert (
        report["partVisibleCanvasColorBuckets"] >= 32
    ), "expected visible browser canvas to preserve Part UI colors"
    assert report["validationErrors"] == 0, f"WebGPU validation errors: {report}"


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"E2E FAILED: {exc}", file=sys.stderr)
        sys.exit(1)
