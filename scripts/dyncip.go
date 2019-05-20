package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"
)

var (
	stockIP     string
	stockId     int
	stockIpFile string = "ip.txt"
)

func main() {

	sec, logger := getParameter("--warehouse", "warehouseid", "dyncip.ini", os.Args[0])
	oracle_sid := sec["oracle_sid"]
	stockId, _ = strconv.Atoi(sec["stock_id"])
	ssh_server := sec["ssh_server"]
	ssh_username := sec["ssh_username"]
	ssh_password := sec["ssh_password"]
	ssh_port, _ := strconv.Atoi(sec["ssh_port"])
	ctxt, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 打开数据库连接
	db, err := sql.Open("goracle", oracle_sid)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	rows := db.QueryRowContext(ctxt, "select ftp_server from ecmdta.sy_stock_param where stock_id = ?", stockId)
	rows.Scan(&stockIP)
	print(stockIP)

	f, err := os.Open(stockIpFile)
	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create(stockIpFile)
			if err != nil {
				logger.Fatal(err)
			}
			f = file
			defer file.Close()
		}
	}

	defer f.Close()

	result, err := ioutil.ReadAll(f)
	if err != nil {
		logger.Fatal(err)
	}
	if result == nil {
		ioutil.WriteFile(stockIpFile, []byte(stockIP), 0777)
	} else {
		if stockIP != string(result) {
			sshClient, err := sshConn(ssh_username, ssh_password, ssh_server, ssh_port)
			session, err := sshClient.NewSession()
			if err != nil {
				logger.Fatal("can not creat  session")
			}
			session.Run()

			session.Run()

		}
	}

}
