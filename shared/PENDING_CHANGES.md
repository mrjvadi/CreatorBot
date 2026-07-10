# Pending Cross-Service Changes

This file tracks changes made in one service that require action in another.
Every service in this project depends on `shared` (directly or via
`shared-core`), so this is the common place any service's maintainer will
have on disk — check it before assuming a contract is finished on both ends.

Format per entry: which service raised it, what changed, who needs to act,
current status. Remove an entry once the required action is done; don't let
this file grow unbounded with stale/resolved items.

---

## 2026-07-02 — botmanager implemented the source-service worker contract

**Raised by:** source-service (an internal MTProto/UserBot automation tool used by core services — not a customer-facing BotInstance).

**Status:** ✅ **Done (botmanager side).** botmanager now responds on `SubjSourceWorkerRegister`/`SubjSourceWorkerUpdate` and subscribes to `SubjSourceWorkerHeartbeat` — see `botmanager/internal/sourceworker/responder.go`, wired from `cmd/main.go`. A new `models.SourceWorkerConfig` table (shared-core) holds the LicenseKey → worker_id/Telegram-credentials mapping; botmanager's admin panel (System → 🛰 Source Workers) creates/lists/enables/deletes rows. `AppHash`/`SessionKey` are stored AES-256-GCM encrypted with the same `ENCRYPTION_KEY` used for `BotInstance.BotToken`.

**Known gap, not yet closed:** `SourceWorkerUpdateRequest` only carries a task-correlation `ID` (from source-service's own `task.Envelope`), not a `WorkerID` or anything else that maps to a botmanager owner/chat. Until whatever dispatches those tasks (and mints that correlation ID) exists somewhere in botmanager, the "relay to a user-facing bot" behavior described below can't be implemented — `handleUpdate` currently just validates the ServiceKey, logs the payload, and acks. Whoever builds the task-dispatch side should add a correlation-id → owner/chat lookup at that point.

**Needs verification:** local sandbox can't fully resolve Go modules (no `go.sum` committed, and module-proxy fetches don't persist across calls here) — `gofmt` passed clean and all new identifiers cross-checked by hand/grep, but `go build ./...` should still be run on your machine before deploying.

---

## 2026-07-03 — dead/orphaned code found in shared-core (review only, no service raised this)

**Raised by:** shared-core code review (not triggered by a change in any consumer — found by grepping for actual usage across the whole monorepo).

**Finding 1 — `shared-core/docstore/uploader.go` (`CodeStore`, `FileStore`) and `shared-core/documents/uploader.go` (`Code`, `File`, `CodeUsage`, all `primitive.ObjectID`-keyed):** not imported anywhere in the monorepo. `uploader-bot` has its own complete, string-UUID-keyed models/store (`internal/models`, `internal/store` — folders, payments, locks, ads, admin perms, backups) that superseded this. `docstore.BotUserStore` in the same `uploader.go` file is still live (used by every bot via `engine.go`) — only `CodeStore`/`FileStore` and the `Code`/`File`/`CodeUsage` types are dead.
**Who needs to act:** whoever owns `shared-core` — safe to delete `CodeStore`, `FileStore`, and the `Code`/`File`/`CodeUsage` structs; nothing else references them so removal can't break a build.
**Status:** 🟡 not yet done — flagged, not removed (didn't want to touch shared-core code without confirmation).

**Finding 2 — `shared-core/schema` package (`Create`/`Drop`/`DSN`/`Exists`/`ListInstanceSchemas` for per-instance Postgres schemas):** also not called from anywhere in the monorepo. Matches the platform's stated long-term goal in root `CLAUDE.md` ("physical DB separation is a long-term goal, not done yet") — reads like scaffolding built ahead of that migration rather than a bug.
**Who needs to act:** whoever is driving the Postgres-per-instance migration — either wire it in when that work starts, or leave as-is if the plan is still active. No action needed if this is intentional groundwork.
**Status:** ℹ️ informational — not broken, just confirming it's unused today.
