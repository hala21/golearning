package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"regexp"
	"strings"
	"time"
)

func sendToMail(user, password, host, to, subject, body, mailtype string) error {
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

	var (
		layoutISO = "2006-01-02"
		ipFile    = "ip.txt"
	)
	logfileName := "dyncip_" + time.Now().Format(layoutISO) + ".log"
	f, err := os.OpenFile(logfileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	var logger = log.New(f, "check：", log.LstdFlags)

	// 获取外网IP地址，地址保存在本地文件，和本地文件进行比对，假如IP改变后，发出邮件通知
	// 错误信息写入错误日志文件
	/* 可用的url地址
	url1: http://members.3322.org/dyndns/getip
	url2: myip.ipip.net
	url3: ifconfig.me
	*/
	resp, err := http.Get("http://members.3322.org/dyndns/getip")

	if resp == nil {
		logger.Fatal("没有获取IP地址，也许外网断了")
		return
	}

	if err != nil {
		logger.Println("copy")
	}

	defer resp.Body.Close()

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("")
		return
	}
	reg := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	ip := reg.FindString(string(result))

	ipF, err := os.Open(ipFile)
	if err != nil {
		if os.IsNotExist(err) {
			f, err := os.Create(ipFile)
			if err != nil {
				logger.Println(err)
			}
			ipF = f
			defer f.Close()
		}
	}

	fileIp, err := ioutil.ReadAll(ipF)
	if err != nil {
		logger.Fatal(err)
	}

	if len(fileIp) == 0 {
		ioutil.WriteFile(ipFile, []byte(ip), 0777)
	} else {
		if ip == string(fileIp) {
			return
		} else {
			ioutil.WriteFile(ipFile, []byte(ip), 0777)
			// 发出邮件
			user := "systemnotice@wwwarehouse.com"
			// 请添加密码信息
			password := ""
			host := "smtp.exmail.qq.com:25"
			to := "ops-team@wwwarehouse.com"
			subject := "浠漾杭州仓IP改变"
			body := `
		<html>
		<body>
		<h3> 外网IP已改变：</h3>` + "<div>新IP：" + fmt.Sprintf(" %v ", ip) + "<br></br></div>" +
				`</body>
		</html>`
			err = sendToMail(user, password, host, to, subject, body, "html")
			if err != nil {
				fmt.Println("Send mail error!")
				fmt.Println(err)
			} else {
				fmt.Println("Send mail success!")
			}
		}
	}

}
