package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	apiURL      = "https://slack.com/api/team.accessLogs"
	token       = ""
	pageSize    = 1000
	apiRateLimit = 20 // Max API calls per minute
)

type AccessLogResponse struct {
	OK     bool `json:"ok"`
	Error  string `json:"error"`
	Logins []struct {
		Username  string `json:"username"`
		DateFirst int64  `json:"date_first"`
		IP        string `json:"ip"`
		UserAgent string `json:"user_agent"`
		ISP       string `json:"isp"`
	} `json:"logins"`
	Paging struct {
		Page  int `json:"page"`
		Pages int `json:"pages"`
	} `json:"paging"`
}

func fetchLogs(before int64, apiCallCount *int) ([]map[string]string, error) {
	var allLogs []map[string]string
	page := 1

	for page <= 50 { // Ensure we don't exceed the API's 100-page limit
		// Check and pause if nearing rate limit
		if *apiCallCount >= apiRateLimit {
			fmt.Println("API rate limit reached. Pausing for a minute...")
			time.Sleep(1 * time.Minute)
			*apiCallCount = 0
		}

		fmt.Printf("Processing page %d with before=%d\n", page, before)

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		q := req.URL.Query()
		q.Add("count", strconv.Itoa(pageSize))
		q.Add("page", strconv.Itoa(page))
		q.Add("before", strconv.FormatInt(before, 10))
		req.URL.RawQuery = q.Encode()

		req.Header.Set("Authorization", "Bearer "+token)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		*apiCallCount++ // Increment API call count

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		var result AccessLogResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w\nResponse body: %s", err, string(body))
		}

		if !result.OK {
			return nil, fmt.Errorf("API error: %s", result.Error)
		}

		for _, login := range result.Logins {
			// Skip duplicates
			loginKey := fmt.Sprintf("%s_%d", login.Username, login.DateFirst)
			if _, exists := processedLogins[loginKey]; exists {
				continue
			}
			processedLogins[loginKey] = true

			isoDate := time.Unix(login.DateFirst, 0).UTC().Format(time.RFC3339)
			allLogs = append(allLogs, map[string]string{
				"username":   login.Username,
				"datelogin":  isoDate,
				"ip":         login.IP,
				"useragent":  login.UserAgent,
				"isp":        login.ISP,
			})
		}

		if page >= result.Paging.Pages {
			break
		}
		page++
	}

	return allLogs, nil
}

func saveLogsToCSV(filename string, logs []map[string]string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"username", "datelogin", "ip", "useragent", "isp"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, login := range logs {
		record := []string{
			login["username"],
			login["datelogin"],
			login["ip"],
			login["useragent"],
			login["isp"],
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

var processedLogins = make(map[string]bool) // To track processed logins

func main() {
	startDate := time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Now().AddDate(0, 1, 0)
	currentDate := startDate
	apiCallCount := 0

	for currentDate.Before(endDate) {
		fmt.Printf("Fetching logs before %s...\n", currentDate.Format(time.RFC3339))
		logs, err := fetchLogs(currentDate.Unix(), &apiCallCount)
		if err != nil {
			fmt.Printf("Error fetching logs: %v\n", err)
			return
		}

		if len(logs) > 0 {
			filename := fmt.Sprintf("slack_logs_%d-%02d.csv", currentDate.Year(), currentDate.Month())
			fmt.Printf("Saving logs to %s...\n", filename)
			if err := saveLogsToCSV(filename, logs); err != nil {
				fmt.Printf("Error saving logs: %v\n", err)
				return
			}
		}

		// Pause for a minute at the end of processing each year
		if currentDate.Month() == time.December {
			fmt.Println("Completed a year of logs. Pausing for a minute...")
			time.Sleep(1 * time.Minute)
		}

		// Increment by 1 month
		currentDate = currentDate.AddDate(0, 1, 0)
	}
}
