package main

import (
	"net/http"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"flag"
	"time"
)

var tr = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}
var client = &http.Client{Transport: tr}

type IndexStatus struct {
	ProgressURL     string `json:"progressUrl"`
	CurrentProgress int    `json:"currentProgress"`
	Type            string `json:"type"`
	SubmittedTime   string `json:"submittedTime"`
	StartTime       string `json:"startTime"`
	FinishTime      string `json:"finishTime"`
	Success         bool   `json:"success"`
}

func main()  {
	url := flag.String("url", "jiraurl.com", "specify user to use")
	username := flag.String("u", "USER", "specify user to use")
	password := flag.String("p", "PASS", "specify the password")
	cli := flag.Bool("cli", false, "cli run, progress view.  default = FALSE")
	flag.Parse()
	jira_url := *url + "/rest/api/2/reindex"
	do_reindex(jira_url, *username, *password)
	status := false
	progress := 0
	if *cli == true{
		for status != true {
			status, progress = check_reindex(jira_url, *username, *password)
			fmt.Printf("\r %d", progress)
			time.Sleep(10 * time.Second)
		}
		fmt.Printf("%s%t", "Reindex success: ", status)
		fmt.Printf("%s%d", "Progress: ", progress)
	}
}

func check_reindex(url, user,pass string)  (bool,int){
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(user, pass)
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	var record IndexStatus
	json.NewDecoder(resp.Body).Decode(&record)

	return record.Success, record.CurrentProgress
}

func do_reindex(url,user,pass string)  {
	req, _ := http.NewRequest("POST", url, nil)
	req.SetBasicAuth(user, pass)
	q := req.URL.Query()
	q.Add("type", "BACKGROUND")
	req.URL.RawQuery = q.Encode()
	resp, _ := client.Do(req)
	defer resp.Body.Close()
}