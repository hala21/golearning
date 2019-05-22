package main

import (
	"flag"
	"fmt"
	"github.com/Unknwon/goconfig"
	"golang.org/x/crypto/ssh"
	"log"
	"net/smtp"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func getParameter(parString string, parUsage string, logLocal string, filename string) (sec map[string]string, loggerReturn *log.Logger) {
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

	//日志文件
	var layoutISO = "2006-01-02"
	// 保存7天以内的日志文件
	sysType := runtime.GOOS
	if sysType == "linux" {
		exec.Command("find ./ - name *jobcheck*.log -ctime +6 |xargs rm -f ")
	}

	logfileName := *server + "_" + filename + "_" + time.Now().Format(layoutISO) + ".log"
	f, err := os.OpenFile(logfileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	var logger = log.New(f, "check：", log.LstdFlags)

	return sec, logger

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

func SendToMail(to, subject, body, mailtype string) error {
	conf, err := goconfig.LoadConfigFile("utils.init")
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
	if mailtype == "html" {
		contentType = "Content-Type: text/" + mailtype + "; charset=UTF-8"
	} else {
		contentType = "Content-Type: text/plain" + "; charset=UTF-8"
	}
	msg := []byte("To: " + to + "\r\nFrom: " + user + "\r\nSubject: " + subject + " \r\n" + contentType + "\r\n\r\n" + body)
	send_to := strings.Split(to, ";")
	err = smtp.SendMail(host, auth, user, send_to, msg)
	return err
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
