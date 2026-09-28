package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	kdmodel "apeadmin-gin/internal/plugin/builtin/kingdee/model"
	kdservice "apeadmin-gin/internal/plugin/builtin/kingdee/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WebhookHandler struct {
	db        *gorm.DB
	feishuSvc *kdservice.FeishuApprovalService
}

func NewWebhookHandler(db *gorm.DB) *WebhookHandler {
	return &WebhookHandler{
		db:        db,
		feishuSvc: kdservice.NewFeishuApprovalService(db),
	}
}

func (h *WebhookHandler) HandleFeishuCallback(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil || len(rawBody) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": -1, "msg": "empty body"})
		return
	}

	encryptKey := h.getConfigValue("feishu_encrypt_key")

	// 1. 可选签名校验
	if encryptKey != "" {
		signature := c.GetHeader("X-Lark-Signature")
		timestamp := c.GetHeader("X-Lark-Request-Timestamp")
		nonce := c.GetHeader("X-Lark-Request-Nonce")
		if !h.verifySignature(encryptKey, signature, timestamp, nonce, rawBody) {
			log.Printf("[FeishuWebhook] 签名校验失败")
			c.JSON(http.StatusOK, gin.H{"code": -1, "msg": "invalid signature"})
			return
		}
	}

	// 2. 解密载荷 (如开启解密)
	payload, err := h.decryptIfNeeded(encryptKey, rawBody)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -1, "msg": "decrypt failed"})
		return
	}

	// 3. url_verification 挑战处理
	if pType, ok := payload["type"].(string); ok && pType == "url_verification" {
		challenge, _ := payload["challenge"].(string)
		log.Printf("[FeishuWebhook] url_verification 响应 challenge: %s", challenge)
		c.JSON(http.StatusOK, gin.H{"challenge": challenge})
		return
	}

	// 4. 解析事件
	var event map[string]interface{}
	eventType := ""
	if schema, ok := payload["schema"].(string); ok && schema == "2.0" {
		if header, ok := payload["header"].(map[string]interface{}); ok {
			eventType, _ = header["event_type"].(string)
		}
		event, _ = payload["event"].(map[string]interface{})
	} else if pType, ok := payload["type"].(string); ok && pType == "event_callback" {
		eventType, _ = payload["event_type"].(string)
		event, _ = payload["event"].(map[string]interface{})
	}

	if event == nil || eventType == "" || !strings.Contains(strings.ToLower(eventType), "approval") {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ignored"})
		return
	}

	instanceCode, _ := event["instance_code"].(string)
	status, _ := event["status"].(string)

	if instanceCode == "" || status == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "missing fields"})
		return
	}

	log.Printf("[FeishuWebhook] 收到审批事件 | instance_code=%s status=%s", instanceCode, status)
	msg, err := h.feishuSvc.HandleCallback(instanceCode, status)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "callback_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": msg})
}

func (h *WebhookHandler) verifySignature(key, signature, timestamp, nonce string, rawBody []byte) bool {
	if signature == "" || timestamp == "" || nonce == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(key))
	content := timestamp + "\n" + nonce + "\n" + string(rawBody)
	mac.Write([]byte(content))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return expected == signature
}

func (h *WebhookHandler) decryptIfNeeded(key string, rawBody []byte) (map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return nil, err
	}

	encStr, ok := payload["encrypt"].(string)
	if !ok || encStr == "" {
		return payload, nil
	}

	if key == "" {
		log.Printf("[FeishuWebhook] 收到加密事件但未配置 feishu_encrypt_key")
		return payload, nil
	}

	cipherText, err := base64.StdEncoding.DecodeString(encStr)
	if err != nil || len(cipherText) <= 16 {
		return payload, nil
	}

	iv := cipherText[:16]
	data := cipherText[16:]

	keyBytes := []byte(key)
	if len(keyBytes) != 16 && len(keyBytes) != 24 && len(keyBytes) != 32 {
		h := sha256.Sum256(keyBytes)
		keyBytes = h[:]
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plainText := make([]byte, len(data))
	mode.CryptBlocks(plainText, data)

	// PKCS7 unpad
	if len(plainText) > 0 {
		padLen := int(plainText[len(plainText)-1])
		if padLen > 0 && padLen <= aes.BlockSize {
			plainText = plainText[:len(plainText)-padLen]
		}
	}

	var decryptedPayload map[string]interface{}
	if err := json.Unmarshal(plainText, &decryptedPayload); err != nil {
		return nil, err
	}
	return decryptedPayload, nil
}

func (h *WebhookHandler) getConfigValue(key string) string {
	var cfg kdmodel.KdConfig
	if err := h.db.Where("key = ?", key).First(&cfg).Error; err == nil {
		return strings.TrimSpace(cfg.Value)
	}
	return ""
}
