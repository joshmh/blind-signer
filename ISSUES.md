# Blind Signer — Audit Issues

Audit of `stevie` v2.0.0 (`6f12b3e`) on 2026-10-04. Scope: `signer/cmd/stevie/`, `signer/internal/btc/`, build scripts. Line refs are against this tree. Severity reflects impact on **signing correctness** or **secret leakage** only.

---

## Non-issues (accepted by design)

- **Trusted PSBT input metadata** — witness UTXO script/value taken from the PSBT without signer-side verification (CVE-2020-14199 class). Accepted: multiple independent devices verify transactions before broadcast; blind signing runs in a physically secure but privacy-untrusted location. Residual risk to note: all verifiers sharing a bug or collusion; signer-side checks would still be defense-in-depth.
- `data/toxic-p/` **mnemonics + password** — confirmed test vectors, no value. Suggestion: switch fixtures to the standard BIP39 test vector (`abandon abandon … about`) so future audits/backups can't confuse them for real vaults.

---

## Open issues

### 1. HIGH — SIGHASH_DEFAULT inputs accepted but signed with SIGHASH_ALL

- **Where:** `internal/btc/btc.go:101-107`
- **Bug:** commit `6f12b3e` widened the accept guard to allow `SigHashDefault` (0x00) but the signing call still hardcodes `txscript.SigHashAll`. Result: signature's trailing hashtype byte (0x01) disagrees with the declared sighash (0x00) — nonstandard for segwit v0 → downstream validation/broadcast failure after the cold trip. This is the regression v2.0.0 was named for.
- **Fix (1 line, fail-closed):**

  ```diff
  - if input.SighashType != txscript.SigHashAll && input.SighashType != txscript.SigHashDefault {
  + if input.SighashType != txscript.SigHashAll {
  ```
- **Do NOT** pass `input.SighashType` through: sighash 0x00 is undefined for segwit v0 → invalid sig. Real SIGHASH_DEFAULT support = Taproot signing (BIP341/342, `RawTxInTapscriptSignature`) — separate feature.
- **Notes: yes, fix**

### 2. HIGH — Empty unsigned dir reports success and deletes seed material

- **Where:** `cmd/stevie/main.go:147-148` (`return signedCount == count`) → `main.go:41-47` (`os.RemoveAll("play/toxic")`)
- **Bug:** 0 == 0 is true. Running `stevie sign` with an empty/failed-copy `play/unsigned` prints "All transactions signed successfully.", **deletes mnemonic+password**, signs nothing, exits 0. Reproduced empirically.
- **Fix:** `return signedCount == count && count > 0`; print an explicit "no unsigned transactions found" branch; never wipe on count==0.
- **Notes: yes, fix**

### 3. HIGH — Hidden/junk files abort the batch with misleading output, exit 0

- **Where:** `cmd/stevie/main.go:116-128`
- **Bug:** `os.ReadDir` returns dotfiles (`.DS_Store`, `._*` AppleDouble, editor swaps). Each is parsed as a PSBT; one failure flips the run to "failed" — yet exit code stays 0. Reproduced: `.DS_Store` → "Signed 0 out of 1", exit 0. Operator can't distinguish drive pollution from real signing failure.
- **Fix:** skip names starting with `.`; optionally only attempt `*.psbt`; count skipped vs failed separately.
- **Notes: yes, \*.psbt only. do the the fix.**

### 4. HIGH — Signed outputs silently clobber / stale outputs look fresh

- **Where:** `internal/btc/btc.go:177` (`os.Create` truncates — comment admits it), `cmd/stevie/main.go:133-134` (output name = input stem + handle)
- **Bug:** (a) `invoice42.psbt` + `invoice42.txt` in unsigned → both map to `invoice42_signed_<handle>.psbt`; last writer wins → valid signature produced for the wrong txn. (b) A run failing before `os.Create` leaves yesterday's stale `_signed_*.psbt` looking like today's output (combined with exit 0, easy to carry home and broadcast). (c) rerun after a top-up silently replaces prior outputs; mid-failure after Create leaves truncated garbage.
- **Fix:** open with `O_CREATE|O_EXCL` (refuse if exists) or temp+rename; consider wiping `play/signed` at run start; encode account/net in filename.
- **Notes: yes, wipe play/signed at run start and** open with `O_CREATE|O_EXCL` 

