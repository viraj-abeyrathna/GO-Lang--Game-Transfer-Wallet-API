package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

const PORT = ":3001"

// GenerateSign generates an MD5 signature for the given form data and MD5 key.
func GenerateSign(formData map[string]interface{}, md5Key string) string {
	// Sort keys alphabetically
	keys := make([]string, 0, len(formData))
	for key := range formData {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Create the query string
	var queryString strings.Builder

	for i, key := range keys {
		value := formData[key]
		if value != nil {
			if i > 0 {
				queryString.WriteString("&") // Add '&' only between parameters
			}
			queryString.WriteString(fmt.Sprintf("%s=%v", key, value))
		}
	}

	queryString.WriteString(md5Key)

	// Generate MD5 hash
	hash := md5.Sum([]byte(queryString.String()))

	fmt.Println("")
	fmt.Println("Query String:", queryString.String())

	return hex.EncodeToString(hash[:])
}

// CallApi sends a POST request to the specified URL with the given data and MD5 key.
func CallApi(url string, data map[string]interface{}, md5Key string) (map[string]interface{}, error) {
	// Generate the signature
	sign := GenerateSign(data, md5Key)
	data["sign"] = sign

	// Convert data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Send POST request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse the response JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// HandleSubmit handles the POST request to /api/submit.
func HandleSubmit(w http.ResponseWriter, r *http.Request) {
	var requestBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	action := requestBody["action"].(string)
	md5Key := requestBody["md5_key"].(string)
	apiUrl := requestBody["api_url"].(string)

	data := make(map[string]interface{})
	var apiPath string

	switch action {
	case "game_init":
		apiPath = "/transfer/game/init"
		data["cid"] = requestBody["cid"]
		data["uid"] = requestBody["uid"]
		data["nickname"] = requestBody["nickname"]
		data["channel_id"] = requestBody["channel_id"]
		data["sub_channel_id"] = requestBody["sub_channel_id"]
		data["game_id"] = requestBody["game_id"]
		data["language"] = requestBody["language"]
		data["user_group"] = requestBody["user_group"]
		data["vip_level"] = requestBody["vip_level"]
		if amount, ok := requestBody["amount"].(float64); ok && amount > 0 {
			data["amount"] = amount
			data["transaction_id"] = requestBody["transaction_id"]
		}

	case "user_balance":
		apiPath = "/transfer/user/balance"
		data["cid"] = requestBody["cid"]
		data["uid"] = requestBody["uid"]

	case "user_deposit":
		apiPath = "/transfer/user/deposit"
		data["cid"] = requestBody["cid"]
		data["uid"] = requestBody["uid"]
		data["amount"] = requestBody["amount"]
		data["transaction_id"] = requestBody["transaction_id"]

	case "user_withdraw":
		apiPath = "/transfer/user/withdraw"
		data["cid"] = requestBody["cid"]
		data["uid"] = requestBody["uid"]
		data["amount"] = requestBody["amount"]
		data["transaction_id"] = requestBody["transaction_id"]

	case "game_list":
		apiPath = "/transfer/game/list"
		data["cid"] = requestBody["cid"]

	case "user_kick":
		apiPath = "/transfer/user/kick"
		data["cid"] = requestBody["cid"]
		data["uid"] = requestBody["uid"]
		data["status"] = requestBody["status"]

	case "game_log":
		apiPath = "/transfer/game/log"
		data["cid"] = requestBody["cid"]
		data["start_time"] = requestBody["start_time"]
		data["end_time"] = requestBody["end_time"]
		data["page"] = requestBody["page"]
		data["page_size"] = requestBody["page_size"]

	case "user_order":
		apiPath = "/transfer/user/order"
		data["cid"] = requestBody["cid"]
		data["transaction_id"] = requestBody["transaction_id"]

	default:
		http.Error(w, "Invalid action provided", http.StatusBadRequest)
		return
	}

	// Call the API
	apiResponse, err := CallApi(apiUrl+apiPath, data, md5Key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the API response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"apiResponse": apiResponse})
}

// HandleIndex serves the index.html file.
func HandleIndex(w http.ResponseWriter, r *http.Request) {
	filePath := "index.html"
	file, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(file)
}

func main() {
	http.HandleFunc("/api/submit", HandleSubmit)
	http.HandleFunc("/", HandleIndex)

	// 404 handler
	http.HandleFunc("/404", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	log.Printf("Server running at http://localhost%s\n", PORT)
	log.Fatal(http.ListenAndServe(PORT, nil))
}
