package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/MaulanaR/zai/model"
	"github.com/MaulanaR/zai/prompt"
	"github.com/joho/godotenv"
	"grest.dev/grest"
)

// Konfigurasi
const (
	BaseAPIURL = "https://go.zahironline.com/api/v2"
)

var (
	BearerToken   string
	Slug          string
	APIKey        string
	APIUrl        string
	Port          string
	ModelAI       string
	CacheChat     CacheEntry
	CacheData     CacheEntry
	VisionAPIKey  string
	VisionAPIUrl  string
	VisionModelAI string
)

func Init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	BearerToken = os.Getenv("BEARER_TOKEN")
	Slug = os.Getenv("SLUG")
	APIKey = os.Getenv("API_KEY")
	APIUrl = os.Getenv("API_URL")
	ModelAI = os.Getenv("MODEL_AI")

	VisionAPIKey = os.Getenv("VISION_API_KEY")
	VisionAPIUrl = os.Getenv("VISION_API_URL")
	VisionModelAI = os.Getenv("VISION_MODEL_AI")
	Port = os.Getenv("PORT")
}

// CacheEntry menyimpan data history chat
type CacheEntry struct {
	Data string
}

// ChatBot struktur untuk menyimpan konfigurasi chatbot
type ChatBot struct {
	client    *http.Client
	cacheChat *CacheEntry
	cacheData *CacheEntry
}

// Struktur lainnya tetap sama
type WebhookRequest struct {
	Message     string `json:"message"`
	Image       string `json:"image"`
	BearerToken string `json:"bearer_token"`
	Slug        string `json:"slug"`
}

type ZahirResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"results"`
	Error   interface{} `json:"error"`
}

type APIDecision struct {
	Input    bool           `json:"input"`
	Endpoint string         `json:"endpoint"`
	Type     string         `json:"type"`
	Params   map[string]any `json:"params"`
}

func NewChatBot() *ChatBot {
	return &ChatBot{
		client:    &http.Client{},
		cacheChat: &CacheChat,
		cacheData: &CacheData,
	}
}

func (bot *ChatBot) getAPIDecisionEndpointCategory(message string) (*APIDecision, error) {
	claudeResp, err := bot.askClaudeJson(message, prompt.SystemMSG())
	if err != nil {
		return nil, err
	}

	// Remove ```json ``` from the response if present
	claudeResp = strings.TrimPrefix(claudeResp, "```json")
	claudeResp = strings.TrimSuffix(claudeResp, "```")

	var decision APIDecision
	if err := json.Unmarshal([]byte(claudeResp), &decision); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return &decision, nil
}

// getAPIDecision menggunakan Claude untuk menentukan endpoint yang sesuai
func (bot *ChatBot) getAPIDecision(message string, systemPrompt string) (*APIDecision, error) {
	claudeResp, err := bot.askClaudeJson(message, systemPrompt)
	if err != nil {
		return nil, err
	}

	var decision APIDecision
	if err := json.Unmarshal([]byte(claudeResp), &decision); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return &decision, nil
}

// Add new function for Vision AI
func (bot *ChatBot) askVisionAI(imageBase64, prompt string) (string, error) {
	message := []map[string]interface{}{
		{
			"role": "user",
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": prompt,
				},
				{
					"type": "image_url",
					"image_url": map[string]string{
						"url": imageBase64,
					},
				},
			},
		},
	}

	visionReq := map[string]interface{}{
		"model":      VisionModelAI,
		"messages":   message,
		"max_tokens": 3500,
	}

	jsonData, err := json.Marshal(visionReq)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", VisionAPIUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+VisionAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := bot.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var visionResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&visionResp); err != nil {
		return "", err
	}

	if choices, ok := visionResp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("invalid response format from Vision AI")
}

