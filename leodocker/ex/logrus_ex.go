package ex

import (
	//log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus"
	"log"
	"os"
)

//func init(){
//	log.SetFormatter(&log.JSONFormatter{})
//	log.SetOutput(os.Stdout)
//	log.SetLevel(log.InfoLevel)
//}

//var log = logrus.New()
//log.Formatter = &logrus.JSONFormatter{}

func main() {
	//log.WithFields(log.Fields{
	//	"animal":"walrus",
	//}).Info("a walrus appears")

	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}

	logfile, err := os.OpenFile("logrus.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		log.SetOutput(logger.Writer())
	} else {
		log.Fatalln("can  not  open file")
	}

	//logrus.WithFields(logrus.Fields{"filename":"walrus"}).Info("faild to open file!")

}
