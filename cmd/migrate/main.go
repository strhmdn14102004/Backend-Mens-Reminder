package main

import (
	"log"
	"os"

	"backend_mens/internal/db"

	_ "github.com/lib/pq"
)

func main() {
	sqlBytes, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		log.Fatal(err)
	}

	d, err := db.Open(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	_, err = d.Exec(string(sqlBytes))
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Migration OK")
}
