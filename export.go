package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	apiURL       = "https://slack.com/api/team.accessLogs"
	token        = "" // Replace with your actual token
	startDate    = "2018-01-01"       // Starting date in YYYY-MM-DD format
	pageSize     = 1000               // Items per page
	outputCSV    = "slack_logins.csv"
	rateLimit    = 20                 // Max requests per minute
	rateLimitGap = 60 / rateLimit     // Time gap in seconds between requests
)

type SlackResponse struct {
	OK     bool `json:"ok"`
	Paging struct {
		Pages int `json:"pages"`
	} `json:"paging"`
	Logins []Login `json:"logins"`
}

type Login struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	DateFirst int64  `json:"date_first"`
	DateLast  int64  `json:"date_last"`
	Count     int    `json:"count"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	ISP       string `json:"isp"`
	Country   string `json:"country"`
	Region    string `json:"region"`
}

func main() {
	currentDate, _ := time.Parse("2006-01-02", startDate)
	allLogins := make(map[string]Login)

	for {
		fmt.Printf("Fetching logs for date: %s\n", currentDate.Format("2006-01-02"))
		epoch := currentDate.Unix()
		hasMoreData := true

		for page := 1; hasMoreData; page++ {
			fmt.Printf("Fetching page %d...\n", page)
			logins, hasMore := fetchLogs(epoch, page)
			for _, login := range logins {
				allLogins[login.UserID+strconv.FormatInt(login.DateLast, 10)] = login
			}
			hasMoreData = hasMore

			// Rate limit: wait before making the next request
			time.Sleep(time.Duration(rateLimitGap) * time.Second)
		}

		// Move to the next month
		currentDate = currentDate.AddDate(0, 1, 0)

		// Stop when reaching the current date
		if currentDate.After(time.Now()) {
			break
		}
	}

	fmt.Println("Saving logs to CSV...")
	saveToCSV(allLogins)
	fmt.Println("Done!")
}

func fetchLogs(before int64, page int) ([]Login, bool) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	q := req.URL.Query()
	q.Add("before", strconv.FormatInt(before, 10))
	q.Add("count", strconv.Itoa(pageSize))
	q.Add("page", strconv.Itoa(page))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to make API call: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("API call failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	var slackResp SlackResponse
	if err := json.Unmarshal(body, &slackResp); err != nil {
		log.Fatalf("Failed to parse response JSON: %v", err)
	}

	if !slackResp.OK {
		log.Fatalf("API response indicates failure: %v", string(body))
	}

	return slackResp.Logins, page < slackResp.Paging.Pages
}

func saveToCSV(logins map[string]Login) {
	file, err := os.Create(outputCSV)
	if err != nil {
		log.Fatalf("Failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"UserID", "Username", "DateFirst", "DateLast", "Count", "IP", "UserAgent", "ISP", "Country", "Region"}
	if err := writer.Write(header); err != nil {
		log.Fatalf("Failed to write header to CSV: %v", err)
	}

	// Write logins
	for _, login := range logins {
		record := []string{
			login.UserID,
			login.Username,
			strconv.FormatInt(login.DateFirst, 10),
			strconv.FormatInt(login.DateLast, 10),
			strconv.Itoa(login.Count),
			login.IP,
			login.UserAgent,
			login.ISP,
			login.Country,
			login.Region,
		}
		if err := writer.Write(record); err != nil {
			log.Fatalf("Failed to write record to CSV: %v", err)
		}
	}
}
