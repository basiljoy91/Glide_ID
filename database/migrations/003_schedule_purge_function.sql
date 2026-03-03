-- 003_schedule_purge_function.sql
-- Schedule automated data purging (GDPR/CCPA compliance) using pg_cron if available.

DO $$
BEGIN
  -- Create pg_cron extension if it exists in this environment.
  -- In Supabase, pg_cron is available but CREATE EXTENSION may be managed by the platform;
  -- the IF NOT EXISTS guard avoids failures on subsequent runs.
  BEGIN
    CREATE EXTENSION IF NOT EXISTS pg_cron;
  EXCEPTION
    WHEN OTHERS THEN
      -- If pg_cron cannot be created (e.g. restricted environment), just skip scheduling.
      PERFORM 1;
  END;

  -- Schedule daily purge at 02:00 if pg_cron is present.
  IF EXISTS (
    SELECT 1
    FROM pg_extension
    WHERE extname = 'pg_cron'
  ) THEN
    -- Avoid duplicate jobs by deleting any existing job with the same name.
    PERFORM cron.unschedule('purge-terminated-employees')
    WHERE EXISTS (
      SELECT 1
      FROM cron.job
      WHERE jobname = 'purge-terminated-employees'
    );

    PERFORM cron.schedule(
      'purge-terminated-employees',
      '0 2 * * *',
      'SELECT purge_terminated_employee_data();'
    );
  END IF;
END;
$$;

