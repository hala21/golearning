package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "gopkg.in/goracle.v2"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type jobInfo struct {
	serverIp string
	jobName  string
}

var (
	Timeout = 60 * time.Second
)

func main() {
	// 获取参数
	sec := GetParameter("dbserver", "数据库服务器", "jobcheck.ini")
	oracleSid := sec["oracle_sid"]
	sshPort, _ := strconv.Atoi(sec["server_ssh_port"])
	mysqlSid := sec["mysql_sid"]

	//日志文件
	var layoutISO = "2006-01-02"
	filename := "ecmjob_check__" + strings.Split(os.Args[1], "=")[1]
	// 保存7天以内的日志文件
	sysType := runtime.GOOS
	if sysType == "linux" {
		exec.Command("find ./logs - name ecmjob_check_*.log -ctime +14 |xargs rm -f ")
	}

	logfileName := filename + "_" + time.Now().Format(layoutISO) + ".log"
	f, err := os.OpenFile("logs/"+logfileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	logger := log.New(f, "check：", log.LstdFlags)

	// 打开数据库连接
	dbOracle, err := sql.Open("goracle", oracleSid)
	if err != nil {
		logger.Println(err)
		callPhone("param", 187571378227, "")
		return
	}
	defer dbOracle.Close()

	dbOracle.SetMaxOpenConns(100)

	sql_1 := `select b.server_ip,a.job_name from ecmdta.sy_jobs a
			inner join ecmdta.sy_job_server b on b.server_name=a.server_name
			where a.next_run_date+6/1440 <sysdate and a.enabled=1 `

	sql_2 := ` select c.server_ip,b.job_name  from sy_job_queue a 
     inner join sy_jobs b on b.job_ukid=a.job_ukid
     inner join sy_job_server c on c.server_name=b.server_name
     where ROUND(TO_NUMBER(sysdate - a.run_time) * 24 * 60)>=nvl(b.over_time_warn,30) and c.start_job=1-- and b.mobile_list is not null
     group by c.server_ip,b.job_name;`

	// 获取服务器地址，分两种情况，一：服务延迟太多，二：服务预期运行时间比现在晚
	//sql_1_test := "select c.server_ip,b.job_name from ecmdta.sy_job_queue a inner join ecmdta.sy_jobs b on b.job_ukid=a.job_ukid inner join ecmdta.sy_job_server c on c.server_name=b.server_name where ROUND(TO_NUMBER(sysdate - a.run_time) * 24 * 60)>=15 group by c.server_ip,b.job_name"
	//sql_2_test := "select c.server_ip,b.job_name from ecmdta.sy_job_queue a inner join ecmdta.sy_jobs b on b.job_ukid=a.job_ukid inner join ecmdta.sy_job_server c on c.server_name=b.server_name where ROUND(TO_NUMBER(sysdate - a.run_time) * 24 * 60)>=15 group by c.server_ip,b.job_name"

	var theData []jobInfo
	var serverIp, jobName string = "", ""
	rows_1, err := dbOracle.Query(sql_1)
	if err != nil {
		logger.Println("Error running query")
		logger.Println(err)
		return
	}
	defer rows_1.Close()

	for rows_1.Next() {
		_ = rows_1.Scan(&serverIp, &jobName)
		theData = append(theData, jobInfo{serverIp, jobName})
	}

	rows_2, err := dbOracle.Query(sql_2)
	if err != nil {
		logger.Println("Error running query")
		logger.Println(err)
		return
	}
	defer rows_2.Close()

	for rows_2.Next() {
		_ = rows_2.Scan(&serverIp, &jobName)
		theData = append(theData, jobInfo{serverIp, jobName})
	}

	// 写入日志文件记录
	logger.Println(theData)

	// 服务器IP去重
	var serverIps []string
	tmpServerIps := map[string]struct{}{}
	for _, value := range theData {
		ip := value.serverIp
		if _, ok := tmpServerIps[ip]; !ok {
			tmpServerIps[ip] = struct{}{}
			serverIps = append(serverIps, ip)
		}
	}

	ctxt, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	// 检查数据库情况
	if len(serverIps) > 0 {
		// 假如有异常的任务，需要和之前的重启记录比对，比对是都是相隔10分钟左右，说明重启还未解决之前的问题，
		// 假如上次重启再10分钟前，且在节假日，需要 call电话

		dbMysql, err := sql.Open("mysql", mysqlSid)
		checkErr(err)

		timeNow := time.Now()

		for _, ip := range serverIps {
			rows, err := dbMysql.QueryContext(ctxt, "select jobip,jobname,reboottime from reboot_record where jobip = ? order by reboottime limit 1 ", ip)
			checkErr(err)
			for rows.Next() {
				var jobIp, jobName, reTime string
				err := rows.Scan(&jobIp, &jobName, &reTime)
				checkErr(err)
				rebootTime, err := time.Parse("2006-01-02 15:04:05", reTime)
				checkErr(err)
				if timeNow.Sub(rebootTime) < 12 && timeNow.Sub(rebootTime) > 5 {
					if standbyUnattendedTime() {
					}
					callPhone("parame", 18757138227, "")
				} else {

				}

			}

		}

		// SSH 执行重启脚本
		for _, ip := range serverIps {
			var buf bytes.Buffer
			//exec.Command("ssh -p "+string(server_ssh_port)+" " +ip+" sh /webapp/scripts/restart.sh")
			//cmdString := "ssh -p " + string(sshPort) + " " + ip + " tail -30 /webapp/tomcat1/logs/catalina.out ;sh /webapp/scripts/restart.sh"
			cmd := exec.CommandContext(ctxt, "ssh", "-p", string(sshPort), "ip", "tail -30 /webapp/tomcat1/logs/catalina.out ;sh /root/restart.sh")
			cmd.Stdout = &buf
			cmd.Stderr = &buf
			err := cmd.Start()
			checkErr(err)
			err = cmd.Wait()
			checkErr(err)
			fmt.Println(buf.Bytes())
			logger.Println(buf.Bytes())

			//插入数据
			stmt, err := dbMysql.Prepare("INSERT INTO reboot_record SET jobname = ?, jobip = ?, reboottime = ?, status = ?")
			checkErr(err)
			// 一个IP可能对应多个job名称，需要分开写入数据库进行记录
			for _, val := range theData {
				if val.serverIp == ip {
					_, err := stmt.Exec(val.jobName, ip, timeNow.Format("2006-01-02 15:04:05"), "success")
					checkErr(err)
				}
			}

		}

	} else {
		// 假如没有异常job，则中断后续处理
		return
	}

	// 发出邮件
	to := "lizhiyong567@gmail.com"
	subject := "后台任务处理记录"
	body := `
		<html>
		<body>
		<h3> 异常后台任务：</h3>` + "<div>JOB 信息：" + fmt.Sprintf(" %v ", serverIps) + "<br></br></div></body></html>"

	logger.Println("send email")
	err = SendToMail(to, subject, body, "html")
	if err != nil {
		logger.Println("Send mail error!")
		logger.Println(err)
	} else {
		fmt.Println("Send mail success!")
	}

}