// Modify ProcessMessage to accept dynamic BearerToken and Slug
func (bot *ChatBot) ProcessMessage(req WebhookRequest) *ZahirResponse {
	// Use dynamic BearerToken and Slug if provided, else fallback to env
	bearerToken := req.BearerToken
	if bearerToken == "" {
		bearerToken = BearerToken
	}
	slug := req.Slug
	if slug == "" {
		slug = Slug
	}

	// If image exists, process with Vision AI first
	if req.Image != "" {
		visionResponse, err := bot.askVisionAI(req.Image, `Analisa gambar lalu berikan data apa yang tampil, tentukan berdasarkan aturan ini : 
		<available_fields>
				<sales_invoices>
					- customer.name
					- payment_status (enum: open, paid)
					- date
					- time
					- number
					- description
					- currency.name
					- subtotal
					- total_discount
					- subtotal_before_tax
					- total_tax
					- total_cash_amount
					- total_amount
					- total_payment
					- line_items (product information)
				</sales_invoices>

				<purchases_invoices>
					- description
					- date
					- time
					- number
					- note
					- total_amount
				</purchases_invoices>

				<products>
					- code
					- name
					- description
					- category.name
					- catalog.name
					- quantity.on_hand
					- quantity.on_order
					- quantity.on_hold
					- unit_price_gross
					- unit_price
					- unit_cogs
				</products>

				<contacts>
					- name
					- note
					- national_id_number
					- tax_id_number
					- is_customer
					- is_supplier
					- is_employee
					- is_salesman
					- is_active
					- customer_category.name
				</contacts>
			</available_fields> jika tidak ada informasi relevan berarti berikan informasi barang tersebut untuk nantinya di input, gunakan aturan products`)
		if err != nil {
			return &ZahirResponse{
				Status:  "error",
				Message: fmt.Sprintf("Failed to analyze image: %v", err),
			}
		}

		// Combine vision analysis with user message
		if req.Message != "" {
			req.Message = fmt.Sprintf("Context from image: %s\n\nUser question: %s", visionResponse, req.Message)
		} else {
			req.Message = visionResponse
		}
	}

	// Continue with existing logic for processing message
	endCat, err := bot.getAPIDecisionEndpointCategory(req.Message)
	if err != nil {
		return &ZahirResponse{
			Status:  "error",
			Message: fmt.Sprintf("Gagal menentukan kebutuhan kategori API: %v", err),
		}
	}

	if endCat.Input {
		if endCat.Type == "kontak" || endCat.Type == "customer" || endCat.Type == "supplier" || endCat.Type == "employee" || endCat.Type == "products" {
			zRes, err := bot.postToAPI(endCat.Endpoint, endCat.Params, bearerToken, slug)
			if err != nil {
				return &ZahirResponse{
					Status:  "error",
					Message: fmt.Sprintf("Gagal input via api: %v", err),
				}
			}

			// jika errornya ada, maka balikan ke ai
			if zRes.Error != nil {
				rs, err := bot.askAI(req.Message, prompt.GenerateForm())
				zRes.Status = "OK"
				zRes.Message = rs
				zRes.Error = nil
				if err != nil {
					return &ZahirResponse{
						Status:  "Error",
						Message: "Gagal generate form",
					}
				}
			}
			return &zRes
		}

		// reset
		CacheData = CacheEntry{}
	} else {
		if endCat.Endpoint != "" && endCat.Endpoint != "null" {
			// memerlukan data baru
			apiResp, err := bot.getDataFromAPIWithAuth(endCat, bearerToken, slug)
			if err != nil {
				return &ZahirResponse{
					Status:  "error",
					Message: fmt.Sprintf("Gagal mengambil data: %v", err),
				}
			}
			fmt.Println("===== RESPON FROM API =====")
			fmt.Println(apiResp.Data)
			fmt.Println("===== END RESPON FROM API =====")

			interpretation, err := bot.interpretAPIResponse(req.Message, apiResp, endCat.Endpoint)
			if err != nil {
				return apiResp
			}

			// add to cache
			CacheChat = CacheEntry{interpretation}

			return &ZahirResponse{
				Status:  "OK",
				Message: interpretation,
			}
		} else {
			interpretation, err := bot.interpretMessage(req.Message)
			if err != nil {
				return &ZahirResponse{
					Status:  "error",
					Message: fmt.Sprintf("Gagal menginterpretasi pesan: %v", err),
				}
			}

			// add to cache
			CacheChat = CacheEntry{interpretation}

			return &ZahirResponse{
				Status:  "OK",
				Message: interpretation,
			}
		}
	}

	return &ZahirResponse{
		Status:  "error",
		Message: fmt.Sprintf("Gagal menentukan kebutuhan ANDA: %v", err),
	}
}

