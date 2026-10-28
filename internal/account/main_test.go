package account_test

import (
	"os"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/testdb"
)

// TestMain serialises this package against the other database-backed test
// packages; both truncate the same tables.
func TestMain(m *testing.M) {
	os.Exit(testdb.Run(m))
}
