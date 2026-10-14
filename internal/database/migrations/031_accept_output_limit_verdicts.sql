-- Keep callback storage compatible with the trusted supervisor output limits.
alter table submissions drop constraint submissions_verdict_check;
alter table submissions add constraint submissions_verdict_check
    check (verdict is null or verdict in ('accepted','wrong_answer','time_limit_exceeded','memory_limit_exceeded','output_limit_exceeded','output_directory_limit_exceeded','runtime_error','compile_error','presentation_error','system_error'));

alter table code_runs drop constraint code_runs_verdict_check;
alter table code_runs add constraint code_runs_verdict_check
    check (verdict is null or verdict in ('accepted','wrong_answer','time_limit_exceeded','memory_limit_exceeded','output_limit_exceeded','output_directory_limit_exceeded','runtime_error','compile_error','presentation_error','system_error'));

alter table submission_test_results drop constraint submission_test_results_status_check;
alter table submission_test_results add constraint submission_test_results_status_check
    check (status in ('accepted','wrong_answer','time_limit_exceeded','memory_limit_exceeded','output_limit_exceeded','runtime_error','skipped','system_error'));

alter table code_run_test_results drop constraint code_run_test_results_status_check;
alter table code_run_test_results add constraint code_run_test_results_status_check
    check (status in ('accepted','wrong_answer','time_limit_exceeded','memory_limit_exceeded','output_limit_exceeded','runtime_error','skipped','system_error'));
