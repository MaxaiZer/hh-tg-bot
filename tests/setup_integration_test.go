package tests

import (
	"context"
	"github.com/maxaizer/hh-parser/internal/config"
	"github.com/maxaizer/hh-parser/internal/repositories"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
)

var dbCtx *repositories.DbContext

func upEnvironment() {

	os.Setenv("DB_CONNECTION_STRING", "testdatabase.db")
	cfg := config.Get()

	var err error
	dbCtx, err = repositories.NewDbContext(cfg.DB.ConnectionString)
	if err != nil {
		log.Fatalf("could not create db context: %s", err)
	}

	err = dbCtx.Migrate()
	if err != nil {
		log.Fatalf("could not migrate db: %s", err)
	}

	searches := repositories.NewSearchRepository(dbCtx.DB)
	err = searches.Add(context.Background(), *search)
	if err != nil {
		log.Fatalf("could not add search: %s", err)
	}
	search.ID = 1
}

func downEnvironment() {
	_ = dbCtx.Close()
	_ = os.Remove("testdatabase.db")
}

func TestMain(m *testing.M) {

	err := os.Chdir("../") //project root to resolve correctly relative paths in code
	if err != nil {
		log.Fatal(err)
	}

	upEnvironment()

	code := m.Run()

	downEnvironment()

	os.Exit(code)
}
