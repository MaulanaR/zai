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

/*
Anda adalah asisten AI yang bertugas menganalisis pesan dan menentukan kebutuhan data API.

INPUT PESAN: %s

TUGAS:
1. Analisis pesan untuk menentukan apakah membutuhkan data baru dari API
2. Identifikasi endpoint yang sesuai
3. Tentukan parameter yang diperlukan

ENDPOINTS YANG TERSEDIA:
1. contacts
   - Customer queries: {"is_customer": "true", "is_skip_pagination": "true"}
   - Vendor queries: {"is_vendor": "true", "is_skip_pagination": "true"}
   - Employee queries: {"is_employee": "true", "is_skip_pagination": "true"}
2. sales_invoices
   - Default params: {"is_skip_pagination": "true"}
3. products
   - Default params: {"is_skip_pagination": "true"}

PARAMETER TANGGAL:
- Rentang tanggal: date[$gte], date[$lte]
- Tanggal spesifik: date[$eq]

RULES:
1. Jika pertanyaan bisa dijawab tanpa data API, kembalikan endpoint: "none"
2. Jika membutuhkan data contact, tentukan tipe (customer/vendor/employee)
3. Jika ada filter tanggal, sertakan dalam parameter
4. Selalu sertakan "is_skip_pagination": "true" untuk semua query

FORMAT RESPONSE (JSON only):
{
    "endpoint": string,
    "params": {
        key: value
    }
}
*/

// getAPIDecision menggunakan Claude untuk menentukan endpoint yang sesuai
func (bot *ChatBot) getAPIDecision(message string) (*APIDecision, error) {
	prompt := fmt.Sprintf(`%s`, message)

	claudeResp, err := bot.askClaudeJson(prompt)
	if err != nil {
		return nil, err
	}

	var decision APIDecision
	if err := json.Unmarshal([]byte(claudeResp), &decision); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return &decision, nil
}

func extractJSON(input string) (string, error) {
	// Mencari awal JSON yang ditandai dengan "json" atau tanda ```json
	start := strings.Index(input, "```json")
	if start == -1 {
		start = strings.Index(input, "json\n")
	}
	if start != -1 {
		// Lewati penanda json
		if strings.HasPrefix(input[start:], "```json") {
			start += 7 // panjang "```json"
		} else {
			start += 5 // panjang "json\n"
		}
	} else {
		return "", nil
	}

	// Mencari akhir JSON
	end := strings.Index(input[start:], "```")
	if end == -1 {
		// Jika tidak ada ```, ambil sampai akhir string
		end = len(input[start:])
	}

	// Ekstrak JSON string
	jsonStr := strings.TrimSpace(input[start : start+end])

	return jsonStr, nil
}

// ProcessMessage dengan logika yang diperbarui
func (bot *ChatBot) ProcessMessage(message string) *ZahirResponse {
	decision, err := bot.getAPIDecision(message)
	if err != nil {
		return &ZahirResponse{
			Status:  "error",
			Message: fmt.Sprintf("Gagal menentukan kebutuhan data: %v", err),
		}
	}

	// Jika tidak memerlukan data baru
	if decision.Endpoint == "" || decision.Endpoint == "null" {
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

	// Jika memerlukan data baru
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
}

// Fungsi helper lainnya (getDataFromAPI, askClaude, interpretAPIResponse) tetap sama
func (bot *ChatBot) getDataFromAPI(decision *APIDecision) (*ZahirResponse, error) {
	params := url.Values{}
	for key, value := range decision.Params {
		params.Add(key, value)
	}

	urlStr := fmt.Sprintf("%s/%s", BaseAPIURL, decision.Endpoint)
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

func (bot *ChatBot) askClaudeJson(prompt string) (string, error) {
	claudeReq := map[string]interface{}{
		"user":  "maulana",
		"model": ModelAI,
		// "model": "gemma2-9b-it",
		"messages": []map[string]string{
			{
				"role":    "user",
				"name":    "maulana",
				"content": prompt,
			},
			{
				"role": "system",
				"content": fmt.Sprint(`Analyze the message and determine if new API data is required or if it can be answered using previously provided data:
1. If new data is needed:
Specify the appropriate endpoint and any required parameters. Use the endpoints below as per the query type:

Customer queries: contacts with params={"is_customer":"true", "is_skip_pagination":"true"}
Vendor queries: contacts with params={"is_vendor":"true", "is_skip_pagination":"true"}
Employee queries: contacts with params={"is_employee":"true", "is_skip_pagination":"true"}
Sales Invoice queries: sales_invoices with params={"is_skip_pagination":"true"} with avaiable field for queries : customer.name, status, payment_status, date, time, number, description, customer.name, currency.name, subtotal, total_discount, total_discount_percentage, subtotal_before_tax, total_tax, total_cash_amount, total_other, total_amount, total_payment, receivable, balance
Product queries: products with params={"is_skip_pagination":"true"}
Purchase Invoice queries: purchases_invoices with params={"is_skip_pagination":"true"}
Profit and loss/profit/laba rugi queries: dashboards/profit_loss_simple
Balance sheet/neraca queries: dashboards/balance_sheet_simple
Daily sales data queries : dashboards/daily_sales
For date filtering, use date[$gte], date[$lte], or date[$eq] with the format YYYY-MM-DD.

2. If data has already been provided:
Use the previously shared information and respond with "endpoint": "null". 

Respond only with the JSON decision object, without any other string:
{"endpoint": "endpoint_name","params": {"param_key":"param_value"}}`),
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

	fmt.Println("RESP AI ")
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
		"user":  "maulana",
		"model": ModelAI,
		// "model": "gemma2-9b-it",
		"messages": []map[string]string{
			{
				"role":    "user",
				"name":    "maulana",
				"content": prompt,
			},
			{
				"role":    "system",
				"content": "Answer this question using existing information, without needing new API data. Be concise and direct. Format any price values in Rupiah currency.",
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
		"user":  "maulana",
		"model": ModelAI,
		// "model": "gemma2-9b-it",
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
Response with human chat
if answer is nominal harga, then format it to rupiah currency.
do not tell that the answer is from api.

API Response: %s
If user want response as chart, then use highchart.
Respond in Bahasa Indonesia and in format HTML`,
					endpoint, apiData),
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
