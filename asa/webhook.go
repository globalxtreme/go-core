package xtremeasa

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	xtremeres "github.com/globalxtreme/go-core/v2/response"
)

// Environment yang wajib ada di service pemanggil:
//
//	ASA_WEBHOOK_HOST    domain ASA beserta skemanya, mis. https://asa.globalxtreme-solutions.net
//	ASA_WEBHOOK_TOKEN   ASA Token milik service ini (menu Services di ASA). Tanpa token ini webhook ditolak
//
// Opsional:
//
//	ASA_WEBHOOK_TIMEOUT batas waktu panggilan dalam detik (default 10)
const (
	// Prefix URL webhook ASA; path node Webhook disambung sesudahnya dengan satu "/"
	ASA_WEBHOOK_PREFIX = "/api/webhook/v1/asa-automations"

	ASA_WEBHOOK_DEFAULT_TIMEOUT = 10 * time.Second

	// Batas isi jawaban yang dibaca, supaya jawaban yang sangat besar tidak memenuhi memori
	ASA_WEBHOOK_MAX_RESPONSE_SIZE = 10 << 20

	// Header metadata eksekusi; dicatat ASA pada eksekusi dan tidak diteruskan ke $json.headers workflow
	ASA_WEBHOOK_HEADER_DESCRIPTION     = "X-ASA-Description"
	ASA_WEBHOOK_HEADER_CREATED_BY      = "X-ASA-Created-By"
	ASA_WEBHOOK_HEADER_CREATED_BY_NAME = "X-ASA-Created-By-Name"
)

// Webhook memulai workflow ASA lewat node Webhook (trigger) dari service mana pun.
//
//	hook := xtremeasa.Webhook{
//		Path:          "wo-created",
//		Data:          map[string]interface{}{"woNumber": "WO-2026-00123"},
//		Description:   "WO-2026-00123 dibuat",
//		CreatedBy:     employee.ID,
//		CreatedByName: employee.FullName,
//	}
//	res := hook.Push() // panggil SETELAH transaction commit
//	res.ExecutionUUID  // UUID eksekusi yang dibuat ASA
//
// Domain ASA dan token selalu dibaca dari environment; pemanggil hanya mengisi Path dan, bila ada, Data, Query,
// Headers, Description, CreatedBy, dan CreatedByName. Node Webhook di ASA harus memakai Authentication "GX Service" dan memilih service pemilik token.
//
// Satu panggilan memulai satu workflow (path + method unik di ASA). Untuk memulai beberapa workflow dari satu
// kejadian, arahkan path ke workflow dispatcher yang memanggil workflow lain dengan node Execute Sub-workflow.
//
// Panggilan tidak diulang otomatis: setiap panggilan yang diterima membuat satu eksekusi baru.
type Webhook struct {
	// Path node Webhook tanpa "/" di depan, mis. "wo-created". Wajib. Slash ditambahkan otomatis oleh package;
	// slash di awal atau akhir yang ikut tertulis dibuang, sehingga hasilnya tetap ".../wo-created"
	Path string

	// Method node Webhook. Default POST
	Method string

	// Data dikirim sebagai body JSON; di workflow dibaca dengan $json.body
	Data interface{}

	// Query dikirim sebagai query string; di workflow dibaca dengan $json.query
	Query url.Values

	// Headers tambahan; di workflow dibaca dengan $json.headers (nama huruf kecil).
	// Header Authorization selalu diisi ASA Token dan tidak bisa diganti dari sini
	Headers map[string]string

	// Description keterangan eksekusi, tampil di detail eksekusi ASA. Opsional, maks. 500 karakter
	Description string

	// CreatedBy ID karyawan yang memicu eksekusi, maks. 45 karakter. Opsional; bila CreatedBy dan CreatedByName
	// kosong, ASA mencatat pemilik workflow sebagai pemicu
	CreatedBy string

	// CreatedByName nama karyawan yang memicu eksekusi, maks. 250 karakter. Opsional
	CreatedByName string
}

// WebhookResult adalah jawaban ASA atas satu panggilan webhook
type WebhookResult struct {
	StatusCode int
	Header     http.Header

	// ExecutionUUID adalah result.executionUuid dari jawaban ASA, sudah diurai dari Body sehingga cukup dibaca
	// res.ExecutionUUID. Terisi bila node Webhook memakai Respond "Immediately" (bawaan); kosong pada mode lain
	ExecutionUUID string

	// Body adalah isi jawaban apa adanya. Pada Respond "When last node finishes" dan
	// "Using Respond to Webhook node" isinya ditentukan workflow
	Body []byte
}

// JSON mengurai Body ke target
func (result WebhookResult) JSON(target interface{}) error {
	return json.Unmarshal(result.Body, target)
}

