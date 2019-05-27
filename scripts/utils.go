package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Unknwon/goconfig"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func GetParameter(parString string, parUsage string, logLocal string) (sec map[string]string) {
	// 获取参数
	server := flag.String(parString, "", parUsage)
	flag.Parse()
	if flag.NFlag() == 0 || len(*server) == 0 {
		log.Println("Please input parameter")
		return
	}
	// 参数文件名
	conf, err := goconfig.LoadConfigFile(logLocal)
	if err != nil {
		log.Fatal(err)
		return
	}
	sec, err = conf.GetSection(*server)
	if err != nil {
		log.Println("Please input correct parameter")
		return
	}

	return sec
}

func LogConf(filename string) *log.Logger {
	//日志文件
	var layoutISO = "2006-01-02"
	// 保存7天以内的日志文件
	sysType := runtime.GOOS
	if sysType == "linux" {
		exec.Command("find ./ - name *jobcheck*.log -ctime +6 |xargs rm -f ")
	}

	logfileName := filename + "_" + time.Now().Format(layoutISO) + ".log"
	f, err := os.OpenFile(logfileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	return log.New(f, "check：", log.LstdFlags)
}

func sshConn(host, user, password string, port int) (*ssh.Client, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		err          error
	)
	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(password))

	clientConfig = &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		Timeout:         60 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// connect to ssh
	addr = fmt.Sprintf("%s:%d", host, port)

	if sshClient, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	return sshClient, nil
}

func SendToMail(to, subject, body, mailType string) error {
	conf, err := goconfig.LoadConfigFile("utils.ini")
	sec, err := conf.GetSection("")
	if err != nil {
		log.Fatal(err)
		return err
	}
	host := sec["smtp_server"]
	user := sec["smtp_user"]
	password := sec["smtp_password"]
	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	var contentType string
	if mailType == "html" {
		contentType = "Content-Type: text/" + mailType + "; charset=UTF-8"
	} else {
		contentType = "Content-Type: text/plain" + "; charset=UTF-8"
	}
	msg := []byte("To: " + to + "\r\nFrom: " + user + "\r\nSubject: " + subject + " \r\n" + contentType + "\r\n\r\n" + body)
	sendTo := strings.Split(to, ";")
	err = smtp.SendMail(host, auth, user, sendTo, msg)
	return err
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// 无人值守时间，非工作时间
func standbyUnattendedTime() bool {
	type holiday struct {
		Code string `json:"code"`
		Data int    `json:"data"`
	}

	/*
		1、接口地址：http://api.goseek.cn/Tools/holiday?date=数字日期，支持https协议。
		//2、返回数据：正常工作日对应结果为 0, 法定节假日对应结果为 1, 节假日调休补班对应的结果为 2，休息日对应结果为 3
	*/
	now := time.Now()
	today := now.Format("20160102")
	resp, err := http.Get("http://api.goseek.cn/Tools/holiday?date=" + today)
	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	var nonWorkday holiday
	err = json.Unmarshal(result, &nonWorkday)
	checkErr(err)
	resultHoliday := nonWorkday.Data

	// 计算工作日的，非工作时间
	if resultHoliday == 0 || resultHoliday == 2 {
		hour, minute, _ := now.Clock()
		// 8点之前，17点之后
		if hour <= 8 || hour >= 17 {
			// 8点半之前
			if hour == 8 {
				if minute < 30 {
					return true
				} else {
					return false
				}
			}
			// 17点半之后
			if hour == 17 {
				if minute > 30 {
					return true
				} else {
					return false
				}
			}

		} else {
			// 工作时间返回false
			return false
		}
	}

	// 假日和周末
	return true
}

/*
func callPhone(phoneNum string, tplId int) {

	const paramStr string =
		"param=%s" +
		"phone=%s" +
		"&tpl_id=%s"

	url := "http://yuyin2.market.alicloudapi.com/dx/voice_notice"
	config, err := goconfig.LoadConfigFile("utils.ini")
	checkErr(err)
	//sec, err := config.GetSection("")
	checkErr(err)
    //appCode := sec["appCode"]

	appCodehttp.Post(url)

}
*/
