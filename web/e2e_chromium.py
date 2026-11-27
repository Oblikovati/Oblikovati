#!/usr/bin/env python3
"""Run headless Chromium WebGPU end-to-end verification against the built Oblikovati Web shell."""
import functools
import gzip
import http.server
import json
import pathlib
import sys
import threading

WEB_ROOT = pathlib.Path(__file__).resolve().parent
PKG = WEB_ROOT / "pkg"


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
            version = browser.version
            page = browser.new_page(viewport={"width": 1280, "height": 800})
            page.on("console", lambda msg: console_logs.append(f"[page:{msg.type}] {msg.text}"))
            page.on("pageerror", lambda err: console_logs.append(f"[pageerror] {err}"))
            page.on(
                "worker",
                lambda w: w.on(
                    "console",
                    lambda msg: console_logs.append(f"[worker:{msg.type}] {msg.text}"),
                ),
            )
            page.goto(f"http://127.0.0.1:{port}/index.html")
            page.wait_for_function(
                "window.__oblikovatiWebReady === true || window.__oblikovatiWebError !== undefined",
                timeout=120_000,
            )
            boot_error = page.evaluate("window.__oblikovatiWebError || null")
            if boot_error:
                raise RuntimeError(f"Boot failed: {boot_error}")

            result = page.evaluate(
                """async () => {
                const app = window.__oblikovatiWeb;
                const cap1 = await app.capture();
                const pixels1 = new Uint8Array(cap1.rgba);
                let lit1 = 0;
                for (let i = 0; i < pixels1.length; i += 4) {
                    if (pixels1[i] > 60 && pixels1[i + 1] > 60 && pixels1[i + 2] > 60) lit1++;
                }

                const edited = await app.dispatchCommand({
                    type: "setParameter",
                    name: "depth",
                    expression: "5 cm"
                });
                const cap2 = await app.capture();
                const pixels2 = new Uint8Array(cap2.rgba);
                let diffPixels = 0;
                for (let i = 0; i < Math.min(pixels1.length, pixels2.length); i += 4) {
                    if (
                        Math.abs(pixels1[i] - pixels2[i]) > 8 ||
                        Math.abs(pixels1[i + 1] - pixels2[i + 1]) > 8 ||
                        Math.abs(pixels1[i + 2] - pixels2[i + 2]) > 8
                    ) diffPixels++;
                }

                const exported = await app.exportDocument();
                const exportedBytes = new Uint8Array(exported.bytes);
                const reopened = await app.loadDocument("roundtrip.opd", Array.from(exportedBytes));
                const undone = await app.dispatchCommand({ type: "orient", view: "front" });
                const finalState = await app.state();
                await app.close();

                return {
                    crossOriginIsolated: app.crossOriginIsolated,
                    mainThreadBlocked: app.mainThreadBlocked,
                    deviceName: cap1.state.deviceName,
                    activeWorkers: cap1.state.activeWorkers,
                    visibleBodies: cap1.state.visibleBodies,
                    triangleCount: cap1.state.triangleCount,
                    litPixels: lit1,
                    paramEditChangedPixels: diffPixels,
                    editedCanUndo: edited.canUndo,
                    exportedBytes: exportedBytes.length,
                    reopenedBodies: reopened.state.visibleBodies,
                    frontViewEyeZ: undone.camera.eye[2],
                    validationErrors: finalState.validationErrors.length,
                };
            }"""
            )
            browser.close()
    except Exception:
        if console_logs:
            print("Console logs:\n" + "\n".join(console_logs), file=sys.stderr)
        raise
    finally:
        server.shutdown()

    report = {
        "chromiumVersion": version,
        "wasmRawBytes": len(raw_bytes),
        "wasmGzipBytes": len(gz_bytes),
        **result,
    }
    print(json.dumps(report, indent=2))
    assert report["crossOriginIsolated"] is True
    assert report["mainThreadBlocked"] is False
    assert report["activeWorkers"] >= 1
    assert report["visibleBodies"] == 1
    assert report["triangleCount"] >= 12
    assert report["litPixels"] > 500, f"expected rendered solid pixels, got {report}"
    assert report["paramEditChangedPixels"] > 100, f"expected parametric edit visual change: {report}"
    assert report["editedCanUndo"] is True
    assert report["exportedBytes"] > 100
    assert report["reopenedBodies"] == 1
    assert report["validationErrors"] == 0, f"WebGPU validation errors: {report}"


if __name__ == "__main__":
    main()
