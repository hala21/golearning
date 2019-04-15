package main

/*
  1. get oracle latest 0 level backup on sunday
  2. get mysql backup # get one of backup files which backup on 00:50 and 11:50
  3. get mongodb backup # need to create windows directory(mongo database directory)
  4. function: 1)scripts get parameter 2)writer log to logfile
*/

import (
	"flag"
	"fmt"
	"github.com/Unknwon/goconfig"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func connect(user, password, host string, port int) (*sftp.Client, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		sftpClient   *sftp.Client
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

	// create sftp client
	if sftpClient, err = sftp.NewClient(sshClient); err != nil {
		return nil, err
	}

	return sftpClient, nil
}

var sftpClient *sftp.Client

func main() {

	nowTime := time.Now()

	//get command parameter
	server := flag.String("server", "", "server_info")

	flag.Parse()

	if flag.NFlag() == 0 || len(*server) == 0 {
		log.Println("Please input parameter")
		return
	}

	// get config file parameter
	conf, err := goconfig.LoadConfigFile("config.ini")
	if err != nil {
		log.Fatal(err)
		return
	}
	sec, err := conf.GetSection(*server)
	if err != nil {
		log.Println("Please input correct parameter")
		return
	}

	//var username, password, hostIP, port, remoteDir, localDir string
	username := sec["username"]
	password := sec["password"]
	hostIp := sec["hostIp"]
	sshPort, _ := strconv.Atoi(sec["sshPort"])
	remoteDir := sec["remoteDir"]
	localDir := sec["localDir"]

	if strings.Contains(*server, "pro_mysql") {
		hour := nowTime.Hour()
		if hour >= 12 {
			remoteDir = remoteDir + nowTime.Format("060102") + "1150/"
		} else {
			remoteDir = remoteDir + nowTime.Format("060102") + "0050/"
		}
	}

	if strings.Contains(*server, "pro_mongo") {
		remoteDir = remoteDir + nowTime.Format("20060102") + "/"
	}

	var layoutISO = "2006-01-02"
	logfileName := *server + "_copyFiles_" + nowTime.Format(layoutISO) + ".log"
	f, err := os.OpenFile(logfileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	var logger = log.New(f, "copy", log.LstdFlags)
	logger.Printf("start copy %v \r ", nowTime.Location())

	//remove local directory files
	_, err = os.Stat(localDir)
	if err == nil {
		if err := os.RemoveAll(localDir); err != nil {
			logger.Printf("remove old data fault %v", err)
		}
	}

	//if os.IsNotExist(err) {
	//	// windows's bug
	//	if !strings.Contains(err.Error(), "The system cannot find the file specified"){
	//		logger.Printf("deal with directory error %v",err)
	//	}
	//}

	// create new directory
	if err := os.Mkdir(localDir, 0644); err != nil {
		logger.Println("mkdir directory fault")
	}

	// 连接ssh服务器
	sftpClient, err = connect(username, password, hostIp, sshPort)
	if err != nil {
		logger.Fatal(err)
	} else {
		logger.Println("ssh login success")
	}

	defer sftpClient.Close()

	sunday := nowTime.AddDate(0, 0, -int(time.Now().Weekday())).Format(layoutISO)

	logger.Printf("walk directory %s \r", remoteDir)
	w := sftpClient.Walk(remoteDir)

	for w.Step() {
		if w.Err() != nil {
			continue
		}
		fileInfo := w.Stat()
		filePath := w.Path()

		// the first value is directory, if open the directory will panic error
		if fileInfo.IsDir() {
			continue
		}

		remoteFilename := ""
		if strings.Contains(*server, "ecm") || strings.Contains(*server, "wms") {
			if fileInfo.ModTime().Format(layoutISO) == sunday {
				remoteFilename = fileInfo.Name()
			}
		} else {
			remoteFilename = fileInfo.Name()
		}

		if len(remoteFilename) == 0 {
			continue
		}

		//srcFile, err := sftpClient.Open(remoteDir+remoteFilename)
		srcFile, err := sftpClient.Open(filePath)
		if err != nil {
			logger.Fatal(err)
		}

		remoteFileInfo := strings.Replace(filePath, remoteDir, "", -1)
		remoteInfo := strings.Split(remoteFileInfo, "/")

		if len(remoteInfo) > 1 {
			localDirTmp := localDir + strings.Join(remoteInfo[0:len(remoteInfo)-1], "\\")
			localDir := localDirTmp

			//dstFile, err := os.Create(path.Join(localDir, remoteFilename))

			_, err = os.Stat(localDir)
			if err != nil {
				if err := os.Mkdir(localDir, 0644); err != nil {
					logger.Println("mkdir directory fault")
				}
			}

			logger.Println(localDir + "\\" + remoteFilename)

			dstFile, err := os.Create(localDir + "\\" + remoteFilename)
			if err != nil {
				logger.Fatal(err)
			}

			if _, err = srcFile.WriteTo(dstFile); err != nil {
				logger.Fatal(err)
			} else {
				srcFile.Close()
				dstFile.Close()
			}
		} else {

			logger.Println(remoteFilename)
			dstFile, err := os.Create(localDir + remoteFilename)
			if err != nil {
				logger.Fatal(err)
			}

			if _, err = srcFile.WriteTo(dstFile); err != nil {
				logger.Fatal(err)
			} else {
				srcFile.Close()
				dstFile.Close()
			}
		}
	}
}
