package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/labstack/gommon/color"
	"github.com/subosito/gozaru"
)

// Podcast struct
type Podcast struct {
	Media struct {
		Title       string `json:"title"`
		MediaURL    string `json:"mediaUrl"`
		MediaType   string `json:"mediaType"`
		Description string `json:"description"`
		Poster      string `json:"poster"`
		ID          int    `json:"id"`
		Rating      struct {
			Rating     int         `json:"rating"`
			UserRating interface{} `json:"userRating"`
		} `json:"rating"`
	} `json:"media"`
	StartAt interface{} `json:"startAt"`
}

var client = http.Client{}

var checkPre = color.Yellow("[") + color.Green("✓") + color.Yellow("]")
var tildPre = color.Yellow("[") + color.Green("~") + color.Yellow("]")
var crossPre = color.Yellow("[") + color.Red("✗") + color.Yellow("]")

func init() {
	// Disable HTTP/2: Empty TLSNextProto map
	client.Transport = http.DefaultTransport
	client.Transport.(*http.Transport).TLSNextProto =
		make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
}

func downloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func downloadPodcast(podcastJSON *Podcast, id string) {
	errorMessage := crossPre + color.Yellow(" [ ") +
		color.Green("Podcast ") + color.Yellow("#") + color.Green(id) + color.Yellow(" ] ") +
		color.Red("Error while processing ") + color.Yellow(podcastJSON.Media.Title)
	podcastPath := arguments.Output + "/" + id + " - " + gozaru.Sanitize(podcastJSON.Media.Title) + "/"

	err := os.MkdirAll(podcastPath, os.ModePerm)
	if err != nil {
		fmt.Println(errorMessage)
		fmt.Println(err.Error())
		return
	}

	file, err := json.MarshalIndent(podcastJSON, "", " ")
	if err != nil {
		fmt.Println(errorMessage)
		fmt.Println(err.Error())
		return
	}
	err = ioutil.WriteFile(podcastPath+id+" - "+gozaru.Sanitize(podcastJSON.Media.Title)+".json", file, 0644)
	if err != nil {
		fmt.Println(errorMessage)
		fmt.Println(err.Error())
		return
	}

	downloadFile(podcastPath+id+" - "+gozaru.Sanitize(podcastJSON.Media.Title)+path.Ext(podcastJSON.Media.MediaURL), podcastJSON.Media.MediaURL)
	downloadFile(podcastPath+id+" - "+gozaru.Sanitize(podcastJSON.Media.Title)+path.Ext(podcastJSON.Media.Poster), podcastJSON.Media.Poster)
}

func getPodcastJSON(URL string, id string, worker *sync.WaitGroup) {
	defer worker.Done()

	podcastJSON := new(Podcast)
	r, err := client.Get(URL)
	if err != nil || r.StatusCode != 200 {
		return
	}
	defer r.Body.Close()

	json.NewDecoder(r.Body).Decode(podcastJSON)

	fmt.Println(checkPre + color.Yellow(" [ ") +
		color.Green("Podcast ") + color.Yellow("#") + color.Green(id) + color.Yellow(" ] ") +
		color.Green("Downloading ") + color.Yellow(podcastJSON.Media.Title))

	downloadPodcast(podcastJSON, id)
}

func crawl() {
	var worker sync.WaitGroup
	var id string
	var count int

	// Loop through pages
	for index := arguments.StartID; index <= arguments.StopID; index++ {
		worker.Add(1)
		count++
		id = strconv.Itoa(index)
		go getPodcastJSON("https://podtail.com/podcast/episode/json/?id="+id, id, &worker)
		if count == arguments.Concurrency {
			worker.Wait()
			count = 0
		}
	}
}

func main() {
	// Parse arguments
	parseArgs(os.Args)

	// Create output directory
	os.MkdirAll(arguments.Output, os.ModePerm)

	crawl()
}
