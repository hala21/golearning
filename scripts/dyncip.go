package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"golang.org/x/crypto/ssh"
	_ "gopkg.in/goracle.v2"
	"io/ioutil"
	"log"
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

	sec, logger := getParameter("warehouse", "warehouseid", "dyncip.ini", strings.Split(os.Args[1], "=")[1])
	oracle_sid := sec["oracle_sid"]
	stockId, _ := strconv.Atoi(sec["stock_id"])
	stockName := sec["stock_name"]
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

	err = db.QueryRowContext(ctxt, "select ftp_server from ecmdta.sy_stock_param where stock_id = :1", stockId).Scan(&stockIP)
	if err != nil {
		logger.Fatal(err)
	}
	print(stockIP)
	logger.Println(stockIP)

	if stockIP == "" {
		logger.Fatal("数据库未能查询到数据，仓库id: %s,%s", stockId, stockName)
		return
	}

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

	//获取文件内容：IP地址
	result, err := ioutil.ReadAll(f)
	if err != nil {
		logger.Fatal(err)
	}
	fileIP := string(result)

	// 判断文件内容为空的话，检测防火墙的设置是否和数据库内容一致，不一致修改防火墙配置；
	// 一致不一致的情况都需要把现有IP写入文件
	if len(result) == 0 {
		//检测防火墙配置
		sshClient, err := sshConn(ssh_username, ssh_password, ssh_server, ssh_port)
		if err != nil {
			logger.Fatal("不能连接到设备：%s", ssh_server)
		}
		session, err := sshClient.NewSession()
		if err != nil {
			logger.Fatal("不能创建session: %s", ssh_server)
		}
		defer session.Close()

		var errOut, out bytes.Buffer
		session.Stderr = &errOut
		session.Stdout = &out

		_ = session.Run(fmt.Sprintf("show configuration security zones security-zone untrust address-book address %s", stockName))
		// 假如存在这个仓库名称的话，进行下一步操作，
		if out.Len() > 7 {
			fwIP := strings.Split(out.String(), "/")[0]
			//数据库里面的IP和防火墙内部的IP不一致的时候，需要修改防火墙配置
			if stockIP != fwIP {
				// Need pseudo terminal if we want to have an SSH session
				// similar to what you have when you use a SSH client
				modes := ssh.TerminalModes{
					ssh.ECHO:          0,     // disable echoing
					ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
					ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
				}

				// 未测试w值是否跟命令截断相关
				err = session.RequestPty("xterm", 80, 200, modes)
				if err != nil {
					logger.Fatalf("request for pseudo terminal failed: %s", err)
				}

				// StdinPipe for commands
				stdin, err := session.StdinPipe()
				if err != nil {
					log.Fatal(err)
				}

				var errOut, out bytes.Buffer
				session.Stderr = &errOut
				session.Stdout = &out

				//开启一个远程shell
				err = session.Shell()
				if err != nil {
					logger.Fatal(err)
				}

				comands := []string{
					"configure",
					"set security zones security-zone untrust address-book address xiyanghzc " + stockIP + "/32",
					"commit and-quit",
					"exit",
				}

				for _, cmd := range comands {
					_, err = fmt.Fprintf(stdin, "%s \n", cmd)
					if err != nil {
						logger.Fatal(err)
					}
				}
				err = session.Wait()
				if err != nil {
					logger.Fatal(err)
				}
			}

			//写入IP到文件
			ioutil.WriteFile(stockIpFile, []byte(stockIP), 0777)
		}

	} else {
		//  判断数据库和文件IP是否一致，不一致就修改防火墙设置，再修改文件内容
		if stockIP != fileIP {
			sshClient, err := sshConn(ssh_username, ssh_password, ssh_server, ssh_port)
			if err != nil {
				logger.Fatal("不能连接到设备：%s", ssh_server)
			}
			session, err := sshClient.NewSession()
			if err != nil {
				logger.Fatal("不能创建session: %s", ssh_server)
			}
			defer session.Close()

			// Need pseudo terminal if we want to have an SSH session
			// similar to what you have when you use a SSH client
			modes := ssh.TerminalModes{
				ssh.ECHO:          0,     // disable echoing
				ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
				ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
			}

			// 未测试w值是否跟命令截断相关
			err = session.RequestPty("xterm", 80, 200, modes)
			if err != nil {
				logger.Fatalf("request for pseudo terminal failed: %s", err)
			}

			// StdinPipe for commands
			stdin, err := session.StdinPipe()
			if err != nil {
				log.Fatal(err)
			}

			var errOut, out bytes.Buffer
			session.Stderr = &errOut
			session.Stdout = &out

			//开启一个远程shell
			err = session.Shell()
			if err != nil {
				logger.Fatal(err)
			}

			comands := []string{
				"configure",
				"set security zones security-zone untrust address-book address xiyanghzc " + fileIP + "/32",
				"commit and-quit",
				"exit",
			}

			for _, cmd := range comands {
				_, err = fmt.Fprintf(stdin, "%s \n", cmd)
				if err != nil {
					logger.Fatal(err)
				}
			}
			//stdin.Write([]byte("configure \n"))
			//stdin.Write([]byte("set security zones security-zone untrust address-book address xiyanghzc " + fileIP +"/32 \n"))
			//stdin.Write([]byte("commit and-quit \n"))
			//stdin.Write([]byte("exit \n"))

			err = session.Wait()
			if err != nil {
				logger.Fatal(err)
			}

			// 成功更新防火墙配置之后，再写入IP到文件
			err = ioutil.WriteFile(stockIpFile, []byte(stockIP), 0777)
			if err != nil {
				logger.Fatal(err)
			}
		}
	}

}
