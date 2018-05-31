package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
)

// Edit your confluence username and password
var confUsername = "USERname" // Confluence username
var confPassword = "PASSword" // Password
var confSpace = "CONFLUENCE_SPACE_NAME"       // Confluence space key

var wpUrl = "https://wordpresspage.com" // Wordpress page
var wp_posturl = wpUrl + "/wp-json/wp/v2/posts/"
var userUrl = wpUrl + "/wp-json/wp/v2/users/"
var replies = wpUrl + "/wp-json/wp/v2/comments"
var confUrl = "https://confluence.com" + confApi // Confluence page
var confApi = "/rest/api/content"
var ParentPageName string

var tr = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

// For control over HTTP client headers,
// redirect policy, and other settings,
// we create a Client
// A Client is an HTTP client
var client = &http.Client{Transport: tr}

type WPPost struct {
	ID      int    `json:"id"`
	Date    string `json:"date"`
	DateGmt string `json:"date_gmt"`
	GUID    struct {
		Rendered string `json:"rendered"`
	} `json:"guid"`
	Modified    string `json:"modified"`
	ModifiedGmt string `json:"modified_gmt"`
	Status      string `json:"status"`
	Type        string `json:"type"`
	Link        string `json:"link"`
	Title       struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
	Content struct {
		Rendered  string `json:"rendered"`
		Protected bool   `json:"protected"`
	} `json:"content"`
	Excerpt struct {
		Rendered  string `json:"rendered"`
		Protected bool   `json:"protected"`
	} `json:"excerpt"`
	Author int `json:"author"`
}

type WPReplies []struct {
	ID         int    `json:"id"`
	Post       int    `json:"post"`
	Parent     int    `json:"parent"`
	AuthorName string `json:"author_name"`
	Date       string `json:"date"`
	Content    struct {
		Rendered string `json:"rendered"`
	} `json:"content"`
}

type WPUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ConfPost struct {
	Type      string       `json:"type"`
	Title     string       `json:"title"`
	Ancestors []ParentPost `json:"ancestors"`
	Space     struct {
		Key string `json:"key"`
	} `json:"space"`
	Body struct {
		Storage struct {
			Value          string `json:"value"`
			Representation string `json:"representation"`
		} `json:"storage"`
	} `json:"body"`
}

type ParentPost struct {
	ID int `json:"id"`
}

type WPPostsID []struct {
	ID   int    `json:"id"`
	Date string `json:"date"`
}

type ConfPostResponse struct {
	ID    string `json:"id"`
	Links struct {
		Base       string `json:"base"`
		Collection string `json:"collection"`
		Self       string `json:"self"`
		Tinyui     string `json:"tinyui"`
		Webui      string `json:"webui"`
	} `json:"_links"`
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Type Confluence parent top page name to be created: ")
	title, _ := reader.ReadString('\n')
	ParentPageName = title

	postId := RetrievePostsId(wp_posturl)

	RetrievePostData(postId)

	fmt.Print("Export&Import Done\n")
}

func users(userid int) string {

	urlidusers := fmt.Sprintf("%s%d", userUrl, userid)       // Concatenate url and userid
	requsers, err := http.NewRequest("GET", urlidusers, nil) // Creating the request
	resp, err := client.Do(requsers)
	defer resp.Body.Close()
	if err != nil {
		log.Fatal("Do: ", err)
	}
	var recorduser WPUser

	if err := json.NewDecoder(resp.Body).Decode(&recorduser); err != nil {
		log.Println(err)
	}
	return recorduser.Name
}

// Func to retrieve posts id from webpage
func RetrievePostsId(url string) WPPostsID {
	req, _ := http.NewRequest("GET", url, nil)
	// Setting request parameters
	q := req.URL.Query()
	q.Add("per_page", "100") // max limit to 100 wp posts per page
	q.Add("orderby", "date")
	q.Add("order", "desc")
	req.URL.RawQuery = q.Encode()
	// Doing the request
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	var keys WPPostsID
	json.NewDecoder(resp.Body).Decode(&keys)

	return keys
}

