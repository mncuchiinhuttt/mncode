package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HandleAPICommand tests REST/HTTP endpoints directly from the terminal
func HandleAPICommand(parts []string) {
	if len(parts) < 2 {
		fmt.Println()
		fmt.Println(BoldCyan("API ENDPOINT TESTER USAGE:"))
		fmt.Println("  /api get <url>                    - Send HTTP GET request")
		fmt.Println("  /api post <url> <json_body>       - Send HTTP POST request with JSON")
		fmt.Println("  /api delete <url>                 - Send HTTP DELETE request")
		fmt.Println()
		fmt.Println(GrayText("Example: /api get http://localhost:3000/api/users"))
		fmt.Println()
		return
	}

	method := "GET"
	targetURL := ""
	var reqBody io.Reader

	if len(parts) == 2 {
		targetURL = parts[1]
	} else {
		method = strings.ToUpper(parts[1])
		targetURL = parts[2]
		if len(parts) > 3 {
			bodyStr := strings.Join(parts[3:], " ")
			reqBody = bytes.NewBufferString(bodyStr)
		}
	}

	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "http://" + targetURL
	}

	fmt.Printf("\n%s %s %s...\n", BoldCyan("🌐 [HTTP Request]"), BoldYellow(method), Bold(targetURL))

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(method, targetURL, reqBody)
	if err != nil {
		fmt.Printf("\n%s Failed creating request: %v\n\n", BoldRed("[Error]"), err)
		return
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", "mncode-cli/v0.1.1")

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("\n%s Request failed: %v\n\n", BoldRed("[Network Error]"), err)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	// Status Badge
	var statusBadge string
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		statusBadge = BoldGreen(fmt.Sprintf("[%s]", resp.Status))
	} else if resp.StatusCode >= 400 {
		statusBadge = BoldRed(fmt.Sprintf("[%s]", resp.Status))
	} else {
		statusBadge = BoldYellow(fmt.Sprintf("[%s]", resp.Status))
	}

	fmt.Println()
	fmt.Printf("  %s  %s %s  %s %s\n",
		statusBadge,
		GrayText("Latency:"), BoldCyan(fmt.Sprintf("%dms", elapsed.Milliseconds())),
		GrayText("Size:"), GrayText(fmt.Sprintf("%.1f KB", float64(len(bodyBytes))/1024.0)))
	fmt.Println(GrayText(strings.Repeat("─", 50)))

	// Formatted JSON Preview
	var prettyJSON bytes.Buffer
	if json.Indent(&prettyJSON, bodyBytes, "", "  ") == nil {
		lines := strings.Split(prettyJSON.String(), "\n")
		maxShow := 30
		for i, l := range lines {
			if i >= maxShow {
				fmt.Printf("  %s\n", GrayText(fmt.Sprintf("... [%d more response lines folded]", len(lines)-maxShow)))
				break
			}
			fmt.Printf("  %s\n", l)
		}
	} else {
		fmt.Printf("  %s\n", string(bodyBytes))
	}
	fmt.Println()
}
