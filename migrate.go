package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
)

func migration(conn *sqlx.DB) {
	driver, err := mysql.WithInstance(conn.DB, &mysql.Config{})
	if err != nil {
		log.Fatal("connect: ", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "mysql", driver)
	if err != nil {
		log.Fatal("read migrations: ", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatal("apply migrations: ", err)
	}

	v, _, _ := m.Version()
	fmt.Println("Migrated to version " + strconv.Itoa(int(v)))
}
