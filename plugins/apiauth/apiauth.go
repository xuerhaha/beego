// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package apiauth provides handlers to enable apiauth support.
// Simple Usage:
//	import(
//		"github.com/astaxie/beego"
//		"github.com/astaxie/beego/plugins/apiauth"
//	)
//
//	func main(){
//		// apiauth every request
//		beego.InsertFilter("*", beego.BeforeRouter,auth.APIAuth("appid","appkey"))
//		beego.Run()
//	}
//
package apiauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
)

type AppIdToAppSecret func(string) string

func APIBaiscAuth(appid, appkey string) beego.FilterFunc {
	ft := func(aid string) string {
		if aid == appid {
			return appkey
		}
		return ""
	}
	return APIAuthWithFunc(ft, 300)
}

func APIAuthWithFunc(f AppIdToAppSecret, timeout int) beego.FilterFunc {
	return func(ctx *context.Context) {
		if ctx.Input.Query("appid") == "" {
			ctx.Output.SetStatus(403)
			ctx.WriteString("miss query param: appid")
			return
		}
		appsecret := f(ctx.Input.Query("appid"))
		if appsecret == "" {
			ctx.Output.SetStatus(403)
			ctx.WriteString("not exist this appid")
			return
		}
		if ctx.Input.Query("signature") == "" {
			ctx.Output.SetStatus(403)
			ctx.WriteString("miss query param: signature")
			return
		}
		if ctx.Input.Query("timestamp") == "" {
			ctx.Output.SetStatus(403)
			ctx.WriteString("miss query param: timestamp")
			return
		}
		u, err := time.Parse("2006-01-02 15:04:05", ctx.Input.Query("timestamp"))
		if err != nil {
			ctx.Output.SetStatus(403)
			ctx.WriteString("timestamp format is error, should 2006-01-02 15:04:05")
			return
		}
		t := time.Now()
		if (t.Second() - u.Second()) > timeout {
			ctx.Output.SetStatus(403)
			ctx.WriteString("timeout! the request time is long ago, please try again")
			return
		}
		if ctx.Input.Query("signature") !=
			Signature(appsecret, ctx.Input.Method(), ctx.Request.Form, ctx.Input.Uri()) {
			ctx.Output.SetStatus(403)
			ctx.WriteString("auth failed")
		}
	}
}

func Signature(appsecret, method string, params url.Values, RequestURI string) (result string) {
	var query string
	pa := make(map[string]string)
	for k, v := range params {
		pa[k] = v[0]
	}
	vs := mapSorter(pa)
	vs.Sort()
	for i := 0; i < vs.Len(); i++ {
		if vs.Keys[i] == "signature" {
			continue
		}
		if vs.Keys[i] != "" && vs.Vals[i] != "" {
			query = fmt.Sprintf("%v%v%v", query, vs.Keys[i], vs.Vals[i])
		}
	}
	string_to_sign := fmt.Sprintf("%v\n%v\n%v\n", method, query, RequestURI)

	sha256 := sha256.New
	hash := hmac.New(sha256, []byte(appsecret))
	hash.Write([]byte(string_to_sign))
	sha := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	sha = url.QueryEscape(sha)
	sha = strings.Replace(sha, "+", "%20", -1)
	sha = strings.Replace(sha, "*", "%2A", -1)
	sha = strings.Replace(sha, "%7E", "~", -1)
	return sha
}

type valSorter struct {
	Keys []string
	Vals []string
}

func mapSorter(m map[string]string) *valSorter {
	vs := &valSorter{
		Keys: make([]string, 0, len(m)),
		Vals: make([]string, 0, len(m)),
	}
	for k, v := range m {
		vs.Keys = append(vs.Keys, k)
		vs.Vals = append(vs.Vals, v)
	}
	return vs
}

func (vs *valSorter) Sort() {
	sort.Sort(vs)
}

func (vs *valSorter) Len() int           { return len(vs.Keys) }
func (vs *valSorter) Less(i, j int) bool { return vs.Keys[i] < vs.Keys[j] }
func (vs *valSorter) Swap(i, j int) {
	vs.Vals[i], vs.Vals[j] = vs.Vals[j], vs.Vals[i]
	vs.Keys[i], vs.Keys[j] = vs.Keys[j], vs.Keys[i]
}