// Fungsi helper lainnya (getDataFromAPI, askClaude, interpretAPIResponse) tetap sama
func (bot *ChatBot) getDataFromAPI(decision *APIDecision) (*ZahirResponse, error) {
	params := url.Values{}
	for key, value := range decision.Params {
		params.Add(key, fmt.Sprintf("%v", value))
	}

	urlStr := fmt.Sprintf("%s/%s", BaseAPIURL, strings.TrimSpace(decision.Endpoint))
	if len(params) > 0 {
		urlStr = fmt.Sprintf("%s?%s", urlStr, params.Encode())
	}

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", BearerToken))
	req.Header.Add("slug", Slug)
	req.Header.Add("Content-Type", "application/json")

	fmt.Println("========")
	fmt.Printf("Request URL: %s\n", urlStr)
	fmt.Printf("Request Headers: %v\n", req.Header)
	fmt.Printf("Request Body: %s\n", params.Encode())
	fmt.Println("========")

	resp, err := bot.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var zahirResp ZahirResponse
	switch decision.Endpoint {
	case "contacts":
		r := model.ContactResp{}
		if err := grest.NewJSON(bodyBytes, true).ToFlat().Unmarshal(&r); err != nil {
			return nil, err
		}
		zahirResp.Data = r.Data
	case "sales_invoices":
		r := model.SalesInvoicesResp{}
		if err := grest.NewJSON(bodyBytes, true).ToFlat().Unmarshal(&r); err != nil {
			return nil, err
		}
		zahirResp.Data = r.Data
	case "products":
		r := model.ProductResp{}
		if err := grest.NewJSON(bodyBytes, true).ToFlat().Unmarshal(&r); err != nil {
			return nil, err
		}
		zahirResp.Data = r.Data
	case "purchases_invoices":
		r := model.PurchaseInvResp{}
		if err := grest.NewJSON(bodyBytes, true).ToFlat().Unmarshal(&r); err != nil {
			return nil, err
		}
		zahirResp.Data = r.Data
	case "dashboards/daily_sales":
		var d interface{}
		if err := json.Unmarshal(bodyBytes, &d); err != nil {
			return nil, err
		}
		zahirResp.Data = d
	case "dashboards/balance_sheet_simple":
		var d interface{}
		if err := json.Unmarshal(bodyBytes, &d); err != nil {
			return nil, err
		}
		zahirResp.Data = d
	default:
		if err := json.Unmarshal(bodyBytes, &zahirResp); err != nil {
			var d interface{}
			if err := json.Unmarshal(bodyBytes, &d); err != nil {
				return nil, err
			}
			zahirResp.Data = d
		}
	}

	return &zahirResp, nil
}

// interpretMessage menangani pesan yang tidak memerlukan data baru
func (bot *ChatBot) interpretMessage(message string) (string, error) {
	prompt := message

	return bot.askClaudePlain(prompt)
}

