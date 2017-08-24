package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var GLOBAL_FLAGS *cliFlags

type cliFlags struct {
	sourceDBUsername string
	sourceDBPassword string
	sourceDBHost     string
	sourceDBPort     int
	sourceDBName     string

	targetDBUsername string
	targetDBPassword string
	targetDBHost     string
	targetDBPort     int
	targetDBName     string

	dryRun bool
	debug  bool
	user   string
}

func parseFlags() *cliFlags {
	flags := &cliFlags{}
	flag.StringVar(&flags.sourceDBUsername, "sourcedbusername", "", "The username to connect to the source db")
	flag.StringVar(&flags.sourceDBPassword, "sourcedbpass", "", "The pass to connect to the source db")
	flag.StringVar(&flags.sourceDBHost, "sourcedbhost", "", "The host of the source db")
	flag.IntVar(&flags.sourceDBPort, "sourcedbport", 3306, "The port of the source db")
	flag.StringVar(&flags.sourceDBName, "sourcedbname", "", "The name of the source db")

	flag.StringVar(&flags.targetDBUsername, "targetdbusername", "", "The username to connect to the target db")
	flag.StringVar(&flags.targetDBPassword, "targetdbpass", "", "The pass to connect to the target db")
	flag.StringVar(&flags.targetDBHost, "targetdbhost", "", "The host of the target db")
	flag.IntVar(&flags.targetDBPort, "targetdbport", 3306, "The port of the target db")
	flag.StringVar(&flags.targetDBName, "targetdbname", "", "The name of the target db")

	flag.StringVar(&flags.user, "user", "", "Limit the migration to this user shares")
	flag.BoolVar(&flags.dryRun, "dryrun", false, "Execute logic without commiting changes to the databases")
	flag.BoolVar(&flags.debug, "debug", false, "Shows debugging info")

	flag.Parse()
	GLOBAL_FLAGS = flags
	return flags
}

type shareInfo8 struct {
	ID          int64          `db:"id"`
	ShareType   int            `db:"share_type"`
	ShareWith   sql.NullString `db:"share_with"`
	UIDOwner    string         `db:"uid_owner"`
	Parent      sql.NullInt64  `db:"parent"`
	ItemType    sql.NullString `db:"item_type"`
	ItemSource  sql.NullString `db:"item_source"`
	ItemTarget  sql.NullString `db:"item_target"`
	FileSource  sql.NullInt64  `db:"file_source"`
	FileTarget  sql.NullString `db:"file_target"`
	Permissions string         `db:"permissions"`
	STime       int            `db:"stime"`
	Accepted    int            `db:"accepted"`
	Expiration  time.Time      `db:"expiration"`
	Token       sql.NullString `db:"token"`
	MailSend    int            `db:"mail_send"`
}

type shareInfo9 struct {
	shareInfo8
	UIDInitiator string `db:"uid_initiator"`
}

type sqlDriver struct {
	db *sqlx.DB
}

func newSQLDriver(dbUsername, dbPassword, dbHost string, dbPort int, dbName string) (*sqlDriver, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbUsername, dbPassword, dbHost, dbPort, dbName)
	d, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &sqlDriver{db: d}, nil
}

func (d *sqlDriver) getAllSharesFrom8(limitToUser string) ([]shareInfo8, error) {
	var entries []shareInfo8
	selectStmt := "SELECT * from oc_share ORDER BY id;"
	if limitToUser != "" {
		selectStmt = fmt.Sprintf("SELECT * from oc_share where uid_owner='%s' ORDER BY id;", limitToUser)
	}
	err := d.db.Select(&entries, selectStmt)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (d *sqlDriver) insertShareTo9(s *shareInfo8) error {
	shareInfo := &shareInfo9{*s, s.UIDOwner}
	fmt.Printf("INSERT INTO oc_share(share_type,share_with,uid_owner,parent,item_type,item_source,item_target,file_source,file_target,permissions,stime,accepted,expiration,token,mail_send) VALUES (%d,%s,%s,%d,%s,%s,%s,%d,%s,%s,%d,%d,%d,%s,%d,%s)",
		shareInfo.ShareType, shareInfo.ShareWith, shareInfo.UIDOwner, shareInfo.Parent, shareInfo.ItemType, shareInfo.ItemSource, shareInfo.ItemTarget, shareInfo.FileSource, shareInfo.FileTarget, shareInfo.Permissions, shareInfo.STime, shareInfo.Accepted, shareInfo.Expiration, shareInfo.Token, shareInfo.MailSend, shareInfo.UIDInitiator)
	if GLOBAL_FLAGS.dryRun {
		return nil
	}
	/*
		query := "UPDATE oc_share SET item_source=?,item_target=?,file_source=?,file_target=? WHERE id=?"
		stmt, err := d.db.Prepare(query)
		if err != nil {
			return err
		}
		defer stmt.Close()
		result, err := stmt.Exec(fmt.Sprintf("%d", versionsMeta.Inode), "/"+fmt.Sprintf("%d", versionsMeta.Inode), versionsMeta.Inode, "/"+path.Base(versionsMeta.Path), shareInfo.ID)
		if err != nil {
			return err
		}
		numRowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if numRowsAffected == 0 || numRowsAffected > 1 {
			return fmt.Errorf("Cannot updated share because share id %d does not exists anymore", shareInfo.ID)
		}
		return nil
	*/
	return nil
}
func main() {
	flags := parseFlags()

	sourceSQLDriver, err := newSQLDriver(flags.sourceDBUsername, flags.sourceDBPassword, flags.sourceDBHost, flags.sourceDBPort, flags.sourceDBName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	targetSQLDriver, err := newSQLDriver(flags.targetDBUsername, flags.targetDBPassword, flags.targetDBHost, flags.targetDBPort, flags.targetDBName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	shares, err := sourceSQLDriver.getAllSharesFrom8(flags.user)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot get all shares because ", err)
		os.Exit(1)
	}
	if len(shares) == 0 {
		fmt.Fprintln(os.Stderr, "oc_share table does not contain any shares")
		os.Exit(1)
	}

	const maxConcurrency = 20 // for example
	var throttle = make(chan int, maxConcurrency)

	var wg sync.WaitGroup
	for _, s := range shares {
		throttle <- 1 // whatever number
		wg.Add(1)

		go func(d *sqlDriver, s shareInfo8, wg *sync.WaitGroup, throttle chan int) {
			defer wg.Done()
			defer func() {
				<-throttle
			}()

			err = d.insertShareTo9(&s)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
		}(targetSQLDriver, s, &wg, throttle)
	}
	wg.Wait()
	fmt.Printf("Sucess. Dry run: %t\n", GLOBAL_FLAGS.dryRun)
	os.Exit(0)
}
