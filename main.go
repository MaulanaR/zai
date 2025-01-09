package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"os"

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
	BearerToken string
	Slug        string
	APIKey      string
	APIUrl      string
	Port        string
	ModelAI     string
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
	Port = os.Getenv("PORT")
}

// EndpointConfig mendefinisikan konfigurasi untuk setiap endpoint
type EndpointConfig struct {
	Endpoint      string
	BaseParams    map[string]string
	Keywords      []string
	CacheDuration time.Duration
}

// APIEndpoints menyimpan konfigurasi semua endpoint yang tersedia
var APIEndpoints = []EndpointConfig{
	{
		Endpoint: "contacts",
		BaseParams: map[string]string{
			"is_customer":        "true",
			"is_skip_pagination": "true",
		},
		Keywords:      []string{"customer", "pelanggan", "pembeli"},
		CacheDuration: 15 * time.Minute,
	},
	{
		Endpoint: "contacts",
		BaseParams: map[string]string{
			"is_vendor":          "true",
			"is_skip_pagination": "true",
		},
		Keywords:      []string{"vendor", "supplier", "pemasok"},
		CacheDuration: 15 * time.Minute,
	},
	{
		Endpoint: "contacts",
		BaseParams: map[string]string{
			"is_employee":        "true",
			"is_skip_pagination": "true",
		},
		Keywords:      []string{"karyawan", "pegawai", "employee"},
		CacheDuration: 15 * time.Minute,
	},
	{
		Endpoint: "sales_invoices",
		BaseParams: map[string]string{
			"is_skip_pagination": "true",
		},
		Keywords:      []string{"sales", "invoice", "penjualan", "faktur"},
		CacheDuration: 5 * time.Minute,
	},
	{
		Endpoint: "products",
		BaseParams: map[string]string{
			"is_skip_pagination": "true",
		},
		Keywords:      []string{"product", "produk", "barang"},
		CacheDuration: 10 * time.Minute,
	},
}

// CacheEntry menyimpan data cache beserta waktu kadaluarsanya
type CacheEntry struct {
	Data      *ZahirResponse
	ExpiresAt time.Time
}

// ChatBot struktur untuk menyimpan konfigurasi chatbot
type ChatBot struct {
	client *http.Client
	cache  map[string]CacheEntry
}

// Struktur lainnya tetap sama
type WebhookRequest struct {
	Message string `json:"message"`
}

type ZahirResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"results"`
}

type APIDecision struct {
	Endpoint string            `json:"endpoint"`
	Params   map[string]string `json:"params"`
}

func NewChatBot() *ChatBot {
	return &ChatBot{
		client: &http.Client{},
		cache:  make(map[string]CacheEntry),
	}
}

func (bot *ChatBot) getAPIDecisionEndpointCategory(message string) (*APIDecision, error) {
	claudeResp, err := bot.askClaudeJson(message, prompt.PromptDetermineAPIEndpoint())
	if err != nil {
		return nil, err
	}

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

// ProcessMessage dengan logika yang diperbarui
func (bot *ChatBot) ProcessMessage(message string) *ZahirResponse {
	endCat, err := bot.getAPIDecisionEndpointCategory(message)
	if err != nil {
		return &ZahirResponse{
			Status:  "error",
			Message: fmt.Sprintf("Gagal menentukan kebutuhan kategori API: %v", err),
		}
	}

	if endCat.Endpoint != "" && endCat.Endpoint != "null" {
		// memerlukan data baru
		systemPrompt := ""

		switch endCat.Endpoint {
		case "sales_invoices":
			systemPrompt = prompt.PromptSalesInvoiceRules()
		case "purchases_invoices":
			systemPrompt = prompt.PromptPurchaseInvoiceRules()
		case "products":
			systemPrompt = prompt.PromptProductRules()
		case "contacts":
			systemPrompt = prompt.PromptContactRules()
		default:
			systemPrompt = prompt.DefaultPromptRules()
		}

		decision := &APIDecision{}
		if systemPrompt != "" {
			decision, err = bot.getAPIDecision(message, systemPrompt)
			if err != nil {
				return &ZahirResponse{
					Status:  "error",
					Message: fmt.Sprintf("Gagal menentukan kebutuhan Endpoint API: %v", err),
				}
			}
		} else {
			decision.Params = map[string]string{
				"is_skip_pagination": "true",
			}
		}

		//set endpoint
		decision.Endpoint = endCat.Endpoint
		apiResp, err := bot.getDataFromAPI(decision)
		if err != nil {
			return &ZahirResponse{
				Status:  "error",
				Message: fmt.Sprintf("Gagal mengambil data: %v", err),
			}
		}
		fmt.Println("===== RESPON FROM API =====")
		fmt.Println(apiResp.Data)
		fmt.Println("===== END RESPON FROM API =====")

		interpretation, err := bot.interpretAPIResponse(message, apiResp, decision.Endpoint)
		if err != nil {
			return apiResp
		}

		return &ZahirResponse{
			Status:  "OK",
			Message: interpretation,
		}

	} else {
		interpretation, err := bot.interpretMessage(message)
		if err != nil {
			return &ZahirResponse{
				Status:  "error",
				Message: fmt.Sprintf("Gagal menginterpretasi pesan: %v", err),
			}
		}
		return &ZahirResponse{
			Status:  "OK",
			Message: interpretation,
		}
	}
}

// Fungsi helper lainnya (getDataFromAPI, askClaude, interpretAPIResponse) tetap sama
func (bot *ChatBot) getDataFromAPI(decision *APIDecision) (*ZahirResponse, error) {
	params := url.Values{}
	for key, value := range decision.Params {
		params.Add(key, value)
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

func (bot *ChatBot) askClaudeJson(prompt string, systemPromt string) (string, error) {
	claudeReq := map[string]interface{}{
		"user":        "maulana",
		"model":       ModelAI,
		"temperature": 0.2,
		"messages": []map[string]string{
			{
				"role":    "user",
				"name":    "maulana",
				"content": prompt,
			},
			{
				"role":    "system",
				"content": systemPromt,
			},
		},
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

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

func (bot *ChatBot) askClaudePlain(prompt string) (string, error) {
	claudeReq := map[string]interface{}{
		"user":        "maulana",
		"model":       ModelAI,
		"temperature": 0.2,
		"messages": []map[string]string{
			{
				"role":    "user",
				"name":    "maulana",
				"content": prompt,
			},
			{
				"role":    "system",
				"content": "Answer this question using existing information. Be concise and direct. Format any price values in Rupiah currency.",
			},
		},
	}

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

func (bot *ChatBot) askClaudeFromAPIRes(prompt, endpoint, apiData string) (string, error) {
	claudeReq := map[string]interface{}{
		"user":        "maulana",
		"model":       ModelAI,
		"temperature": 0.2,
		"messages": []map[string]string{
			{
				"role":    "user",
				"name":    "maulana",
				"content": prompt,
			},
			{
				"role": "system",
				"content": fmt.Sprintf(`Based on this API response from the %s endpoint, answer the user's question naturally.
Focus on directly answering what they asked about. Only include relevant information.
format all prices in Rupiah currency.
use Highcharts for chart if user want it.
respond in HTML format and in bahasa indonesia.

API Response: %s`, endpoint, apiData),
			},
		},
	}

	// 				"content": fmt.Sprintf(`Based on this API response from the %s endpoint, answer the user's question naturally.
	// Focus on directly answering what they asked about. Only include relevant information.
	// Response with human chat
	// if answer is nominal harga, then format it to rupiah currency.
	// do not tell that the answer is from api.

	// API Response: %s
	// If user want response as chart, then use highchart.
	// Respond in Bahasa Indonesia and in format HTML`,endpoint, apiData),

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

		response := bot.ProcessMessage(req.Message)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func main() {
	Init()
	bot := NewChatBot()

	http.HandleFunc("/webhook", webhookHandler(bot))

	log.Printf("Server starting on port %s", Port)
	if err := http.ListenAndServe(Port, nil); err != nil {
		log.Fatal(err)
	}
}
