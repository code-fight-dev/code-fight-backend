-- Enable verified runtimes only for the built-in algorithmic catalog.
-- Custom tasks and SQL tasks keep their existing language configuration.
with starters(slug, cpp, javascript, typescript) as (
    values
    ('signal-stabilizer', '// The judge passes parsed arguments to Solve. Return the answer.
#include <string>
#include <vector>

long long Solve(const std::vector<long long>& band, const std::vector<long long>& readings) {
    return 0;
}
', '// The judge passes parsed arguments to Solve. Return the answer.
function Solve(band, readings) {
    return 0;
}
', '// The judge passes parsed arguments to Solve. Return the answer.
export function Solve(band: number[], readings: number[]): number {
  return 0;
}
'),
    ('mirror-log', '// The judge passes parsed arguments to Solve. Return the answer.
#include <string>
#include <vector>

bool Solve(const std::string& log) {
    return false;
}
', '// The judge passes parsed arguments to Solve. Return the answer.
/** @param {string} log */
function Solve(log) {
    return false;
}
', '// The judge passes parsed arguments to Solve. Return the answer.
export function Solve(log: string): boolean {
  return false;
}
'),
    ('packet-window-median', '// The judge passes parsed arguments to Solve. Return the answer.
#include <string>
#include <vector>

std::vector<double> Solve(const std::vector<long long>& latency, int k) {
    return {};
}
', '// The judge passes parsed arguments to Solve. Return the answer.
function Solve(latency, k) {
    return [];
}
', '// The judge passes parsed arguments to Solve. Return the answer.
export function Solve(latency: number[], k: number): number[] {
  return [];
}
'),
    ('adaptive-pricing', '// The judge passes parsed arguments to Solve. Return the answer.
#include <string>
#include <vector>

long long Solve(const std::vector<long long>& prices) {
    return 0;
}
', '// The judge passes parsed arguments to Solve. Return the answer.
function Solve(prices) {
    return 0;
}
', '// The judge passes parsed arguments to Solve. Return the answer.
export function Solve(prices: number[]): number {
  return 0;
}
'),
    ('binary-signal-boundary', '// The judge passes parsed arguments to Solve. Return the answer.
#include <string>
#include <vector>

long long Solve(const std::vector<long long>& signals, long long threshold) {
    return 0;
}
', '// The judge passes parsed arguments to Solve. Return the answer.
function Solve(signals, threshold) {
    return 0;
}
', '// The judge passes parsed arguments to Solve. Return the answer.
export function Solve(signals: number[], threshold: number): number {
  return 0;
}
'),
    ('colony-perimeter', '// The judge passes parsed arguments to Solve. Return the answer.
#include <string>
#include <vector>

long long Solve(const std::vector<std::vector<int>>& grid) {
    return 0;
}
', '// The judge passes parsed arguments to Solve. Return the answer.
function Solve(grid) {
    return 0;
}
', '// The judge passes parsed arguments to Solve. Return the answer.
export function Solve(grid: number[][]): number {
  return 0;
}
'),
    ('quantum-safe-lock', '// The judge passes parsed arguments to Solve. Return the answer.
#include <string>
#include <vector>

std::string Solve(int n) {
    return "";
}
', '// The judge passes parsed arguments to Solve. Return the answer.
function Solve(n) {
    return "";
}
', '// The judge passes parsed arguments to Solve. Return the answer.
export function Solve(n: number): string {
  return "";
}
'),
    ('trade-ledger-balance', '// The judge passes parsed arguments to Solve. Return the answer.
#include <string>
#include <vector>

long long Solve(const std::vector<long long>& ledger) {
    return 0;
}
', '// The judge passes parsed arguments to Solve. Return the answer.
function Solve(ledger) {
    return 0;
}
', '// The judge passes parsed arguments to Solve. Return the answer.
export function Solve(ledger: number[]): number {
  return 0;
}
'),
    ('sensor-drift', '// The judge passes parsed arguments to Solve. Return the answer.
#include <string>
#include <vector>

long long Solve(const std::vector<long long>& readings, int k, double threshold) {
    return 0;
}
', '// The judge passes parsed arguments to Solve. Return the answer.
function Solve(readings, k, threshold) {
    return 0;
}
', '// The judge passes parsed arguments to Solve. Return the answer.
export function Solve(readings: number[], k: number, threshold: number): number {
  return 0;
}
'),
    ('cache-key-collisions', '// Parse the full test input and return the answer as a string.
#include <string>

std::string Solve(const std::string& input) {
    return "";
}
', '// Parse the full test input and return the answer as a string.
function solve(input) {
    return "";
}
', '// Parse the full test input and return the answer as a string.
export function Solve(input: string): string {
  return "";
}
'),
    ('courier-routes', '// Parse the full test input and return the answer as a string.
#include <string>

std::string Solve(const std::string& input) {
    return "";
}
', '// Parse the full test input and return the answer as a string.
function solve(input) {
    return "";
}
', '// Parse the full test input and return the answer as a string.
export function Solve(input: string): string {
  return "";
}
'),
    ('build-pipeline-order', '// Parse the full test input and return the answer as a string.
#include <string>

std::string Solve(const std::string& input) {
    return "";
}
', '// Parse the full test input and return the answer as a string.
function solve(input) {
    return "";
}
', '// Parse the full test input and return the answer as a string.
export function Solve(input: string): string {
  return "";
}
'),
    ('archive-compression', '// Parse the full test input and return the answer as a string.
#include <string>

std::string Solve(const std::string& input) {
    return "";
}
', '// Parse the full test input and return the answer as a string.
function solve(input) {
    return "";
}
', '// Parse the full test input and return the answer as a string.
export function Solve(input: string): string {
  return "";
}
'),
    ('latency-budget', '// Parse the full test input and return the answer as a string.
#include <string>

std::string Solve(const std::string& input) {
    return "";
}
', '// Parse the full test input and return the answer as a string.
function solve(input) {
    return "";
}
', '// Parse the full test input and return the answer as a string.
export function Solve(input: string): string {
  return "";
}
'),
    ('priority-inbox', '// Parse the full test input and return the answer as a string.
#include <string>

std::string Solve(const std::string& input) {
    return "";
}
', '// Parse the full test input and return the answer as a string.
function solve(input) {
    return "";
}
', '// Parse the full test input and return the answer as a string.
export function Solve(input: string): string {
  return "";
}
'),
    ('unioned-accounts', '// Parse the full test input and return the answer as a string.
#include <string>

std::string Solve(const std::string& input) {
    return "";
}
', '// Parse the full test input and return the answer as a string.
function solve(input) {
    return "";
}
', '// Parse the full test input and return the answer as a string.
export function Solve(input: string): string {
  return "";
}
')
)
update tasks
set frontend_payload = frontend_payload
    || jsonb_build_object('languageVersions',
        coalesce(frontend_payload->'languageVersions', '{}'::jsonb)
        || '{"cpp":"20","javascript":"node24","typescript":"5.9.3"}'::jsonb)
    || jsonb_build_object('starterCodeByLanguage',
        coalesce(frontend_payload->'starterCodeByLanguage', '{}'::jsonb)
        || jsonb_build_object('cpp', starters.cpp, 'javascript', starters.javascript, 'typescript', starters.typescript)),
    updated_at = clock_timestamp()
from starters
where tasks.slug = starters.slug and frontend_payload->>'kind' = 'algorithmic';

update task_languages set is_enabled = false
where task_id in (select id from tasks where slug in ('signal-stabilizer', 'mirror-log', 'packet-window-median', 'adaptive-pricing', 'binary-signal-boundary', 'colony-perimeter', 'quantum-safe-lock', 'trade-ledger-balance', 'sensor-drift', 'cache-key-collisions', 'courier-routes', 'build-pipeline-order', 'archive-compression', 'latency-budget', 'priority-inbox', 'unioned-accounts'))
    and language in ('cpp', 'javascript', 'typescript')
    and (language, language_version) not in (('cpp','20'),('javascript','node24'),('typescript','5.9.3'));

insert into task_languages (id, task_id, language, language_version, is_enabled, created_at)
select gen_random_uuid(), tasks.id, runtimes.language, runtimes.version, true, clock_timestamp()
from tasks cross join (values ('cpp','20'),('javascript','node24'),('typescript','5.9.3')) as runtimes(language, version)
where tasks.slug in ('signal-stabilizer', 'mirror-log', 'packet-window-median', 'adaptive-pricing', 'binary-signal-boundary', 'colony-perimeter', 'quantum-safe-lock', 'trade-ledger-balance', 'sensor-drift', 'cache-key-collisions', 'courier-routes', 'build-pipeline-order', 'archive-compression', 'latency-budget', 'priority-inbox', 'unioned-accounts') and frontend_payload->>'kind' = 'algorithmic'
on conflict (task_id, language, language_version) do update set is_enabled = true;
