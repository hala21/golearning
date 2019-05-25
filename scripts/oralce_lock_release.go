package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "gopkg.in/goracle.v2"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func main() {
	// 获取参数
	sec, logger := getParameter("dbserver", "数据库服务器", "oracle_lock_check.ini", strings.Split(os.Args[1], "=")[1])
	oracleSid := sec["oracle_sid"]
	sshPort, _ := strconv.Atoi(sec["server_ssh_port"])

	// 打开数据库连接
	dbOracle, err := sql.Open("goracle", oracleSid)
	checkErr(err)
	defer dbOracle.Close()

	dbOracle.SetMaxOpenConns(100)

	sql_lock_check := `select s1.username ||' ' || s1.machine || 
    ' ( SID=' || s1.sid ||' serial#='||s1.serial#||  ' )  is blocking ' || s2.username ||' '|| s2.machine || ' ( SID=' || s2.sid || ' ) ' AS blocking_status 
    from v$lock l1, v$session s1, v$lock l2, v$session s2
    where s1.sid = l1.sid   and s2.sid = l2.sid   and l1.BLOCK = 1   and l2.request > 0   and l1.id1 = l2.id1   and l2.id2 = l2.id2;
    `
	sql_lock_sqltext := `select s.username, s.sid, s.status, t.sql_text from v$session s, v$sqltext_with_newlines t
	where DECODE (s.sql_address, '00', s.prev_sql_addr, s.sql_address) = t.address 	and DECODE (s.sql_hash_value, 0, s.prev_hash_value, s.sql_hash_value) = t.hash_value
	and s.sid = :1 order by t.piece `
	killSql := "alter system kill session':1'"

	ctxt, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	rowsLock, err := dbOracle.QueryContext(ctxt, sql_lock_check)
	checkErr(err)
	defer rowsLock.Close()

	var messages []string
	for rowsLock.Next() {
		err := rowsLock.Scan(&messages)
		checkErr(err)
		messages = append(messages)
	}

	// 写入日志文件记录
	logger.Println(messages)

	// 测试语句
	messages = []string{"ECMDTA dcjob1asp02 ( SID=8345 serial#=33553 )  is blocking ECMDTA dcjob1asp11 ( SID=50 )", "ECMDTA dcjob1asp02 ( SID=8345 serial#=33553 )  is blocking ECMDTA dcjob1asp11 ( SID=50 )"}

	// jobsSidOrigin为了获取锁语句的sql文本；jobsServerOrigin 目的是去重启tomcat；sidSerialOrigin目的是killsql语句
	var jobsServerOrigin, sidSerialOrigin []string
	for _, msg := range messages {
		tempStr := strings.Split(msg, " ")
		jobServer := tempStr[2]
		jobsServerOrigin = append(jobsServerOrigin, jobServer)
		sidSerial := strings.Split(tempStr[4], "=")[1] + "," + strings.Split(tempStr[5], "=")[1]
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
		err := dbOracle.QueryRowContext(ctxt, sql_lock_sqltext, sid).Scan(&lockSqlText)
		if err != nil {
			logger.Printf("查询lock SQL失败： %v", err)
		}
		//暂时写入日志文件
		logger.Println(sid + lockSqlText)
		lockSqlTexts = append(lockSqlTexts, lockSqlText)
	}
	// kill SQL语句
	for _, sqlSid := range sidSerials {
		stmt, err := dbOracle.PrepareContext(ctxt, killSql)
		if err != nil {
			logger.Printf("预编译killSQL失败： %v", err)
		}
		_, err = stmt.Exec(sqlSid)
		if err != nil {
			logger.Println(err)
		}
		err = stmt.Close()
		checkErr(err)

		//暂时写入日志文件
		logger.Println(time.Now().Format("2016-01-02 15:04:05") + "killed sql" + sqlSid)
	}

	// 执行重启tomcat脚本
	for _, server := range jobServers {
		cmdString := "ssh -p " + string(sshPort) + " " + server + " sh /root/restart.sh "
		cmd := exec.CommandContext(ctxt, cmdString)
		err := cmd.Start()
		checkErr(err)
		err = cmd.Wait()
		checkErr(err)
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