### 5. HIGH — Derive errors swallowed while walking the derivation path

- **Where:** `internal/btc/btc.go:67-68` (`derivationKey, _ = derivationKey.Derive(d)`)
- **Bug:** malformed/hostile Bip32Path fails mid-walk; code proceeds with the partially-derived key and reports the misleading "Public Key does not match" instead of "invalid derivation path". Wrong diagnosis costs secure-location trips and trains users to ignore errors.
- **Fix:** check the error, wrap as `invalid derivation path`.
- **Notes: yes, fix**

### 6. HIGH — Weak filesystem permissions around seed material

- **Where:** `cmd/stevie/main.go:100` (`os.MkdirAll(dir, os.ModePerm)` → 0755 for `play/toxic`), `internal/btc/btc.go:177` (outputs 0644)
- **Bug:** raw mnemonic+password in world-readable-on-default dir on any multi-user/shared rescue host.
- **Fix:** 0700 for `play/toxic`, 0600 for files under it; 0600 (or at least 0640) for signed outputs.
- **Notes: yes, fix — but this is not meant to run a multi-user box so severity is medium or low**

### 7. HIGH — Seed retention: memory hygiene, interrupts, non-secure delete

- **Where:** `cmd/stevie/main.go` (mnemonic/password/seed/masterKey as GC strings, never zeroized), no SIGINT/SIGTERM handler anywhere, `os.RemoveAll` = plain unlink
- **Bug:** crash or ctrl-C leaves `play/toxic` fully populated; strings live in GC heap for process lifetime (swap/coredump exposure on the linux/amd64 build); deleted-but-unwiped content recoverable from slack/FTL.
- **Fix (staged):** signal handler → delete `play/toxic` before exit; wipe-before-unlink (overwrite + rename + remove); zero byte slices on defer.
- **Notes: play/toxic is meant to be memory mapped, not to disk; do not fix at this point.**

### 8. HIGH — Companion tools print secrets (outside the stevie binary, same module)

- **Where:** `cmd/stage/main.go:37-38` prints `Mnemonic:`/`Password:` literally; `internal/armory/mmc.go:48` hex-dumps proto payloads (mnemonic/password) → serial/UART logs on the Armory path
- **Fix:** remove/redact both.
- **Notes: remove**

### 9. MEDIUM — Exit code 0 on failure

- **Where:** `cmd/stevie/main.go:148` (always returns nil error; only usage errors exit 1)
- **Bug:** wrappers gating on `$?` proceed after failed/partial runs.
- **Fix:** non-zero exit when `signedCount != count` or count==0.
- **Notes: yes, fix**

### 10. MEDIUM — Account ≥ 2³¹ wraps the hardened bit

- **Where:** `cmd/stevie/main.go:81` (`ParseUint(...,32)` unbounded), `internal/btc/btc.go:155` (`HardenedKeyStart + account` wraps)
- **Bug:** `stevie sign frog 2147483648` silently becomes non-hardened m/48'/0'/0; fails closed via pubkey mismatch but with garbage diagnosis.
- **Fix:** reject account ≥ 2³¹ (BIP-32) at the CLI boundary.
- **Notes: make sure that this isn’t handled by the library; if it’s not, fix**

### 11. MEDIUM — Panic on crafted PSBT (index out of range)

- **Where:** `internal/btc/btc.go:30` (`input.NonWitnessUtxo.TxOut[voutIndex]` unguarded)
- **Bug:** malformed PSBT panics → crash → toxic dir retained (amplifies issue 7).
- **Fix:** bounds check, return error.
- **Notes: yes, fix**

