# Slack Access Logs Exporter

This Go script exports access logs from a Slack workspace using the `team.accessLogs` API. The script respects Slack's API rate limits and efficiently organizes logs by month, saving them to CSV files.

## Features

- Fetches logs for the entire Slack workspace.
- Organizes logs into monthly CSV files (e.g., `slack_logs_2022-01.csv`).
- Stops fetching for the current month when encountering logins from the next month.
- Handles Slack's API rate limit of 20 calls per minute.
- Skips duplicate logins based on unique keys (`username`, `dateFirst`, `ip`, and `userAgent`).
- Excludes logins from the `outlook_calendar` username.

## Prerequisites

1. Go installed on your system.
2. A valid Slack token with the required permissions:
   - Token format: `xoxp-...`
   - Permissions: `team.accessLogs:read`

## Installation

1. Clone the repository or save the script to a file (e.g., `export.go`).
2. Ensure Go is installed:
   ```bash
   go version
   ```

## Configuration

- Replace the placeholder Slack token in the script:
  ```go
  const token = "xoxp-your-slack-token"
  ```

## Usage

1. Run the script:
   ```bash
   go run export.go
   ```
2. The script will:
   - Fetch logs starting from January 2021.
   - Save monthly logs to CSV files in the same directory.

### CSV File Format

Each CSV file will have the following fields:

| Field      | Description                        |
|------------|------------------------------------|
| `username` | Slack username of the user.        |
| `datelogin`| Timestamp of the login in `yyyy-MM-dd-HH:mm:ss` format. |
| `ip`       | IP address of the user.            |
| `useragent`| User agent string of the login.    |
| `isp`      | ISP information for the IP address.|

## Example Output

```bash
Fetching logs for 2022-01...
Processing page 1 with before=1640995199
Processing page 2 with before=1640995199
Encountered a login from the next month. Stopping further requests for this month...
Saving logs to slack_logs_2022-01.csv...
Next Month: 2022-02-28-23:59:59
```

### CSV Example

`slack_logs_2022-01.csv`:
```csv
username,datelogin,ip,useragent,isp
john_doe,2022-01-15-08:30:00,192.168.1.1,Slack/4.0.0 (iOS),ISP Inc.
jane_smith,2022-01-20-14:50:00,10.0.0.1,Slack/4.0.0 (Android),ISP Inc.
```

## Notes

1. **Rate Limiting**:
   - The script ensures no more than 20 API calls per minute.
   - If the limit is reached, it waits before continuing.

2. **Duplicate Detection**:
   - Each login is uniquely identified by:
     - `username`
     - `dateFirst`
     - `ip`
     - `userAgent`

3. **Date Handling**:
   - The script correctly handles transitions between months, including edge cases like February in leap years.

## Troubleshooting

1. **API Errors**:
   - Ensure your token has the correct permissions.
   - Check the Slack API documentation for potential issues: [Slack API Docs](https://api.slack.com/).

2. **Invalid Logs**:
   - If logs appear incorrect, verify the CSV files for duplicates or incorrect timestamps.

## License

This script is provided under the MIT License. Feel free to modify and distribute it.
