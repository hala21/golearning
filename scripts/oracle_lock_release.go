package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "gopkg.in/goracle.v2"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func main() {
	// 获取参数
	sec := GetParameter("dbserver", "数据库服务器", "oracle_lock_check.ini")
	oracleSid := sec["oracle_sid"]
	sshPort := sec["server_ssh_port"]

	//日志文件
	var layoutISO = "2006-01-02"
	filename := "oracle_lock_release_" + strings.Split(os.Args[1], "=")[1]
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
	logRec := log.New(f, "check：", log.LstdFlags)

	// 打开数据库连接
	dbOracle, err := sql.Open("goracle", oracleSid)
	if err != nil {
		logRec.Fatal(err)
	}
	defer dbOracle.Close()
	dbOracle.SetConnMaxLifetime(time.Duration(5 * time.Minute))
	dbOracle.SetMaxOpenConns(100)
	dbOracle.SetMaxIdleConns(10)

	sql_lock_check := `select s1.username ||' ' || s1.machine || 
    ' ( SID=' || s1.sid ||' serial#='||s1.serial#||  ' )  is blocking ' || s2.username ||' '|| s2.machine || ' ( SID=' || s2.sid || ' ) ' AS blocking_status 
    from v$lock l1, v$session s1, v$lock l2, v$session s2
    where s1.sid = l1.sid   and s2.sid = l2.sid   and l1.BLOCK = 1   and l2.request > 0   and l1.id1 = l2.id1   and l2.id2 = l2.id2
    `
	sql_lock_sqltext := `select s.username ||' ' || s.sid ||' ' ||  s.status ||' ' || t.sql_text from v$session s, v$sqltext_with_newlines t
	where DECODE (s.sql_address, '00', s.prev_sql_addr, s.sql_address) = t.address 	and DECODE (s.sql_hash_value, 0, s.prev_hash_value, s.sql_hash_value) = t.hash_value
	and s.sid = :1 order by t.piece `
	//killSql := "alter system kill session ':1'"

	ctxt, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	rowsLock, err := dbOracle.QueryContext(ctxt, sql_lock_check)
	if err != nil {
		logRec.Println(err)
	}
	defer rowsLock.Close()

	var messages []string
	for rowsLock.Next() {
		message := ""
		err := rowsLock.Scan(&message)
		checkErr(err)
		messages = append(messages, message)
	}

	// 没有lock 就返回，结束运行
	if len(messages) == 0 {
		return
	}

	// 写入日志文件记录
	logRec.Println(messages)

	// 测试语句
	// messages = []string{"ECMDTA dcjob1asp02 ( SID=8345 serial#=33553 )  is blocking ECMDTA dcjob1asp11 ( SID=50 )", "ECMDTA dcjob1asp02 ( SID=8345 serial#=33553 )  is blocking ECMDTA dcjob1asp11 ( SID=50 )"}

	// jobsSidOrigin为了获取锁语句的sql文本；jobsServerOrigin 目的是去重启tomcat；sidSerialOrigin目的是killsql语句
	var jobsServerOrigin, sidSerialOrigin []string
	for _, msg := range messages {
		tempStr := strings.Split(msg, " ")
		jobServer := tempStr[1]
		jobsServerOrigin = append(jobsServerOrigin, jobServer)
		sidSerial := strings.Split(tempStr[3], "=")[1] + "," + strings.Split(tempStr[4], "=")[1]
		sidSerialOrigin = append(sidSerialOrigin, sidSerial)
	}

	// 去重服务器名
	var jobServers []string
	tmpServers := map[string]struct{}{}
	for _, server := range jobsServerOrigin {
		if _, ok := tmpServers[server]; !ok {
			tmpServers[server] = struct{}{}
			jobServers = append(jobServers, server)
		}
	}

	// 去重sidSerials
	var sidSerials []string
	tmpSidSerials := map[string]struct{}{}
	for _, sidSerial := range sidSerialOrigin {
		if _, ok := tmpSidSerials[sidSerial]; !ok {
			tmpSidSerials[sidSerial] = struct{}{}
			sidSerials = append(sidSerials, sidSerial)
		}
	}

	// 收集lock sql 语句
	var lockSqlTexts []string
	for _, sqlSid := range sidSerials {
		sid := strings.Split(sqlSid, ",")[0]
		lockSqlText := ""
		rows, err := dbOracle.QueryContext(ctxt, sql_lock_sqltext, sid)

		if err != nil {
			logRec.Printf("查询lock SQL失败： %v", err)
			fmt.Printf("查询lock SQL失败： %v", err)
		}
		if rows.Next() {
			err := rows.Scan(&lockSqlText)
			if err != nil {
				logRec.Printf("读取lock SQL失败： %v", err)
				fmt.Printf("读取lock SQL失败： %v", err)
			}
			lockSqlTexts = append(lockSqlTexts, lockSqlText)
		}

		//暂时写入日志文件
		logRec.Println(sid + lockSqlText)
	}
	fmt.Println(lockSqlTexts)

	// kill SQL语句
	for _, sqlSid := range sidSerials {
		result, err := dbOracle.Exec("alter system kill session '" + sqlSid + "' immediate")
		// 测试时检测到ORA-00030时正常现象，因为session不存在的话，会有告警提示。
		if err != nil {
			logRec.Println(err)
			fmt.Printf("执行killSQL失败：%v, %v", result, err)
		}
		//暂时写入日志文件
		logRec.Println("killed sql: " + sqlSid)
	}

	// 执行重启tomcat脚本
	for _, server := range jobServers {
		//cmdString := "ssh -p " + sshPort + " " + server + " sh /root/restart.sh "
		cmd := exec.CommandContext(ctxt, "ssh", "-p", sshPort, server, "sh", "/root/restart.sh")
		err := cmd.Start()
		logRec.Println(err)
		err = cmd.Wait()
		logRec.Println(err)
	}

	// 发出邮件
	to := "lizhiyong567@gmail.com"
	subject := "Oracle 数据库锁"
	body := `
		<html>
		<body>
		<h3> Oracle 数据库锁：</h3>` + "<div>JOB 信息：" + fmt.Sprintf(" %v ", jobServers) + "<br></br></div>" + "<p>数据库锁信息：" + strings.Join(lockSqlTexts, "\n\t") +
		`</p>
		</body>
		</html>`
	fmt.Println("send email")
	err = SendToMail(to, subject, body, "html")
	if err != nil {
		fmt.Println("Send mail error!")
		fmt.Println(err)
	} else {
		fmt.Println("Send mail success!")
	}
}
