# SQL jobs

The `sql` job runs on each deployment. When `SQL_JOB_FILE` is unset, empty, or
whitespace-only, it logs that no script was selected and exits successfully.
Otherwise it executes a repository-owned file from `tasks/sql_job/sql/` in the
deployed image, after the database readiness check succeeds.

## Selecting and running a script

Set `SQL_JOB_FILE` in the deployment configuration to the filename of a script
bundled in the image. Surrounding whitespace is trimmed. Paths and raw SQL are
not accepted. Deploy the configuration; the deployment creates a
`sql-${IMAGE_TAG}-${CJI_UID}` ClowdJobInvocation automatically.

Confirm that the generated pod uses the expected image and `SQL_JOB_FILE`.
Check the job status and logs for `Starting SQL job`, followed by either
`SQL job completed` or `SQL job failed`. Failures exit nonzero.
After completion, restore `SQL_JOB_FILE` to its empty default to avoid accidental
reuse. Leaving a file selected reruns it on subsequent deployments. Kubernetes
may also retry failed pods. Script authors must make scripts idempotent or
document when reruns are safe, and operators must clear the selection when done.

The job uses the `vmaas_sync` database role and primary database. Scripts are
limited to that role's permissions. Do not grant admin access just to make a
script work. Schema changes belong in database migrations.

## Execution contract

- Each file runs in one transaction. Errors, timeout, or cancellation roll it back.
- Files contain PostgreSQL SQL, without `psql` commands or transaction control
  such as `BEGIN`, `COMMIT`, or `ROLLBACK`.
- A transaction-scoped advisory lock, 74201901, rejects concurrent SQL runners.
  It does not coordinate with application writers or migration lock 123.
- `SQL_JOB_CONFIG` configures `sql_lock_timeout_seconds`, default 30, and
  `sql_statement_timeout_seconds`, default 1500. Both must be positive.
  The statement timeout also bounds the runner's total execution context.
- `SQL_JOB_TIMEOUT`, default 1800 seconds, bounds the Kubernetes job lifetime.
  Keep it longer than the execution timeout to allow startup and rollback.
- The runner does not track execution history or make scripts idempotent.
  Review rerun safety, lock duration, WAL volume, replication impact, and
  verification before running a script on production data.

## Adding a script

Add a `.sql` file under `tasks/sql_job/sql/`. The image bundles this directory.
Prefix filenames with a three-digit number followed by a descriptive name, using
`NNN_description.sql`, for example `001_backfill_owner_id.sql`.
Use the next unused number; gaps are allowed.
This is a naming convention, not a runner requirement or execution order.
Include tests and document the script's prerequisites, rerun safety, and
verification queries alongside the script.

Before selecting a script for deployment:

1. Test it in stage and assess runtime, locking, and database load.
2. Check that it is safe while old and new application pods run concurrently.
   The database readiness check waits for schema migrations, not application rollout.
3. Define how to verify the result and recover from failure or application rollback.
4. Confirm when its selection must be removed, especially if later code changes
   would make rerunning it unsafe.

After execution, verify the data as well as the job's completion status. Clear
`SQL_JOB_FILE` when the operation is finished.
