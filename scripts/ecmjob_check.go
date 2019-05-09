package main

import (
	"database/sql"
	"fmt"
	_ "gopkg.in/goracle.v2"
)

func main() {
	db, err := sql.Open("goracle", "")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	rows, err := db.Query("select sysdate from dual")
	if err != nil {
		fmt.Println("Error running query")
		fmt.Println(err)
		return
	}
	defer rows.Close()

	var theDate string
	for rows.Next() {
		rows.Scan(&theDate)
	}

	fmt.Printf("The date is: %s\n", theDate)

}
