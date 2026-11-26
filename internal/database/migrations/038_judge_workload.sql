CREATE VIEW judge_workload AS
SELECT language, status, created_at, updated_at, started_at, finished_at,
       jsonb_array_length(tests_json) AS tests,
       result_payload IS NOT NULL AS has_result,
       result_payload->>'verdict' AS verdict,
       (result_payload->>'compile_time_ms')::double precision AS compile_time_ms
FROM judge_jobs;

-- Legacy workers still need their token while an execution is active.
UPDATE judge_jobs SET callback_token = ''
WHERE status IN ('finished', 'failed') AND callback_token <> '';
