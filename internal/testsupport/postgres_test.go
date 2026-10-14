package testsupport

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestRequirePostgresSkipsWithoutDSN(t *testing.T) {
	t.Setenv("BACKEND_REQUIRE_POSTGRES_TESTS", "")
	t.Setenv("BACKEND_TEST_DATABASE_URL", "")
	t.Setenv("TEST_DATABASE_URL", "")
	RequirePostgres(t)
}

func TestPostgresDSNRequiresConfiguration(t *testing.T) {
	if os.Getenv("CODEFIGHT_TEST_REQUIRED_POSTGRES_CHILD") == "1" {
		PostgresDSN(t)
		return
	}
	t.Setenv("CODEFIGHT_TEST_REQUIRED_POSTGRES_CHILD", "1")
	t.Setenv("BACKEND_REQUIRE_POSTGRES_TESTS", "1")
	t.Setenv("BACKEND_TEST_DATABASE_URL", "")
	t.Setenv("TEST_DATABASE_URL", "")
	cmd := exec.Command(os.Args[0], "-test.run=^TestPostgresDSNRequiresConfiguration$")
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "PostgreSQL integration tests are required") {
		t.Fatalf("missing required DSN did not fail the test process: %v\n%s", err, output)
	}
}
