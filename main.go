package main

import (
	"Cx_Mcdean_Backend/config"
	"Cx_Mcdean_Backend/db"
	"Cx_Mcdean_Backend/router"
	"fmt"
	"log"
)

func main() {
	config.Load()

	if _, err := db.Connect(); err != nil {
		log.Fatalf("connect db failed: %v", err)
	}

	r := router.Setup()
	addr := fmt.Sprintf(":%s", config.C.AppPort)
	log.Printf("listening on %s ...", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
