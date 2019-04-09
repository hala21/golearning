package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//type project struct {
//	name string
//	projectId string
//}

type Projects struct {
	project []project
}

type project struct {
	Name      string `json:"name"`
	ProjectId int    `json:"project_id"`
}

type repositories struct {
	Name string
}

type tags struct {
	Name string
}

func httpRequest(url string) io.ReadCloser {
	// json data
	//url := "https://192.168.6.11/api/projects"
	username := "admin"
	password := "Harbor12345"
	tr := &http.Transport{ // ignore x509: certificate signed by unknown authority
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Timeout: 15 * time.Second, Transport: tr}
	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	//fmt.Println(res.Body)

	return res.Body

}

func getContent() {

	//tags := "192.168.6.11"

	//body, err := ioutil.ReadAll(res.Body)

	// proejct
	//data := make([]project, 0)
	var project []project
	projectUrl := "https://192.168.6.11/api/projects"
	body := httpRequest(projectUrl)
	defer body.Close()

	err := json.NewDecoder(body).Decode(&project)
	if err != nil {
		panic(err)
	}
	//for _,value :=range project{
	//	fmt.Println(strconv.Itoa(value.ProjectId))
	//}

	//repositories
	for _, value := range project {
		//repositories
		repositoriesUrl := "https://192.168.6.11/api/repositories?project_id="
		//projectName := value.Name
		projectId := value.ProjectId
		repositoriesUrlNew := strings.Join([]string{repositoriesUrl, strconv.Itoa(projectId)}, "")
		//fmt.Println(repositoriesUrlNew)
		var repo []repositories
		bodyRepo := httpRequest(repositoriesUrlNew)
		//defer bodyRepo.Close()

		//f2 := ioutil.NopCloser(bodyRepo)
		//byte3,err := ioutil.ReadAll(f2)
		//if err != nil {
		//	panic(err)
		//}
		//fmt.Println(string(byte3))

		//content, _ := ioutil.ReadAll(bodyRepo)
		//fmt.Println(string(content))

		err = json.NewDecoder(bodyRepo).Decode(&repo)
		if err != nil {
			panic(err)
		}

		for _, value := range repo {

			tagsUrl := "https://192.168.6.11/api/repositories/"
			var repoName = value.Name
			tagsUrlNew := strings.Join([]string{tagsUrl, repoName, "/tags"}, "")

			var tag []tags
			bodyTags := httpRequest(tagsUrlNew)
			defer bodyTags.Close()

			err := json.NewDecoder(bodyTags).Decode(&tag)
			if err != nil {
				panic(err)
			}
			for _, value := range tag {
				fmt.Println(repoName + ":" + value.Name)
			}

		}

	}

}

func main() {
	getContent()
}