func (bot *ChatBot) askAI(prompt string, systemPromt string) (string, error) {
	//build message with assistant
	message := []map[string]string{
		{
			"role":    "system",
			"content": systemPromt,
		},
	}

	// cache data
	if bot.cacheData.Data != "" {
		message = append(message, map[string]string{
			"role":    "system",
			"content": strings.NewReplacer("\n", " ", "\t", " ").Replace(bot.cacheData.Data),
		})
	}
	// cache chat
	if bot.cacheChat.Data != "" {
		message = append(message, map[string]string{
			"role": "assistant",
			// "name":    "maulana",
			"content": strings.NewReplacer("\n", " ", "\t", " ").Replace(bot.cacheChat.Data),
		})
	}
	message = append(message, map[string]string{
		"role":    "user",
		"content": strings.NewReplacer("\n", " ", "\t", " ").Replace(prompt),
	})

	claudeReq := map[string]interface{}{
		// "user":        "maulana",
		"model":       ModelAI,
		"temperature": 0,
		"top_p":       0.01,
		"max_tokens":  3500,
		"messages":    message,
		// "response_format": map[string]string{
		// 	"type": "json_object",
		// },
	}

	fmt.Println("==== REQ yang dikirim ke AI ====")
	fmt.Println(claudeReq)
	fmt.Println("==== END REQ yang dikirim ke AI ====")

	jsonData, err := json.Marshal(claudeReq)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", APIUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := bot.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var claudeResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return "", err
	}

	fmt.Println("RESP AI JSON ====")
	fmt.Println("claudeResp", claudeResp)

	if choices, ok := claudeResp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(interface{}); ok {
					return fmt.Sprintf("%s", content), nil
				}
			}
		}
	}

	return "", fmt.Errorf("invalid response format from Claude")
}

func (bot *ChatBot) askClaudeJson(prompt string, systemPromt string) (string, error) {
	//build message with assistant
	message := []map[string]string{
		{
			"role":    "system",
			"content": systemPromt,
		},
	}

	// cache data
	if bot.cacheData.Data != "" {
		message = append(message, map[string]string{
			"role":    "system",
			"content": strings.NewReplacer("\n", " ", "\t", " ").Replace(bot.cacheData.Data),
		})
	}
	// cache chat
	if bot.cacheChat.Data != "" {
		message = append(message, map[string]string{
			"role": "assistant",
			// "name":    "maulana",
			"content": strings.NewReplacer("\n", " ", "\t", " ").Replace(bot.cacheChat.Data),
		})
	}
	message = append(message, map[string]string{
		"role":    "user",
		"content": strings.NewReplacer("\n", " ", "\t", " ").Replace(prompt),
	})

	claudeReq := map[string]interface{}{
		// "user":        "maulana",
		"model":       ModelAI,
		"temperature": 0,
		"top_p":       0.01,
		"max_tokens":  3500,
		"messages":    message,
		// "response_format": map[string]string{
		// 	"type": "json_object",
		// },
	}

	fmt.Println("==== REQ yang dikirim ke AI ====")
	fmt.Println(claudeReq)
	fmt.Println("==== END REQ yang dikirim ke AI ====")

	jsonData, err := json.Marshal(claudeReq)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", APIUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := bot.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var claudeResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return "", err
	}

	fmt.Println("RESP AI JSON ====")
	fmt.Println("claudeResp", claudeResp)

	if choices, ok := claudeResp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("invalid response format from Claude")
}

func (bot *ChatBot) askClaudePlain(userMsg string) (string, error) {
	//build message with assistant
	message := []map[string]string{
		{
			"role":    "system",
			"content": prompt.GenerateResRule(),
		},
	}
	// cache data
	if bot.cacheData.Data != "" {
		message = append(message, map[string]string{
			"role": "system",
			// "name":    "maulana",
			"content": strings.NewReplacer("\n", " ", "\t", " ").Replace(bot.cacheData.Data),
		})
	}
	// cache chat
	if bot.cacheChat.Data != "" {
		message = append(message, map[string]string{
			"role": "assistant",
			// "name":    "maulana",
			"content": strings.NewReplacer("\n", " ", "\t", " ").Replace(bot.cacheChat.Data),
		})
	}
	message = append(message, map[string]string{
		"role":    "user",
		"content": strings.NewReplacer("\n", " ", "\t", " ").Replace(userMsg),
	})

	claudeReq := map[string]interface{}{
		// "user":        "maulana",
		"model":       ModelAI,
		"temperature": 0,
		"top_p":       0.01,
		"messages":    message,
		"max_tokens":  3500,
	}

	fmt.Println("==== REQ yang dikirim ke AI PLAIN====")
	fmt.Println(claudeReq)
	fmt.Println("==== END REQ yang dikirim ke AI PLAIN====")

	jsonData, err := json.Marshal(claudeReq)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", APIUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := bot.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var claudeResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return "", err
	}

	if choices, ok := claudeResp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("invalid response format from Claude, detail %v", claudeResp)
}

