#!/usr/bin/env bash
# Codesign, notarize, and staple the Oblikovati .app so a downloaded release launches
# on a clean Mac with no Gatekeeper prompt (the last piece of "no workarounds").
#
# Why each step:
#   - Sign inside-out (dylibs, then the bundle) with one Developer ID, under the
#     HARDENED RUNTIME (--options runtime). Notarization REQUIRES hardened runtime; and
#     because every dylib shares the binary's Team ID, hardened-runtime library
#     validation passes without the insecure disable-library-validation entitlement.
#   - Notarize: Apple scans the signed app and records its ticket. WITHOUT this, a
#     downloaded (quarantined) app is blocked even when correctly signed.
#   - Staple: attach the ticket to the .app so it verifies offline / first launch.
#
#   Usage: scripts/macos-sign.sh <app> <version> <out-dir>
# Env (provided by CI from repo secrets). When ANY is missing the macOS release is
# skipped (exit 0) rather than failing the pipeline — forks and unconfigured repos can
# still build everything else:
#   DEVELOPER_ID_APP  codesign identity, e.g. "Developer ID Application: Foo (TEAMID)"
#   AC_APPLE_ID       Apple ID email for notarytool
#   AC_PASSWORD       app-specific password for that Apple ID
#   AC_TEAM_ID        Apple Developer Team ID
set -euo pipefail

app="$1"
version="$2"
out="${3:-dist}"
mkdir -p "$out"
out="$(cd "$out" && pwd)"

# A signed+notarized bundle is the only way to ship a no-workaround Mac download, so
# without credentials we skip the whole macOS release instead of publishing something
# Gatekeeper would block.
for v in DEVELOPER_ID_APP AC_APPLE_ID AC_PASSWORD AC_TEAM_ID; do
	if [ -z "${!v:-}" ]; then
		echo "macos-sign: \$$v not set — Apple credentials not configured; skipping macOS signing + notarization."
		exit 0
	fi
done

sign() { codesign --force --options runtime --timestamp --sign "$DEVELOPER_ID_APP" "$@"; }

# Nested code first, then the bundle (which signs Contents/MacOS/oblikovati-head and
# seals the bundle). --timestamp contacts Apple's TSA, required for notarization.
sign "$app/Contents/Frameworks/"*.dylib
sign "$app"
codesign --verify --deep --strict --verbose=2 "$app"

# Notarize the zipped bundle and wait for Apple's verdict (non-zero exit fails the job).
# The zipped asset is lowercase `oblikovati-` for parity with the Linux/Windows/CLI
# release names; the bundle inside stays `Oblikovati.app` (its display name).
zip="$out/oblikovati-$version-macos-universal.zip"
ditto -c -k --keepParent "$app" "$zip"
xcrun notarytool submit "$zip" \
	--apple-id "$AC_APPLE_ID" --password "$AC_PASSWORD" --team-id "$AC_TEAM_ID" --wait

# Staple the ticket into the .app, then re-zip so the published artifact carries it.
xcrun stapler staple "$app"
xcrun stapler validate "$app"
rm -f "$zip"
ditto -c -k --keepParent "$app" "$zip"
echo "signed + notarized + stapled: $zip"
