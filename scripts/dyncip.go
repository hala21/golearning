package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
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

	// 假如ip.txt为空，需要更新文件，并且判断是否有文件
	if result == nil {
		//空文件的时候新建ip.txt文件
		ioutil.WriteFile(stockIpFile, []byte(stockIP), 0777)
		// 判断数据库和防火墙的IP是否相同，不同的话，需要修改防火墙IP地址
		sshClient, err := sshConn(ssh_username, ssh_password, ssh_server, ssh_port)
		if err != nil {
			logger.Fatal("连接防火墙失败")
		}
		session, err := sshClient.NewSession()
		if err != nil {
			logger.Fatal("不能创建防火墙ssh session")
		}
		defer session.Close()
		var sessionErr, out bytes.Buffer
		session.Stdout = &out
		session.Stderr = &sessionErr
		session.Run("show configuration security zones security-zone untrust address-book address xiyanghzc")
		logger.Fatal((sessionErr).String())
		ip := out.String()
		// 比较IP字符串
		if !strings.Contains(ip, stockIP) {
			session.Run("configure")
			session.Run("set security zones security-zone untrust address-book address xiyanghzc " + stockIP + "/32")
			session.Run("commit and-quit")
			session.Run("exit")
			session.Wait()
			logger.Fatal((sessionErr).String())
		}

	} else {
		// 数据库和文件IP不匹配的时候
		if stockIP != string(result) {
			sshClient, err := sshConn(ssh_username, ssh_password, ssh_server, ssh_port)
			if err != nil {
				logger.Fatal("连接防火墙失败")
			}
			session, err := sshClient.NewSession()
			if err != nil {
				logger.Fatal("不能创建防火墙ssh session")
			}
			defer session.Close()

			// 防火墙配置
			var sessionErr, out bytes.Buffer
			session.Stdout = &out
			session.Stderr = &sessionErr

			session.Run("configure")
			session.Run("set security zones security-zone untrust address-book address xiyanghzc " + stockIP + "/32")
			session.Run("commit and-quit")
			session.Run("exit")
			session.Wait()
			logger.Fatal((sessionErr).String())
		}
	}

}