func (bot *ChatBot) askClaudeFromAPIRes(userMsg, endpoint, apiData string) (string, error) {
	//build message with assistant
	message := []map[string]string{
		{
			"role":    "system",
			"content": prompt.GenerateResRule(),
		},
	}
	// cache data
	if bot.cacheData.Data != "" {
		message = append(message, map[string]string{
			"role":    "system",
			"content": strings.NewReplacer("\n", " ", "\t", " ").Replace(bot.cacheData.Data),
		})
	}
	// cache chat
	if bot.cacheChat.Data != "" {
		message = append(message, map[string]string{
			"role":    "assistant",
			"content": strings.NewReplacer("\n", " ", "\t", " ").Replace(bot.cacheChat.Data),
		})
	}

	message = append(message, map[string]string{
		"role":    "user",
		"content": strings.NewReplacer("\n", " ", "\t", " ").Replace(userMsg),
	})

	claudeReq := map[string]interface{}{
		// "user":        "maulana",
		"model":       ModelAI,
		"temperature": 0,
		"top_p":       0.6,
		"messages":    message,
		"max_tokens":  3500,
	}

	fmt.Println("==== REQ yang dikirim ke AI gen after API ====")
	fmt.Println(claudeReq)
	fmt.Println("==== END REQ yang dikirim ke AI gen after API ====")

	jsonData, err := json.Marshal(claudeReq)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", APIUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := bot.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var claudeResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return "", err
	}

	fmt.Println("==== RESPONSE SETELAH OLAHAN API ====")
	fmt.Println(claudeResp)
	fmt.Println("==== END RESPONSE SETELAH OLAHAN API ====")
	if choices, ok := claudeResp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("invalid response format from Claude, detail %v", claudeResp)
}

func (bot *ChatBot) interpretAPIResponse(userMessage string, apiResp *ZahirResponse, endpoint string) (string, error) {
	apiData, err := json.Marshal(apiResp)
	if err != nil {
		return "", err
	}

	prompt := userMessage

	// add to cache
	if string(apiData) != "" || string(apiData) != "[]" {
		CacheData = CacheEntry{"data " + endpoint + ":" + string(apiData)}
	}

	return bot.askClaudeFromAPIRes(prompt, endpoint, string(apiData))
}

// Main dan webhook handler tetap sama
func webhookHandler(bot *ChatBot) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req WebhookRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		response := bot.ProcessMessage(req)
		response.Message = strings.ReplaceAll(response.Message, "```html", "")
		response.Message = strings.ReplaceAll(response.Message, "```", "")
		response.Message = strings.ReplaceAll(response.Message, "``json", "")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func main() {
	Init()
	bot := NewChatBot()

	http.HandleFunc("/webhook", webhookHandler(bot))

	// Serve the index.html file and inject WEBHOOK_URL from env
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		webhookUrl := os.Getenv("WEBHOOK_URL")
		if webhookUrl == "" {
			webhookUrl = "http://127.0.0.1:8991/webhook"
		}
		indexBytes, err := os.ReadFile("index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		// Inject <script>window.WEBHOOK_URL = "...";</script> after <head>
		htmlStr := string(indexBytes)
		inject := fmt.Sprintf(`<script>window.WEBHOOK_URL = "%s";</script>`, webhookUrl)
		htmlStr = strings.Replace(htmlStr, "<head>", "<head>\n    "+inject, 1)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlStr))
	})

	log.Printf("Server starting on port %s", Port)
	if err := http.ListenAndServe(Port, nil); err != nil {
		log.Fatal(err)
	}
}

