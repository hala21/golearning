package main

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"github.com/Unknwon/goconfig"
	_ "gopkg.in/goracle.v2"
	"log"
	"net/smtp"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type job_info struct {
	serverIp string
	jobName  string
}

var (
	Timeout = 60 * time.Second
)

func SendToMail(user, password, host, to, subject, body, mailtype string) error {
	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	var content_type string
	if mailtype == "html" {
		content_type = "Content-Type: text/" + mailtype + "; charset=UTF-8"
	} else {
		content_type = "Content-Type: text/plain" + "; charset=UTF-8"
	}
	msg := []byte("To: " + to + "\r\nFrom: " + user + "\r\nSubject: " + subject + " \r\n" + content_type + "\r\n\r\n" + body)
	send_to := strings.Split(to, ";")
	err := smtp.SendMail(host, auth, user, send_to, msg)
	return err
}

func main() {
	// 获取参数
	server := flag.String("dbserver", "", "server_info")
	flag.Parse()
	if flag.NFlag() == 0 || len(*server) == 0 {
		log.Println("Please input parameter")
		return
	}

	// 参数文件名
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

	//日志文件
	var layoutISO = "2006-01-02"
	// 保存7天以内的日志文件
	sysType := runtime.GOOS
	if sysType == "linux" {
		exec.Command("find ./ - name *jobcheck*.log -ctime +6 |xargs rm -f ")
	}

	logfileName := *server + "_jobcheck_" + time.Now().Format(layoutISO) + ".log"
	f, err := os.OpenFile(logfileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	var logger = log.New(f, "check：", log.LstdFlags)

	oracle_sid := sec["oracle_sid"]
	server_ssh_port, _ := strconv.Atoi(sec["server_ssh_port"])

	// 打开数据库连接
	db, err := sql.Open("goracle", oracle_sid)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	db.SetMaxOpenConns(100)

	/*
			sql_1 := `select b.server_ip,a.job_name from ecmdtasy_jobs a
			inner join ecmdta.sy_job_server b on b.server_name=a.server_name
			where a.next_run_date+6/1440 <sysdate and a.enabled=1 `

			sql_2 := ` select c.server_ip,b.job_name from ecmdta.sy_job_queue a
		        inner join ecmdta.sy_jobs b on b.job_ukid=a.job_ukid
		        inner join ecmdta.sy_job_server c on c.server_name=b.server_name
		      where  ROUND(TO_NUMBER(sysdate - a.run_time) * 24 * 60)>=nvl(b.over_time_warn,30)`
	*/

	sql_lock_check := `select s1.username ||' ' || s1.machine || 
    ' ( SID=' || s1.sid ||' serial#='||s1.serial#||  ' )  is blocking ' || s2.username ||' '|| s2.machine || ' ( SID=' || s2.sid || ' ) ' AS blocking_status 
    from v$lock l1, v$session s1, v$lock l2, v$session s2
    where s1.sid = l1.sid   and s2.sid = l2.sid   and l1.BLOCK = 1   and l2.request > 0   and l1.id1 = l2.id1   and l2.id2 = l2.id2;
    `
	sql_lock_sqltext := `select s.username, s.sid, s.status, t.sql_text from v$session s, v$sqltext_with_newlines t
	where DECODE (s.sql_address, '00', s.prev_sql_addr, s.sql_address) = t.address 	and DECODE (s.sql_hash_value, 0, s.prev_hash_value, s.sql_hash_value) = t.hash_value
	and s.sid = ? order by t.piece `

	// 获取服务器地址，分两种情况，一：服务延迟太多，二：服务预期运行时间比现在晚
	sql_1_test := "select c.server_ip,b.job_name from ecmdta.sy_job_queue a inner join ecmdta.sy_jobs b on b.job_ukid=a.job_ukid inner join ecmdta.sy_job_server c on c.server_name=b.server_name where ROUND(TO_NUMBER(sysdate - a.run_time) * 24 * 60)>=15 group by c.server_ip,b.job_name"
	sql_2_test := "select c.server_ip,b.job_name from ecmdta.sy_job_queue a inner join ecmdta.sy_jobs b on b.job_ukid=a.job_ukid inner join ecmdta.sy_job_server c on c.server_name=b.server_name where ROUND(TO_NUMBER(sysdate - a.run_time) * 24 * 60)>=15 group by c.server_ip,b.job_name"

	var theData []job_info
	var serverIp, jobName string = "", ""
	rows_1, err := db.Query(sql_1_test)
	if err != nil {
		fmt.Println("Error running query")
		fmt.Println(err)
		return
	}
	defer rows_1.Close()

	for rows_1.Next() {
		rows_1.Scan(&serverIp, &jobName)
		theData = append(theData, job_info{serverIp, jobName})
	}

	rows_2, err := db.Query(sql_2_test)
	if err != nil {
		fmt.Println("Error running query")
		fmt.Println(err)
		return
	}
	defer rows_2.Close()

	for rows_2.Next() {
		rows_2.Scan(&serverIp, &jobName)
		theData = append(theData, job_info{serverIp, jobName})
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
	var sql_lock_txt string = " "
	if len(serverIps) > 0 {
		//检查是否有数据库的情况，假如有锁，读取任意一行数据，获得SQL文件，假如锁很多，很那收集到所有的信息，所以选取其中一条所情况
		rows_lock := db.QueryRowContext(ctxt, sql_lock_check)
		var sidString = ""
		rows_lock.Scan(&sidString)
		fmt.Println(" oracle sql id: " + sidString)

		//假如有锁的情况，根据SQD 获取SQL文本信息。
		sidString = "ECMDTA dcjob1asp02 ( SID=8345 serial#=33553 )  is blocking ECMDTA dcjob1asp11 ( SID=50 )" // 测试语句
		if len(sidString) > 1 {
			sid := strings.Split(strings.Split(sidString, " ")[4], "=")[2]
			lock_sql := db.QueryRowContext(ctxt, sql_lock_sqltext, sid)
			var userName, sidStr, status string
			lock_sql.Scan(&userName, &sidStr, &status, &sql_lock_txt)
			fmt.Println(sql_lock_txt)
		}

	} else {
		// 假如没有异常job，则中断后续处理
		return
	}

	// SSH 执行重启脚本
	for _, ip := range serverIps {
		var buf bytes.Buffer
		//exec.Command("ssh -p "+string(server_ssh_port)+" " +ip+" sh /webapp/scripts/restart.sh")
		cmdString := "ssh -p " + string(server_ssh_port) + " " + ip + " tail -30 /webapp/tomcat1/logs/catalina.out ;sh /webapp/scripts/restart.sh"
		cmd := exec.CommandContext(ctxt, cmdString)
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		cmd.Start()
		cmd.Wait()
		fmt.Println(buf.Bytes())
		logger.Println(buf.Bytes())
	}

	// 发出邮件
	user := "systemnotice@wwwarehouse.com"
	password := "j9sw3iFfS%zjArRX"
	host := "smtp.exmail.qq.com:25"
	to := "lizhiyong567@gmail.com"
	subject := "后台任务处理记录"
	body := `
		<html>
		<body>
		<h3> 异常后台任务：</h3>` + "<div>JOB 信息：" + fmt.Sprintf(" %v ", serverIps) + "<br></br></div>" + "<p>数据库锁信息：" + sql_lock_sqltext +
		`</p>
		</body>
		</html>`
	fmt.Println("send email")
	err = SendToMail(user, password, host, to, subject, body, "html")
	if err != nil {
		fmt.Println("Send mail error!")
		fmt.Println(err)
	} else {
		fmt.Println("Send mail success!")
	}

}
