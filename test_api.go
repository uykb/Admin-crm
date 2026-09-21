package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"apeadmin-gin/internal/model"
	"apeadmin-gin/internal/plugin/builtin/hikiot/client"
)

func main() {
	db, err := gorm.Open(sqlite.Open("apeadmin_gin"), &gorm.Config{})
	if err != nil {
		fmt.Println("DB error:", err)
		return
	}
	var k, s model.SysSetting
	db.Where("key = ?", "hikiot_app_key").First(&k)
	db.Where("key = ?", "hikiot_app_secret").First(&s)
	
	fmt.Println("AppKey:", k.Value)
	if k.Value == "" {
		fmt.Println("AppKey not found in local db. Cannot test.")
		return
	}
	
	cli := client.NewClient("https://open-api.hikiot.com", k.Value, s.Value)
	tok, err := cli.ExchangeAppToken()
	if err != nil {
		fmt.Println("Token error:", err)
		return
	}
	fmt.Println("Token acquired:", tok.AppAccessToken[:10], "...")
	
	endpoints := []string{
		"/device/v1/device",
		"/device/v1/device/list",
		"/device/v1/device/page",
		"/device/v2/device/page",
		"/resource/v1/door/list",
		"/acs/v1/door/list",
		"/acs/v1/door/page",
		"/door/v1/door/list",
		"/door/v1/device/list",
		"/attendance/v1/door/list",
		"/team/v1/door/list",
		"/team/v1/device/list",
		"/base/v1/device/list",
	}
	
	for _, ep := range endpoints {
		fmt.Printf("\nTesting GET %s\n", ep)
		req, _ := http.NewRequest("GET", "https://open-api.hikiot.com"+ep, nil)
		req.Header.Set("access_token", tok.AppAccessToken)
		req.Header.Set("Authorization", "Bearer "+tok.AppAccessToken)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			b, _ := io.ReadAll(resp.Body)
			fmt.Println("Response:", string(b))
		}
		
		fmt.Printf("Testing POST %s\n", ep)
		b, _ := json.Marshal(map[string]int{"pageNo": 1, "pageSize": 100})
		req2, _ := http.NewRequest("POST", "https://open-api.hikiot.com"+ep, bytes.NewBuffer(b))
		req2.Header.Set("access_token", tok.AppAccessToken)
		req2.Header.Set("Authorization", "Bearer "+tok.AppAccessToken)
		req2.Header.Set("Content-Type", "application/json")
		resp2, err := http.DefaultClient.Do(req2)
		if err == nil {
			b2, _ := io.ReadAll(resp2.Body)
			fmt.Println("Response:", string(b2))
		}
	}
}