// Tambahkan versi baru getDataFromAPI yang menerima bearerToken dan slug
func (bot *ChatBot) getDataFromAPIWithAuth(decision *APIDecision, bearerToken, slug string) (*ZahirResponse, error) {
	params := url.Values{}
	for key, value := range decision.Params {
		params.Add(key, fmt.Sprintf("%v", value))
	}

	urlStr := fmt.Sprintf("%s/%s", BaseAPIURL, strings.TrimSpace(decision.Endpoint))
	if len(params) > 0 {
		urlStr = fmt.Sprintf("%s?%s", urlStr, params.Encode())
	}

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	req.Header.Add("slug", slug)
	req.Header.Add("Content-Type", "application/json")

	fmt.Println("========")
	fmt.Printf("Request URL: %s\n", urlStr)
	fmt.Printf("Request Headers: %v\n", req.Header)
	fmt.Printf("Request Body: %s\n", params.Encode())
	fmt.Println("========")

	resp, err := bot.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var zahirResp ZahirResponse
	switch decision.Endpoint {
	case "contacts":
		r := model.ContactResp{}
		if err := grest.NewJSON(bodyBytes, true).ToFlat().Unmarshal(&r); err != nil {
			return nil, err
		}
		zahirResp.Data = r.Data
	case "sales_invoices":
		r := model.SalesInvoicesResp{}
		if err := grest.NewJSON(bodyBytes, true).ToFlat().Unmarshal(&r); err != nil {
			return nil, err
		}
		zahirResp.Data = r.Data
	case "products":
		r := model.ProductResp{}
		if err := grest.NewJSON(bodyBytes, true).ToFlat().Unmarshal(&r); err != nil {
			return nil, err
		}
		zahirResp.Data = r.Data
	case "purchases_invoices":
		r := model.PurchaseInvResp{}
		if err := grest.NewJSON(bodyBytes, true).ToFlat().Unmarshal(&r); err != nil {
			return nil, err
		}
		zahirResp.Data = r.Data
	case "dashboards/daily_sales":
		var d interface{}
		if err := json.Unmarshal(bodyBytes, &d); err != nil {
			return nil, err
		}
		zahirResp.Data = d
	case "dashboards/balance_sheet_simple":
		var d interface{}
		if err := json.Unmarshal(bodyBytes, &d); err != nil {
			return nil, err
		}
		zahirResp.Data = d
	default:
		if err := json.Unmarshal(bodyBytes, &zahirResp); err != nil {
			var d interface{}
			if err := json.Unmarshal(bodyBytes, &d); err != nil {
				return nil, err
			}
			zahirResp.Data = d
		}
	}

	return &zahirResp, nil
}

// Ubah postToAPI agar menerima bearerToken dan slug
func (bot *ChatBot) postToAPI(endpoint string, params map[string]any, bearerToken, slug string) (ZahirResponse, error) {
	zRes := ZahirResponse{}
	fmt.Println("==== POST PAYLOAD====")
	fmt.Println(params)
	fmt.Println("==== POST PAYLOAD ====")

	jsonData, err := json.Marshal(params)
	if err != nil {
		return zRes, err
	}

	req, err := http.NewRequest("POST", BaseAPIURL+"/"+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return zRes, err
	}

	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("slug", slug)
	req.Header.Set("Content-Type", "application/json")

	resp, err := bot.client.Do(req)
	if err != nil {
		return zRes, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&zRes); err != nil {
		return zRes, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		zRes.Status = "OK"
		zRes.Message = "Sukses input data"
	}

	return zRes, nil
}
