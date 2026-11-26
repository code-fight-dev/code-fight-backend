ALTER TABLE matches ADD COLUMN tournament_fixture_id uuid REFERENCES tournament_fixtures(id) ON DELETE SET NULL;
CREATE INDEX matches_tournament_fixture_idx ON matches(tournament_fixture_id) WHERE tournament_fixture_id IS NOT NULL;

UPDATE matches m SET tournament_fixture_id = f.id FROM tournament_fixtures f WHERE f.match_id = m.id;

-- Scheduling has recorded both IDs since fixtures were introduced. Recover
-- earlier attempts whose current match pointer was replaced before this migration.
UPDATE matches m SET tournament_fixture_id = f.id
FROM audit_events e JOIN tournament_fixtures f
  ON f.id::text = e.metadata->>'fixtureId' AND f.tournament_id = e.entity_id
WHERE e.action = 'tournament.match_scheduled' AND e.entity_type = 'tournament'
  AND m.id::text = e.metadata->>'matchId' AND m.tournament_fixture_id IS NULL;

-- Keep attempts attached when an older Backend schedules a replay during rollout.
CREATE FUNCTION record_fixture_match() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    UPDATE matches SET tournament_fixture_id = NEW.id WHERE id = NEW.match_id;
    RETURN NEW;
END $$;
CREATE TRIGGER tournament_fixture_match_history AFTER INSERT OR UPDATE OF match_id ON tournament_fixtures
FOR EACH ROW WHEN (NEW.match_id IS NOT NULL) EXECUTE FUNCTION record_fixture_match();