func RetrievePostData(posts_id WPPostsID) {
	for id := 0; id < len(posts_id); id++ {
		// Build the request
		url := fmt.Sprintf("%s%d", wp_posturl, posts_id[id].ID)
		CheckoutReplies(posts_id[id].ID)
		req, _ := http.NewRequest("GET", url, nil)
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal("Do: ", err)
			break
		}
		// Always close the Responce body
		defer resp.Body.Close()

		// Fill the record with the data from the JSON
		var record WPPost
		// Use json.Decode for reading streams of JSON data
		json.NewDecoder(resp.Body).Decode(&record)
		post_content := "<br /> Author: " + users(record.Author) + "<br />" + record.Content.Rendered + "<br />" + CheckoutReplies(posts_id[id].ID)
		//// Import posts to Confluence
		fmt.Println(url)

		ConfluenceImport(post_content, record.Title.Rendered)
	}
}

var parentExist = false
var parentPageId string

func ConfluenceImport(page, title string) {
	if !parentExist {
		parentPageId = ConfParent()
		parentExist = true
	}
	ancestor, _ := strconv.Atoi(parentPageId) // Converting string to int
	var p ConfPost

	p.Type = "page"
	p.Title = title
	p.Space.Key = confSpace
	ms := ParentPost{ID: ancestor}
	p.Ancestors = append(p.Ancestors, ms)
	p.Body.Storage.Value = page
	p.Body.Storage.Representation = "storage"

	bytesRepresentation, _ := json.Marshal(p)

	req, err2 := http.NewRequest("POST", confUrl, bytes.NewBuffer(bytesRepresentation))
	if err2 != nil {
		log.Fatalln(err2)
		return
	}

	req.SetBasicAuth(confUsername, confPassword)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Transport: tr}
	resp, _ := client.Do(req)

	fmt.Printf("Confluence Responce SatusCode: %d\n", resp.StatusCode)

	if resp.StatusCode >= 299 {
		if resp.StatusCode == 401 {
			fmt.Println("Authentication error")
			errorBody, _ := ioutil.ReadAll(resp.Body)
			responce := string(errorBody)
			fmt.Println(responce)
			os.Exit(resp.StatusCode)
		} else {
			fmt.Println("Error")
			errorBody, _ := ioutil.ReadAll(resp.Body)
			responce := string(errorBody)
			fmt.Println(responce)
		}
	}
	defer resp.Body.Close()

	var record ConfPostResponse
	json.NewDecoder(resp.Body).Decode(&record)
	fmt.Println(record.Links.Base + record.Links.Webui) // Prints Confluence imported post link

	//bodyText, _ := ioutil.ReadAll(resp.Body)
	//responce := string(bodyText)
	//fmt.Println(responce)    // Uncomment these 3 lines if you want to see the response from Confluence
}

func ConfParent() string {

	var p ConfPost

	p.Type = "page"
	p.Title = ParentPageName
	p.Space.Key = confSpace
	p.Ancestors = nil
	p.Body.Storage.Value = ""
	p.Body.Storage.Representation = "storage"

	bytesRepresentation, _ := json.Marshal(p)

	req, _ := http.NewRequest("POST", confUrl, bytes.NewBuffer(bytesRepresentation))

	req.SetBasicAuth(confUsername, confPassword)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Transport: tr}
	resp, _ := client.Do(req)

	defer resp.Body.Close()

	var record ConfPostResponse
	json.NewDecoder(resp.Body).Decode(&record)

	return record.ID
}

// Script is created by DevOps Engineer - Max Tacu

func CheckoutReplies(posts_id int) string {
	var reply_content string

	reply_url := fmt.Sprintf("%s%s%d%s", replies, "?post=", posts_id, "&order=asc")
	req, _ := http.NewRequest("GET", reply_url, nil)
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	// Fill the record with the data from the JSON
	var record WPReplies
	json.NewDecoder(resp.Body).Decode(&record)
	reply_content = "<br /><h3>Comments</h3>"

	if len(record) != 0 {
		for id := 0; id < len(record); id++ {
			reply_content += "<br /><b>" + record[id].AuthorName + "</b><br />" + record[id].Content.Rendered
		}
	}
	return reply_content
}