# macOS Code Signing & Notarization

This document describes how to sign and notarize cc-dailyuse-bar for macOS distribution.

## Prerequisites

- An Apple Developer account ($99/year) with a **Developer ID Application** certificate
- Xcode command line tools installed (`xcode-select --install`)
- The certificate installed in your Keychain

## Build the .app Bundle

```bash
make bundle-macos
```

This creates `CC Daily Use Bar.app/` with the correct directory structure.

## Code Signing

Sign the bundle with hardened runtime (required for notarization):

```bash
codesign --deep --force \
  --options=runtime \
  --entitlements=packaging/macos/entitlements.plist \
  --sign "Developer ID Application: YOUR_NAME (TEAM_ID)" \
  "CC Daily Use Bar.app"
```

### Entitlements

The `entitlements.plist` includes:

- `allow-unsigned-executable-memory` — required because the Go runtime allocates executable memory
- `disable-library-validation` — required for Go's dynamic library loading

### Verify Signing

```bash
codesign --verify --deep --strict --verbose=2 "CC Daily Use Bar.app"
```

## Notarization

### Store Credentials (one-time setup)

```bash
xcrun notarytool store-credentials "CC_DAILYUSE_BAR" \
  --apple-id "your@email.com" \
  --team-id "TEAM_ID"
```

You'll be prompted for an app-specific password (generate at appleid.apple.com).

### Create a ZIP for Submission

```bash
ditto -c -k --keepParent "CC Daily Use Bar.app" "CC Daily Use Bar.zip"
```

### Submit for Notarization

```bash
xcrun notarytool submit "CC Daily Use Bar.zip" \
  --keychain-profile "CC_DAILYUSE_BAR" \
  --wait
```

### Staple the Ticket

After successful notarization, staple the ticket for offline verification:

```bash
xcrun stapler staple "CC Daily Use Bar.app"
```

### Verify Notarization

```bash
spctl --assess --type exec --verbose=2 "CC Daily Use Bar.app"
```

Expected output: `CC Daily Use Bar.app: accepted`

## Gatekeeper Notes

- **Unsigned binaries** will trigger "cannot be opened because the developer cannot be verified" on macOS 10.15+
- **Signed but not notarized** binaries will trigger a warning dialog that users can bypass
- **Signed and notarized** binaries open without any warnings

## Ad-hoc Signing (for local development)

For local testing without a Developer ID certificate:

```bash
codesign --deep --force --sign - "CC Daily Use Bar.app"
```

This won't pass Gatekeeper but is useful for local development and testing.
