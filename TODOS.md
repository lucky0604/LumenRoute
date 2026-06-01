# TODOS

Items deferred from implementation plans, with context for future pickup.

---

## Export Path Random I/O Optimization

**What:** Optimize `readCaptureLine` in the export handler to avoid per-record file open/seek/read/close.

**Why:** Exporting 10,000+ records hits random I/O hard. Each record opens a JSONL file, seeks to offset, reads one line, closes. Batch export of a full day's captures could take 10x longer than sequential reads.

**How to fix:** Sort export records by `file_path + byte_offset` before reading. Keep an LRU file handle pool (separate from the write-side pool) so sequential reads from the same file reuse the handle. This turns random I/O into mostly-sequential I/O.

**Depends on:** P4 (API layer / export handler) must exist first.

**Added:** 2026-05-29 via /plan-eng-review (Issue 14, unresolved)

---

## Sensitive Data Masking

**What:** Add configurable PII masking for captured request/response bodies before storage.

**Why:** At 7,000 users, prompts and responses will contain PII (names, emails, phone numbers, addresses). Storing raw bodies creates compliance risk, especially when data is exported to ipsa-eval for training dataset use.

**How to fix:** Add a per-project masking config (regex patterns or named entity types). Apply masking in `processLoop` before writing to JSONL. Consider making masking pluggable so different teams can define their own rules.

**Depends on:** Organization-level compliance strategy. Need to decide masking granularity (field-level vs full-body) and whether masked data is still useful for training.

**Added:** 2026-05-29 via /plan-eng-review
