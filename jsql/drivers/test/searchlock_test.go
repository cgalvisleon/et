package test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/jsql"
)

// searchLockRows: the rows of the table of TestSearchIndexesLock, enough for a GIN build to last a while
const searchLockRows = 50000

/**
* serviceDB: A *jsql.DB of its own over the target, as another service (process) that starts against the same database:
* its own connection pool and session.
* @param t *testing.T, name string
* @return *jsql.DB
**/
func serviceDB(t *testing.T, name string) *jsql.DB {
	t.Helper()
	tg := findTarget("postgres")
	pg := tg.params.Connection.(*jsql.PgConection)
	pg.Id = name
	pg.AppName = name
	tg.params.ID = name
	db, err := jsql.ConnectTo(tg.params)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

/**
* startService: Defines the articles model in db (full_text over fields, in es and en) and starts it (Init), as a
* service does on start.
* @param t *testing.T, tg target, db *jsql.DB, fields []string
* @return error
**/
func startService(t *testing.T, tg target, db *jsql.DB, fields []string) error {
	model, err := db.Define(fullTextDefine(t, tg.schema, fields, []string{"en"}))
	if err != nil {
		return err
	}
	return model.Init()
}

/**
* TestSearchIndexesLock: Postgres only. (1) Three services (each its own *jsql.DB, pool and session) start at the same
* time over a table with rows and no search index: the lock of ApplyIndexes (pg_try_advisory_lock tried again) lets
* one build, the others find the indexes made and none fails; each index is there once and valid. (2) A CREATE INDEX
* CONCURRENTLY cancelled for real while it builds (pg_cancel_backend): that start fails and leaves the index invalid,
* and the next start finds it, drops it and builds it again.
**/
func TestSearchIndexesLock(t *testing.T) {
	tg := findTarget("postgres")
	db := connect(t, tg)
	fields := []string{"code", "name", "description"}

	// The table with its rows and without indexes, as a model that gets full_text after it has data
	envar.SetStr("DB_SEARCH_INDEXES", "off")
	err := startService(t, tg, db, fields)
	envar.SetStr("DB_SEARCH_INDEXES", "on")
	if err != nil {
		t.Fatal("init:", err)
	}
	fill := fmt.Sprintf(`INSERT INTO %s.articles (id, code, name, description, tags)
SELECT 'r' || i, 'C-' || i, 'bomba ' || md5(i::text), repeat(md5((i * 7)::text) || ' agua sumergible ', 20), ''
FROM generate_series(1, %d) AS i`, tg.schema, searchLockRows)
	if _, err := db.Sql(fill); err != nil {
		t.Fatal("rows:", err)
	}
	if got := pgFtsIndexes(t, db, tg.schema, "articles"); len(got) != 0 {
		t.Fatalf("indexes before the start: %v", got)
	}

	t.Run("services at once", func(t *testing.T) {
		const services = 3
		dbs := make([]*jsql.DB, services)
		for i := range services {
			dbs[i] = serviceDB(t, fmt.Sprintf("jsql_service_%d", i))
		}
		errs := make([]error, services)
		took := make([]time.Duration, services)
		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := range services {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				begin := time.Now()
				errs[i] = startService(t, tg, dbs[i], fields)
				took[i] = time.Since(begin)
			}()
		}
		close(start)
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				t.Errorf("service %d: %v", i, err)
			}
		}
		t.Logf("starts: %v", took)

		got := pgFtsIndexes(t, db, tg.schema, "articles")
		spanish, english := pgIndexWith(got, "'spanish'"), pgIndexWith(got, "'english'")
		if len(got) != 2 || spanish == "" || english == "" || !got[spanish].valid || !got[english].valid {
			t.Fatalf("one valid GIN per lang after the starts: %v", got)
		}
		if items, err := db.Sql("SELECT count(*) AS n FROM pg_locks WHERE locktype = 'advisory'"); err != nil || items.Int(0, "n") != 0 {
			t.Errorf("advisory locks left: %v %v", items, err)
		}
	})

	t.Run("interrupted build", func(t *testing.T) {
		before := pgFtsIndexes(t, db, tg.schema, "articles")
		spanish := pgIndexWith(before, "'spanish'")
		if _, err := db.Sql(fmt.Sprintf("DROP INDEX %s.%s", tg.schema, spanish)); err != nil {
			t.Fatal("drop:", err)
		}

		service := serviceDB(t, "jsql_service_cancelled")
		done := make(chan error, 1)
		go func() { done <- startService(t, tg, service, fields) }()

		// The session that builds the index, cancelled while it is in the build
		pid, phase := 0, ""
		for range 1000 {
			items, err := db.Sql("SELECT pid, phase FROM pg_stat_progress_create_index WHERE relid = $1::regclass", tg.schema+".articles")
			if err == nil && items.Count > 0 && items.Str(0, "phase") != "initializing" {
				pid, phase = items.Int(0, "pid"), items.Str(0, "phase")
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if pid == 0 {
			<-done
			t.Fatal("the build was not seen in pg_stat_progress_create_index (too fast for the rows)")
		}
		if _, err := db.Sql("SELECT pg_cancel_backend($1)", pid); err != nil {
			t.Fatal("cancel:", err)
		}
		err := <-done
		if err == nil {
			t.Fatal("the start whose build was cancelled did not fail")
		}
		t.Logf("cancelled in phase %q: %v", phase, err)

		left := pgFtsIndexes(t, db, tg.schema, "articles")
		invalid, ok := left[spanish]
		if !ok || invalid.valid {
			t.Fatalf("the cancelled build did not leave the index invalid: %v", left)
		}

		// The next start finds it invalid and builds it again
		if err := startService(t, tg, serviceDB(t, "jsql_service_next"), fields); err != nil {
			t.Fatal("next start:", err)
		}
		again := pgFtsIndexes(t, db, tg.schema, "articles")
		if idx, ok := again[spanish]; !ok || !idx.valid || idx.oid == invalid.oid || len(again) != 2 {
			t.Errorf("the invalid index was not built again: %v (invalid oid %d)", again, invalid.oid)
		}
	})
}