### 12. MEDIUM — Fingerprint computed at account level (BIP-174 deviation)

- **Where:** `internal/btc/btc.go:140` (`ComputeFingerprint(childKey)` at account level; spec says master key fingerprint)
- **Bug:** internally consistent today (verified compatible with our PSBTs) but a spec deviation — silent breakage risk if the coordinator toolchain changes.
- **Fix:** document the deviation, or align with spec + update coordinators together.
- **Notes: if we change this, will it be incompatible with our PSBTs? we need to avoid breaking the workflow in the field.**

### 13. MEDIUM — No network validation of the PSBT

- **Where:** `cmd/stevie/main.go:29` (`chaincfg.MainNetParams` hardcoded); `--testnet` flips only the derivation coin-type
- **Bug:** mainnet PSBT + `--testnet` fails closed with confusing pubkey-mismatch noise instead of "wrong network".
- **Fix:** detect network from PSBT (bech32 HRP / addresses) and error explicitly on mismatch.
- **Notes: yes, fix**

### 14. MEDIUM — Legacy (non-segwit) inputs produce invalid signatures silently

- **Where:** `internal/btc/btc.go:105` (`RawTxInWitnessSignature` applied regardless of input type)
- **Bug:** legacy input → signature invalid; failure surfaces only at broadcast (DoS, not theft).
- **Fix:** detect missing `WitnessScript`/witness UTXO → explicit "legacy inputs unsupported" error before signing.
- **Notes: yes, fix**

---

## Low / smells

Note: Fix the checked items. explain the unchecked ones in more detail

- [x] `internal/btc/btc.go:40-53` — `log.Fatalf` inside library code (`ComputeFingerprint`) kills the process; unusable in tests.

- [x] `internal/btc/btc.go:121-123` — `errors.Wrap(err, ...)` wrapping a provably-nil error; Printf-without-verbs used for errors elsewhere.

- [x] `internal/btc/btc.go:129-132` — unreachable tail `return p, nil`; multi-derivation inputs silently sign only the first match; contract undocumented.

- [x] Dead debug prints at `internal/btc/btc.go:64,78`.

- [ ] `readData` TrimSpace silently alters passwords with meaningful whitespace (fails closed, misleading).

- [x] Arg parsing: extra args silently ignored (`stevie sign h 1 --testnet extra` accepted); no `--version`.

- [x] `linux-build.sh` builds single-file `./main.go` — any second file in package main is silently excluded (synced copy already has `main_extra_instructions.go` divergence); no `-trimpath`/ldflags version stamp; no `set -euo pipefail`.

- [x] `pop-tox.sh` — unquoted `$HANDLE`, no cleanup trap on interrupt.

- [ ] Supply chain: verified clean — no `replace` hijacks in any `go.mod` across both copies.

---

## Testing gap

**Zero** `*_test.go` **files in the entire workspace** (all five modules, both copies — verified). Minimum sufficient set, all deterministic with `t.TempDir()`:

- [ ] `processTransactions` table test: empty dir / only-dotfile / junk+valid / multi-file → assert counts, toxic-deletion, exit semantics (pins issues 2, 3)

- [ ] `SignTx` asserts emitted signature's sighash byte equals the input's declared type (pins issue 1 — exactly the delta of `6f12b3e`)

- [ ] Golden end-to-end PSBT fixture with fixed seed → byte-stable signature output (pins any future signing corruption)

- [ ] CLI boundary: account range, handle/path validation rejected cleanly

- [ ] Output overwrite policy (refuse-if-exists)

- **Notes: do the tests.**

---

## Decision log

- 2026-10-04 — PSBT-metadata trust accepted by design (multi-device verification downstream). See "Non-issues".
- 2026-10-04 — toxic-p fixtures confirmed test vectors, no rotation needed.