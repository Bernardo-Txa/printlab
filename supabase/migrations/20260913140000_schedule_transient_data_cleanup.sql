create extension if not exists pg_cron;

-- Supabase Cron schedules use UTC. Run during a low-traffic window.
select cron.unschedule(jobid)
from cron.job
where jobname = 'printlab_transient_data_cleanup';

select cron.schedule(
  'printlab_transient_data_cleanup',
  '17 3 * * *',
  $$
    delete from public.admin_sessions
    where expires_at <= now();

    delete from public.carts
    where expires_at <= now();
  $$
);