// Push memanggil webhook dan panic (xtremeres.ErrXtremeAPI) bila gagal, seperti Push RabbitMQ.
// Pakai TryPush bila kegagalan memulai workflow tidak boleh menggagalkan proses pemanggil.
func (hook Webhook) Push() WebhookResult {
	result, err := hook.TryPush()
	if err != nil {
		xtremeres.ErrXtremeAPI(err.Error())
	}

	return result
}

// TryPush memanggil webhook dan mengembalikan error bila panggilan gagal atau ASA menjawab status di luar 2xx.
// Pada status di luar 2xx, result tetap berisi jawaban ASA.
func (hook Webhook) TryPush() (WebhookResult, error) {
	var result WebhookResult

	// Slash di awal atau akhir dibuang; package yang menyambungnya ke prefix dengan satu "/"
	hook.Path = strings.Trim(strings.TrimSpace(hook.Path), "/")

	req, err := hook.request()
	if err != nil {
		return result, err
	}

	client := &http.Client{Timeout: hook.timeout()}

	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("call ASA webhook %s: %w", hook.Path, err)
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Header = resp.Header

	result.Body, err = io.ReadAll(io.LimitReader(resp.Body, ASA_WEBHOOK_MAX_RESPONSE_SIZE))
	if err != nil {
		return result, fmt.Errorf("read ASA webhook %s response: %w", hook.Path, err)
	}

	// Amplop standar GX: sukses berisi result.executionUuid, error berisi status.message
	var envelope struct {
		Status struct {
			Message     string `json:"message"`
			InternalMsg string `json:"internalMsg"`
		} `json:"status"`
		Result struct {
			ExecutionUUID string `json:"executionUuid"`
		} `json:"result"`
	}
	_ = json.Unmarshal(result.Body, &envelope)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		message := envelope.Status.Message
		if envelope.Status.InternalMsg != "" {
			message += ". Internal-Msg: " + envelope.Status.InternalMsg
		}
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}

		return result, fmt.Errorf("ASA webhook %s responded with status %d: %s", hook.Path, resp.StatusCode, message)
	}

	result.ExecutionUUID = envelope.Result.ExecutionUUID

	return result, nil
}

/** --- UNEXPORTED FUNCTIONS --- */

func (hook Webhook) request() (*http.Request, error) {
	path := hook.Path
	if path == "" {
		return nil, errors.New("ASA webhook path is required")
	}

	host := strings.TrimRight(os.Getenv("ASA_WEBHOOK_HOST"), "/")
	if host == "" {
		return nil, errors.New("Please set ASA_WEBHOOK_HOST environment variable")
	}

	token := os.Getenv("ASA_WEBHOOK_TOKEN")
	if token == "" {
		return nil, errors.New("Please set ASA_WEBHOOK_TOKEN environment variable")
	}

	target := host + ASA_WEBHOOK_PREFIX + "/" + path
	if query := hook.Query.Encode(); query != "" {
		target += "?" + query
	}

	method := strings.ToUpper(hook.Method)
	if method == "" {
		method = http.MethodPost
	}

	var body io.Reader
	if hook.Data != nil {
		payload, err := json.Marshal(hook.Data)
		if err != nil {
			return nil, fmt.Errorf("marshal ASA webhook %s payload: %w", hook.Path, err)
		}

		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, target, body)
	if err != nil {
		return nil, fmt.Errorf("create ASA webhook %s request: %w", hook.Path, err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	for name, value := range hook.Headers {
		req.Header.Set(name, value)
	}

	// Metadata eksekusi; bila diisi, menimpa header bernama sama dari Headers
	metadata := [][2]string{
		{ASA_WEBHOOK_HEADER_DESCRIPTION, hook.Description},
		{ASA_WEBHOOK_HEADER_CREATED_BY, hook.CreatedBy},
		{ASA_WEBHOOK_HEADER_CREATED_BY_NAME, hook.CreatedByName},
	}
	for _, header := range metadata {
		if value := headerValue(header[1]); value != "" {
			req.Header.Set(header[0], value)
		}
	}

	// Selalu paling akhir, supaya Headers dari pemanggil tidak menimpa token
	req.Header.Set("Authorization", "Bearer "+token)

	return req, nil
}

func (hook Webhook) timeout() time.Duration {
	if seconds, err := strconv.Atoi(os.Getenv("ASA_WEBHOOK_TIMEOUT")); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	return ASA_WEBHOOK_DEFAULT_TIMEOUT
}

// headerValue: nilai satu baris untuk header; baris baru (mis. pada Description) diganti spasi
func headerValue(value string) string {
	return strings.TrimSpace(strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(value))
}
