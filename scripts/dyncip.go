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
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	stockIpFile = "ip.txt"
	stockIP     = ""
)

func main() {

	sec := GetParameter("warehouse", "warehouseid", "dyncip.ini")
	oracleSid := sec["oracle_sid"]
	stockId, _ := strconv.Atoi(sec["stock_id"])
	stockName := sec["stock_name"]
	sshServer := sec["ssh_server"]
	sshUsername := sec["ssh_username"]
	sshPassword := sec["ssh_password"]
	sshPort, _ := strconv.Atoi(sec["ssh_port"])

	//日志文件
	var layoutISO = "2006-01-02"
	filename := "dyncip_" + strings.Split(os.Args[1], "=")[1]
	// 保存7天以内的日志文件
	sysType := runtime.GOOS
	if sysType == "linux" {
		exec.Command("find ./logs - name oracle_lock_release_*.log -ctime +14 |xargs rm -f ")
	}

	logfileName := filename + "_" + time.Now().Format(layoutISO) + ".log"
	f, err := os.OpenFile("logs/"+logfileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	logger := log.New(f, "", log.LstdFlags)

	ctxt, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// 打开数据库连接
	db, err := sql.Open("goracle", oracleSid)
	if err != nil {
		logger.Println(err)
		return
	}
	defer db.Close()

	err = db.QueryRowContext(ctxt, "select ftp_server from ecmdta.sy_stock_param where stock_id = :1", stockId).Scan(&stockIP)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Println(stockIP)

	if stockIP == "" {
		logger.Fatal("数据库未能查询到数据，仓库id: %s,%s", stockId, stockName)
		return
	}

	f, err = os.Open(stockIpFile)
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
		sshClient, err := sshConn(sshServer, sshUsername, sshPassword, sshPort)
		if err != nil {
			logger.Fatal("不能连接到设备：%s", sshServer)
		}
		session, err := sshClient.NewSession()
		if err != nil {
			logger.Fatal("不能创建session: %s", sshServer)
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

				commands := []string{
					"configure",
					"set security zones security-zone untrust address-book address xiyanghzc " + stockIP + "/32",
					"commit and-quit",
					"exit",
				}

				for _, cmd := range commands {
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
			err := ioutil.WriteFile(stockIpFile, []byte(stockIP), 0777)
			if err != nil {
				logger.Fatal(err)
			}
		}

	} else {
		//  判断数据库和文件IP是否一致，不一致就修改防火墙设置，再修改文件内容
		if stockIP != fileIP {
			sshClient, err := sshConn(sshServer, sshUsername, sshPassword, sshPort)
			if err != nil {
				logger.Fatal("不能连接到设备：%s", sshServer)
			}
			session, err := sshClient.NewSession()
			if err != nil {
				logger.Fatal("不能创建session: %s", sshServer)
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

			commands := []string{
				"configure",
				"set security zones security-zone untrust address-book address xiyanghzc " + fileIP + "/32",
				"commit and-quit",
				"exit",
			}

			for _, cmd := range commands {
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
			// 更新数据库oracle TCP白名单
			oracleWhilelist := `insert into DBCTRL.TRUSTED_IPS 
				select distinct IPADDRESS, sysdate  from dbctrl.user_access_log_his 
				where IPADDRESS is not null and logon_time > trunc(sysdate)
                and ipaddress not in (select sourceip from DBCTRL.TRUSTED_IPS);`
			stmt, err := db.PrepareContext(ctxt, oracleWhilelist)
			if err != nil {
				logger.Fatal(err)
			}
			_, err = stmt.ExecContext(ctxt)
			if err != nil {
				logger.Fatal(err)
			}
			// 删除老的IP记录
			oracleWhilelistRecord := `DELETE DBCTRL.TRUSTED_IPS where sourceip = :1 `
			stmtDel, err := db.PrepareContext(ctxt, oracleWhilelistRecord)
			if err != nil {
				logger.Fatal(err)
			}
			_, err = stmtDel.ExecContext(ctxt, fileIP)
			if err != nil {
				logger.Fatal(err)
			}
			// 删除Linux TCP 监控

			sessionDb, err := sshClient.NewSession()
			if err != nil {
				logger.Fatal(err)
			}
			cmd := "cd /usr/local/scripts && sed -i `s/" + fileIP + "//g` trustip && echo " + stockIP + " >> trustip && echo > excepip.txt"
			err = sessionDb.Run(cmd)
			if err != nil {
				logger.Fatal(err)
			}
		}
	}

}
