package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/Unknwon/goconfig"
	_ "gopkg.in/goracle.v2"
	"log"
)

type job_info struct {
	serverIp string
	jobNmae  string
}

func main() {

	//get command parameter
	server := flag.String("dbserver", "", "server_info")
	flag.Parse()
	if flag.NFlag() == 0 || len(*server) == 0 {
		log.Println("Please input parameter")
		return
	}

	// get config file parameter
	conf, err := goconfig.LoadConfigFile("jobcheck.ini")
	if err != nil {
		log.Fatal(err)
		return
	}
	sec, err := conf.GetSection(*server)
	if err != nil {
		log.Println("Please input correct parameter")
		return
	}

	oracle_sid := sec["oracle_sid"]
	//server_ssh_port , _ := strconv.Atoi(sec["server_ssh_port"])
	fmt.Println(oracle_sid)

	db, err := sql.Open("goracle", oracle_sid)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	sql_1 := `select b.server_ip,a.job_name from sy_jobs a 
	inner join sy_job_server b on b.server_name=a.server_name 
	where a.next_run_date+6/1440 <sysdate and a.enabled=1 `

	sql_2 := ` select c.server_ip,b.job_name from sy_job_queue a
        inner join sy_jobs b on b.job_ukid=a.job_ukid
        inner join sy_job_server c on c.server_name=b.server_name
      where  ROUND(TO_NUMBER(sysdate - a.run_time) * 24 * 60)>=nvl(b.over_time_warn,30)`

	var theDate []job_info
	rows_1, err := db.Query(sql_1)
	if err != nil {
		fmt.Println("Error running query")
		fmt.Println(err)
		return
	}
	defer rows_1.Close()

	for rows_1.Next() {
		rows_1.Scan(&theDate)
	}

	rows_2, err := db.Query(sql_2)

	if err != nil {
		fmt.Println("Error running query")
		fmt.Println(err)
		return
	}
	defer rows_2.Close()

	for rows_2.Next() {
		rows_2.Scan(&theDate)
	}

	fmt.Printf("The date is: %s\n", theDate)

}
