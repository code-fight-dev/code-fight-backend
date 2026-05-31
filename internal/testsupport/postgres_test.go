package testsupport

import "testing"

func TestRequirePostgresSkipsWithoutDSN(t *testing.T) {
	t.Setenv("BACKEND_TEST_DATABASE_URL", "")
	t.Setenv("TEST_DATABASE_URL", "")
	RequirePostgres(t)
}
