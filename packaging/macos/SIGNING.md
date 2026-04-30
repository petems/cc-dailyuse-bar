# macOS Code Signing & Notarization

For tagged releases this happens automatically in `.github/workflows/release.yml`
on `macos-latest`. This document is the local/manual reference for ad-hoc
debugging — if you're cutting a release, just push a tag.

## Prerequisites

- An Apple Developer membership ($99/year) with a **Developer ID Application** certificate
- Xcode command line tools (`xcode-select --install`)
- The certificate installed in your login Keychain
- An [App Store Connect API key](https://appstoreconnect.apple.com/access/integrations/api)
  (the .p8 file, plus its Key ID and Issuer ID) for notarization

## Build the .app bundle

```bash
make build
make bundle-macos
```

The `bundle-macos` target stamps `CFBundleVersion` / `CFBundleShortVersionString`
from `$(VERSION)` (defaults to `git describe`). Override `BINARY_PATH` if you want
to bundle a pre-built binary (for example, the universal Mach-O produced by
`goreleaser build`):

```bash
make bundle-macos BINARY_PATH=dist/darwin-universal_darwin_all/cc-dailyuse-bar
```

## Code Signing

Sign the bundle with hardened runtime + secure timestamp (both required for notarization):

```bash
codesign --force --deep --options=runtime --timestamp \
  --entitlements packaging/macos/entitlements.plist \
  --sign "Developer ID Application: YOUR_NAME (TEAM_ID)" \
  "CC Daily Use Bar.app"
```

### Entitlements

`packaging/macos/entitlements.plist` enables:

- `com.apple.security.cs.allow-unsigned-executable-memory` — required because the Go runtime allocates executable memory
- `com.apple.security.cs.disable-library-validation` — required for Go's dynamic loading of system libraries

### Verify signing

```bash
codesign --verify --deep --strict --verbose=2 "CC Daily Use Bar.app"
```

## Notarization (App Store Connect API key)

This is what CI uses. The legacy `notarytool store-credentials` /
`--keychain-profile` flow still works for one-off local runs but isn't
documented here — see `xcrun notarytool store-credentials --help`.

### Submit the .app for notarization

```bash
ditto -c -k --keepParent "CC Daily Use Bar.app" "CC Daily Use Bar.zip"

xcrun notarytool submit "CC Daily Use Bar.zip" \
  --key   /path/to/AuthKey_XXXXXXXXXX.p8 \
  --key-id "$AC_API_KEY_ID" \
  --issuer "$AC_API_ISSUER_ID" \
  --wait
```

`--wait` blocks until Apple finishes (typically 1-3 minutes). On success the
final line says `status: Accepted`. On failure, run
`xcrun notarytool log <submission-id> --key ... --key-id ... --issuer ...` to
see what tripped.

### Staple the ticket

After acceptance, embed the notarization ticket so Gatekeeper can verify offline:

```bash
xcrun stapler staple "CC Daily Use Bar.app"
```

### Verify notarization

```bash
spctl --assess --type exec --verbose=2 "CC Daily Use Bar.app"
xcrun stapler validate "CC Daily Use Bar.app"
```

Expected: `accepted source=Notarized Developer ID` and
`The validate action worked!`.

## Build, sign, and notarize the DMG

```bash
make dmg-macos

DMG=$(ls dist/cc-dailyuse-bar_*_universal.dmg)
codesign --force --timestamp --sign "Developer ID Application: YOUR_NAME (TEAM_ID)" "$DMG"

xcrun notarytool submit "$DMG" \
  --key   /path/to/AuthKey_XXXXXXXXXX.p8 \
  --key-id "$AC_API_KEY_ID" \
  --issuer "$AC_API_ISSUER_ID" \
  --wait

xcrun stapler staple "$DMG"
```

Notarizing both the .app and the DMG is belt-and-braces — the .app's stapled
ticket guarantees offline acceptance after the DMG is unpacked, and the DMG's
ticket means Gatekeeper accepts the disk image itself without an internet check.

## Gatekeeper notes

- **Unsigned binaries** trigger "cannot be opened because the developer cannot be verified" on macOS 10.15+
- **Signed but not notarized** binaries trigger a bypassable warning dialog
- **Signed and notarized** (and stapled) binaries open without any warnings, even offline

## Ad-hoc signing (for local development)

For local testing without a Developer ID certificate:

```bash
codesign --deep --force --sign - "CC Daily Use Bar.app"
```

This won't pass Gatekeeper but is fine for spot-checking the bundle layout.
