// 根据官方的示例文件做了一些微小的调整。

/*
Copyright (year) Beijing Volcano Engine Technology Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package volcengine

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

const (
	// 请求凭证，从访问控制申请
	AccessKeyID     = ""
	SecretAccessKey = ""

	// 请求地址
	Addr = "https://open.volcengineapi.com"
	Path = "/" // 路径，不包含 Query

	// 请求接口信息
	Service = "speech_saas_prod"
	Region  = "cn-north-1"
	Action  = "UsageMonitoring"
	Version = "2021-08-30"
)

func SpeechSvcBillQuery(ctx context.Context) error {
	method := http.MethodGet

	// 从配置文件获取 AK/SK
	AccessKeyID := g.Cfg().MustGet(ctx, "volc.ak").String()
	secretKeyFromConfig := g.Cfg().MustGet(ctx, "volc.sk").String()

	// SK 可能是 Base64 编码的，尝试解码
	var SecretAccessKey string
	if skBytes, err := base64.StdEncoding.DecodeString(secretKeyFromConfig); err == nil {
		SecretAccessKey = string(skBytes)
		g.Log().Infof(ctx, "SK Base64 解码成功")
	} else {
		SecretAccessKey = secretKeyFromConfig
		g.Log().Infof(ctx, "SK 直接使用原始值")
	}

	// 添加调试日志
	g.Log().Infof(ctx, "AccessKeyID: %s", AccessKeyID)
	g.Log().Infof(ctx, "SecretAccessKey length: %d", len(SecretAccessKey))

	Addr := g.Cfg().MustGet(ctx, "volc.lark.billQueryAddr").String()
	Path := "/"
	Service := g.Cfg().MustGet(ctx, "volc.lark.service").String()
	Region := g.Cfg().MustGet(ctx, "volc.region").String()
	Action := "UsageMonitoring"
	Version := "2021-08-30"

	// 添加更多调试日志
	g.Log().Infof(ctx, "Service: %s, Region: %s", Service, Region)

	queries := make(url.Values)
	queries.Set("Action", Action)
	queries.Set("Version", Version)
	queries.Set("AppID", g.Cfg().MustGet(ctx, "volc.lark.appid").String())
	queries.Set("ResourceID", g.Cfg().MustGet(ctx, "volc.lark.service").String())
	queries.Set("Start", "2025-02-23")
	queries.Set("End", "2025-10-25")
	queries.Set("Mode", "daily")

	// 添加调试日志，显示完整的查询参数
	g.Log().Infof(ctx, "Query parameters: %s", queries.Encode())

	body := []byte{}

	// 1. 构建请求
	requestAddr := fmt.Sprintf("%s%s?%s", Addr, Path, queries.Encode())
	g.Log().Infof(ctx, "request addr: %s\n", requestAddr)

	request, err := http.NewRequest(method, requestAddr, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("bad request: %w", err)
	}

	// 2. 构建签名材料
	now := time.Now()
	date := now.UTC().Format("20060102T150405Z")
	authDate := date[:8]

	// 添加时间调试信息
	g.Log().Infof(ctx, "Current UTC time: %s", now.UTC().String())
	g.Log().Infof(ctx, "Formatted date: %s", date)
	g.Log().Infof(ctx, "Auth date: %s", authDate)

	request.Header.Set("X-Date", date)

	payload := hex.EncodeToString(hashSHA256(ctx, body))
	request.Header.Set("X-Content-Sha256", payload)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	queryString := strings.Replace(queries.Encode(), "+", "%20", -1)
	signedHeaders := []string{"host", "x-date", "x-content-sha256", "content-type"}
	var headerList []string
	for _, header := range signedHeaders {
		if header == "host" {
			headerList = append(headerList, header+":"+request.Host)
		} else {
			// 根据官方示例，需要正确处理头部字段的获取
			var v string
			switch header {
			case "x-date":
				v = request.Header.Get("X-Date")
			case "x-content-sha256":
				v = request.Header.Get("X-Content-Sha256")
			case "content-type":
				v = request.Header.Get("Content-Type")
			default:
				v = request.Header.Get(header)
			}
			headerList = append(headerList, header+":"+strings.TrimSpace(v))
		}
	}
	headerString := strings.Join(headerList, "\n")

	canonicalString := strings.Join([]string{
		method,
		Path,
		queryString,
		headerString + "\n",
		strings.Join(signedHeaders, ";"),
		payload,
	}, "\n")
	g.Log().Infof(ctx, "canonical string:\n%s\n", canonicalString)

	hashedCanonicalString := hex.EncodeToString(hashSHA256(ctx, []byte(canonicalString)))
	g.Log().Infof(ctx, "hashed canonical string: %s\n", hashedCanonicalString)

	credentialScope := authDate + "/" + Region + "/" + Service + "/request"
	signString := strings.Join([]string{
		"HMAC-SHA256",
		date,
		credentialScope,
		hashedCanonicalString,
	}, "\n")
	g.Log().Infof(ctx, "sign string:\n%s\n", signString)

	// 3. 构建认证请求头
	signedKey := getSignedKey(SecretAccessKey, authDate, Region, Service)
	signature := hex.EncodeToString(hmacSHA256(signedKey, signString))
	g.Log().Infof(ctx, "signature: %s\n", signature)

	authorization := "HMAC-SHA256" +
		" Credential=" + AccessKeyID + "/" + credentialScope +
		", SignedHeaders=" + strings.Join(signedHeaders, ";") +
		", Signature=" + signature
	request.Header.Set("Authorization", authorization)
	g.Log().Infof(ctx, "authorization: %s\n", authorization)

	// 4. 打印请求，发起请求
	requestRaw, err := httputil.DumpRequest(request, true)
	if err != nil {
		return fmt.Errorf("dump request err: %w", err)
	}

	g.Log().Infof(ctx, "request:\n%s\n", string(requestRaw))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("do request err: %w", err)
	}

	// 5. 打印响应
	responseRaw, err := httputil.DumpResponse(response, true)
	if err != nil {
		return fmt.Errorf("dump response err: %w", err)
	}

	g.Log().Infof(ctx, "response:\n%s\n", string(responseRaw))

	if response.StatusCode == 200 {
		g.Log().Infof(ctx, "请求成功")
	} else {
		g.Log().Infof(ctx, "请求失败")
	}

	return nil
}

// TestSpeechSvcBillQuery 用于测试账单查询功能
func TestSpeechSvcBillQuery(ctx context.Context) {
	g.Log().Info(ctx, "开始测试语音服务账单查询...")

	err := SpeechSvcBillQuery(ctx)
	if err != nil {
		g.Log().Errorf(ctx, "账单查询失败: %v", err)
	} else {
		g.Log().Info(ctx, "账单查询测试完成")
	}
}

func hmacSHA256(key []byte, content string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(content))
	return mac.Sum(nil)
}

func getSignedKey(secretKey, date, region, service string) []byte {
	kDate := hmacSHA256([]byte(secretKey), date)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	kSigning := hmacSHA256(kService, "request")

	return kSigning
}

func hashSHA256(ctx context.Context, data []byte) []byte {
	hash := sha256.New()
	if _, err := hash.Write(data); err != nil {
		g.Log().Infof(ctx, "input hash err:%s", err.Error())
	}

	return hash.Sum(nil)
}
