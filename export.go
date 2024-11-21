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
	token        = ""
	startDate    = "2021-01-01"
	pageSize     = 1000
	rateLimit    = 20
	rateLimitGap = 60 / rateLimit
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
	DateLast  int64  `json:"date_last"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	ISP       string `json:"isp"`
}

func main() {
	currentDate, _ := time.Parse("2006-01-02", startDate)
	allLogins := make(map[int]map[string]Login) // Separate logins by year

	for {
		fmt.Printf("Fetching logs for date: %s\n", currentDate.Format("2006-01-02"))
		epoch := currentDate.Unix()
		hasMoreData := true

		for page := 1; hasMoreData; page++ {
			fmt.Printf("Fetching page %d...\n", page)
			logins, hasMore := fetchLogs(epoch, page)

			for _, login := range logins {
				year := time.Unix(login.DateLast, 0).Year()
				if _, exists := allLogins[year]; !exists {
					allLogins[year] = make(map[string]Login)
				}
				key := login.Username + strconv.FormatInt(login.DateLast, 10)
				allLogins[year][key] = login
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

	fmt.Println("Saving logs to separate yearly CSV files...")
	for year, logins := range allLogins {
		saveToYearlyCSV(year, logins)
	}
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

func saveToYearlyCSV(year int, logins map[string]Login) {
	fileName := fmt.Sprintf("slack_logins_%d.csv", year)
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("Failed to create CSV file for year %d: %v", year, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Username", "DateLogin", "IP", "UserAgent", "ISP"}
	if err := writer.Write(header); err != nil {
		log.Fatalf("Failed to write header to CSV: %v", err)
	}

	// Write logins
	for _, login := range logins {
		dateLogin := time.Unix(login.DateLast, 0).Format("2006-01-02-15:04:05")
		record := []string{
			login.Username,
			dateLogin,
			login.IP,
			login.UserAgent,
			login.ISP,
		}
		if err := writer.Write(record); err != nil {
			log.Fatalf("Failed to write record to CSV: %v", err)
		}
	}
}
