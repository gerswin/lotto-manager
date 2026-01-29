package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// TelegramAdminAuth verifica que el usuario sea un admin autorizado de Telegram
func TelegramAdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Obtener initData del header o query param
		initData := r.Header.Get("X-Telegram-Init-Data")
		if initData == "" {
			initData = r.URL.Query().Get("tg_init_data")
		}
		if initData == "" {
			// Intentar obtener de cookie
			cookie, err := r.Cookie("tg_init_data")
			if err == nil {
				initData = cookie.Value
			}
		}

		if initData == "" {
			http.Error(w, "Acceso denegado: No autorizado", http.StatusUnauthorized)
			return
		}

		// Verificar y extraer usuario
		user, valid := validateTelegramInitData(initData)
		if !valid {
			http.Error(w, "Acceso denegado: Datos inválidos", http.StatusUnauthorized)
			return
		}

		// Verificar si el usuario está en la lista de admins
		adminIDs := os.Getenv("ADMIN_TELEGRAM_IDS")
		if !isAdmin(user.ID, adminIDs) {
			http.Error(w, "Acceso denegado: No eres administrador", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
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
