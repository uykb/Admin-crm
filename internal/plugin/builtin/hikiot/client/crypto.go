package client

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
)

// CalculateSignature 计算海康 Open API HMAC-SHA256 签名字符串
func CalculateSignature(appSecret, method, accept, contentType, headers, pathWithQuery string) string {
	var sb strings.Builder
	sb.WriteString(strings.ToUpper(method))
	sb.WriteString("\n")
	sb.WriteString(accept)
	sb.WriteString("\n")
	sb.WriteString(contentType)
	sb.WriteString("\n")
	sb.WriteString(headers)
	sb.WriteString("\n")
	sb.WriteString(pathWithQuery)

	stringToSign := sb.String()

	h := hmac.New(sha256.New, []byte(appSecret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// BuildSignedHeaders 格式化签署的 Header 头
func BuildSignedHeaders(signedHeaderMap map[string]string) (string, string) {
	if len(signedHeaderMap) == 0 {
		return "", ""
	}

	keys := make([]string, 0, len(signedHeaderMap))
	for k := range signedHeaderMap {
		keys = append(keys, strings.ToLower(k))
	}
	sort.Strings(keys)

	var headerList []string
	var keyList []string

	for _, k := range keys {
		v := signedHeaderMap[k]
		headerList = append(headerList, fmt.Sprintf("%s:%s", k, v))
		keyList = append(keyList, k)
	}

	return strings.Join(headerList, "\n"), strings.Join(keyList, ",")
}
