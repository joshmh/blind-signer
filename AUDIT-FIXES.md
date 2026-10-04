# Audit fix log

This file records decisions taken while implementing the notes in `ISSUES.md`.

## Issue 12 — fingerprint computed at account level

**Decision: not changed.** Changing `ComputeFingerprint` input to the master
key would change every fingerprint this signer emits. The PSBT flow matches
input derivations by comparing the fingerprint embedded in the PSBT against
the signer's computed fingerprint; existing coordinator-exported PSBTs carry
the account-level fingerprint, so aligning with the BIP-174 spec (master
fingerprint) would reject every in-flight PSBT and break the field workflow.
The deviation is internally consistent and stays documented here; revisit only
as a coordinated change with the coordinator toolchain.

## Issue 7 — seed retention (not fixed, per notes)

Not addressed: `play/toxic` is intended to be a memory-mapped location, not
disk, and the fix (signal handlers, wipe-before-unlink) is deferred.

## Issue 6 — permissions

Applied despite the medium/low severity note: 0700 for `play/toxic` creation
(via `createDirectory`) and the recreated `play/signed`, 0600 for signed
outputs (`internal/btc/btc.go`). No cost on a single-user rescue box.

## Low/smells left unchecked (explanation requested by the audit)

- **`readData` TrimSpace** (`cmd/stevie/main.go`): left as is. The mnemonic
  and password are burned to SD via `host-burner`, whose prompt path strips
  the trailing newline before writing (`burner.go` drops the final `\n`), and
  files on the transfer medium are plain text drops; a password with
  meaningful leading/trailing whitespace would have to survive the burner
  prompt round-trip first, which it cannot today. Removing TrimSpace now
  would change behavior for whitespace-padded files without a fixture that
  proves the need; if passwords ever gain meaningful whitespace, this must be
  revisited together with the burner prompt handling. Note: fails closed —
  a mangled password yields a wrong key and a pubkey mismatch, never a wrong
  signature.

## Testing

Deterministic tests added under `signer/internal/btc/` and
`signer/cmd/stevie/` (see `Testing gap` in ISSUES.md). The golden end-to-end
fixture pins a fixed seed → byte-stable PSBT signature.
