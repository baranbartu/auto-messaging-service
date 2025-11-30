# Assumptions & Tradeoffs

This document records the explicit decisions made while building the Auto Messaging Service so reviewers understand the intent behind the implementation.

## Database Access
- **No ORM** – All persistence is handled through `database/sql` + pgx to keep control over queries, avoid extra abstractions, and keep migrations/queries transparent during the assessment.
- **Custom migration runner** – SQL files in `migrations/` are executed on startup to guarantee schema availability in local/docker environments. In production this would likely move to a separate migration step, but auto-running keeps onboarding simple here.

## Scheduler Behavior
- **Interval guard** – Even if `SCHEDULER_INTERVAL` is set lower, the loader clamps it to ≥2 minutes to respect the spec, preventing accidental rapid polling.
- **Control endpoints** – `/api/v1/control/*` start and stop the background goroutine so operators can pause/resume the loop without restarting the container.

## Webhook Handling
- **Static webhook.site response** – Webhook.site cannot generate randomized `messageId` values without custom scripts/a paid plan, so the demo uses a fixed JSON payload. The service still validates the body shape (`{ "message": "Accepted", "messageId": "..." }`) before marking DB rows as sent.
- **Headers** – `x-ins-auth-key` is forwarded from `WEBHOOK_AUTH_KEY` even though webhook.site ignores it, because the spec required the header.

## Redis Usage
- **Cached fields** – Each accepted send stores the remote `messageId`, the local UUID, and the timestamp in a Redis hash (key `sent_message:<remote-id>`). This mirrors the “bonus” requirement exactly while also giving a practical lookup path to correlate webhook IDs with local records if future features need it.

## Configuration & Secrets
- **.env-driven defaults** – Docker Compose, the Go app, and README instructions all reference the same `.env` variables so secrets (DB creds, webhook URL) are defined once. In real deployments these would live in a secret manager rather than plaintext env files.

## Testing Footprint
- **Demonstrative unit test** – Added a focused config test to verify the scheduler interval floor and to show the intended testing approach. Broader behavior and integration tests would follow as the feature set matures.

These notes should clarify the “why” behind the current design and highlight areas that would be revisited for a production deployment.
