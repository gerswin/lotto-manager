package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
)

type TelegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

// TelegramAdminAuth verifica autenticación por Telegram O BasicAuth
func TelegramAdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Método 1: Verificar BasicAuth (para acceso web normal)
		if checkBasicAuth(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Método 2: Verificar Telegram initData
		initData := r.Header.Get("X-Telegram-Init-Data")
		if initData == "" {
			initData = r.URL.Query().Get("tg_init_data")
		}
		if initData == "" {
			cookie, err := r.Cookie("tg_init_data")
			if err == nil {
				decoded, err := url.QueryUnescape(cookie.Value)
				if err == nil {
					initData = decoded
				}
			}
		}

		if initData != "" {
			user, valid := validateTelegramInitData(initData)
			if valid {
				adminIDs := os.Getenv("ADMIN_TELEGRAM_IDS")
				if isAdmin(user.ID, adminIDs) {
					log.Printf("Admin Telegram autenticado: %s (ID: %d)", user.FirstName, user.ID)
					next.ServeHTTP(w, r)
					return
				}
				log.Printf("Usuario Telegram no es admin: %d", user.ID)
			} else {
				log.Printf("initData inválido")
			}
		}

		// Si no hay autenticación válida, pedir BasicAuth
		w.Header().Set("WWW-Authenticate", `Basic realm="Lotto Admin"`)
		http.Error(w, "Acceso denegado: No autorizado", http.StatusUnauthorized)
	})
}

func checkBasicAuth(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Basic ") {
		return false
	}

	payload, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		return false
	}

	pair := strings.SplitN(string(payload), ":", 2)
	if len(pair) != 2 {
		return false
	}

	expectedPassword := os.Getenv("ADMIN_PASSWORD")
	return pair[0] == "admin" && pair[1] == expectedPassword
}

func validateTelegramInitData(initData string) (*TelegramUser, bool) {
	botToken := os.Getenv("TELEGRAM_TOKEN")
	if botToken == "" {
		return nil, false
	}

	// Parsear los parámetros
	params, err := url.ParseQuery(initData)
	if err != nil {
		return nil, false
	}

	// Obtener el hash
	hash := params.Get("hash")
	if hash == "" {
		return nil, false
	}

	// Crear data-check-string (ordenado alfabéticamente, sin hash)
	var keys []string
	for k := range params {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var dataCheckParts []string
	for _, k := range keys {
		dataCheckParts = append(dataCheckParts, k+"="+params.Get(k))
	}
	dataCheckString := strings.Join(dataCheckParts, "\n")

	// Calcular secret key: HMAC-SHA256(bot_token, "WebAppData")
	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))

	// Calcular hash: HMAC-SHA256(data_check_string, secret_key)
	h := hmac.New(sha256.New, secretKey.Sum(nil))
	h.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(h.Sum(nil))

	if calculatedHash != hash {
		return nil, false
	}

	// Extraer usuario
	userJSON := params.Get("user")
	if userJSON == "" {
		return nil, false
	}

	var user TelegramUser
	if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
		return nil, false
	}

	return &user, true
}

func isAdmin(userID int64, adminIDs string) bool {
	if adminIDs == "" {
		return false
	}

	ids := strings.Split(adminIDs, ",")
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if adminID, err := strconv.ParseInt(id, 10, 64); err == nil {
			if adminID == userID {
				return true
			}
		}
	}
	return false
}
