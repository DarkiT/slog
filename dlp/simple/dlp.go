package main

import (
	"fmt"
	"os"
	"strings"

	Init "github.com/darkit/slog/dlp"
	"github.com/darkit/slog/dlp/dlpheader"
)

var inStr = `
我的邮件是：abcd@abcd.com,
13800138000 是我的电话
你在哪里啊? 广东省广州市市南大道金沙路17号楼222-3室,
mac地址: 06-06-06-aa-bb-cc
mac地址: 06:06:06:aa:bb:cc
银行卡: 9988002866797031
车牌: 川A123AG 电话: 010-86551122
账号: admin 密码: 123456
收件人: 张真人 手机号码：18612341234
网址: https://admin:123456@www.zishuo.net/live/demo?token=123456&user=zishuo
网址: rtsp://admin:123456@192.168.1.188/live/demo?token=123456&user=zishuo
网址: rtmp://admin:123456@www.zishuo.net/live/demo?token=123456&user=zishuo
ipv4地址: 192.168.1.188 ipv6地址: 2001:0db8:85a3:0000:0000:8a2e:0370:7334
`

func dlpDemo() {
	caller := "replace.your.caller"
	// 使用时请将NewEngine()放到循环外，每个线程独立一个Engine Object
	// remove NewEngein() outside for loop, and one Engine Object one thread/goroutin
	if eng, err := Init.DlpInit(caller); err == nil {
		_ = eng.ApplyConfigDefault()
		fmt.Printf("DLP %s Demo:\n\n", eng.GetVersion())
		urlStr := "https://admin:123456@192.168.1.188/live/demo?token=123456&user=zishuo"
		if outStr, err := eng.Mask(urlStr, dlpheader.URL); err == nil {
			fmt.Printf("\toutStr: %s\n", outStr)
			// eng.ShowResults(results)
			fmt.Println()
		}
		if results, err := eng.Detect(inStr); err == nil {
			fmt.Printf("\t1. Detect( inStr: %s )\n", inStr)
			eng.ShowResults(results)
		}
		if outStr, _, err := eng.Deidentify(inStr); err == nil {
			fmt.Printf("\t2. Deidentify( inStr: %s )\n", inStr)
			fmt.Printf("\toutStr: %s\n", outStr)
			// eng.ShowResults(results)
			fmt.Println()
		}
		os.Exit(0)
		inStr = `18612341234`
		if outStr, err := eng.Mask(inStr, dlpheader.CHINAPHONE); err == nil {
			fmt.Printf("\t3. Mask( inStr: %s )\n", inStr)
			fmt.Printf("\toutStr: %s\n", outStr)
			fmt.Println()
		}

		inMap := map[string]string{"nothing": "nothing", "uid": "10086", "k1": "my phone is 18612341234 and 18612341234"} // extract KV?

		if results, err := eng.DetectMap(inMap); err == nil {
			fmt.Printf("\t4. DetectMap( inMap: %+v )\n", inMap)
			eng.ShowResults(results)
		}

		fmt.Printf("\t5. DeidentifyMap( inMap: %+v )\n", inMap)
		if outMap, results, err := eng.DeidentifyMap(inMap); err == nil {
			fmt.Printf("\toutMap: %+v\n", outMap)
			eng.ShowResults(results)
			fmt.Println()
		}

		inJSON := `{"objList":[{"uid":"10086"},{"uid":"[\"aaaa\",\"bbbb\"]"}]}`

		if results, err := eng.DetectJSON(inJSON); err == nil {
			fmt.Printf("\t6. DetectJSON( inJSON: %s )\n", inJSON)
			eng.ShowResults(results)
		}

		if outJSON, results, err := eng.DeidentifyJSON(inJSON); err == nil {
			fmt.Printf("\t7. DeidentifyJSON( inJSON: %s )\n", inJSON)
			fmt.Printf("\toutJSON: %s\n", outJSON)
			eng.ShowResults(results)
			fmt.Println()
		}
		inStr = "abcd@abcd.com"
		maskRule := "EmailMaskRule01"
		if outStr, err := eng.Mask(inStr, maskRule); err == nil {
			fmt.Printf("\t8. Mask( inStr: %s , %s)\n", inStr, maskRule)
			fmt.Printf("\toutStr: %s\n", outStr)
			fmt.Println()
		}
		// 自定义脱敏，邮箱用户名保留首尾各一个字符，保留所有域名
		_ = eng.RegisterMasker("EmailMaskRule02", func(in string) (string, error) {
			list := strings.Split(in, "@")
			if len(list) >= 2 {
				prefix := list[0]
				domain := list[1]
				if len(prefix) > 2 {
					prefix = "*" + prefix[1:len(prefix)-1] + "*"
				} else if len(prefix) > 0 {
					prefix = "*" + prefix[1:]
				} else {
					return in, fmt.Errorf("%s is not Email", in)
				}
				ret := prefix + "@" + domain
				return ret, nil
			} else {
				return in, fmt.Errorf("%s is not Email", in)
			}
		})
		inStr = "abcd@abcd.com"
		maskRule = "EmailMaskRule02"
		if outStr, err := eng.Mask(inStr, maskRule); err == nil {
			fmt.Printf("\t9. Mask( inStr: %s , %s)\n", inStr, maskRule)
			fmt.Printf("\toutStr: %s\n", outStr)
			fmt.Println()
		}

		inStr = "loginfo:[ uid:10086, phone:18612341234]"
		if outStr, results, err := eng.Deidentify(inStr); err == nil {
			fmt.Printf("\t10. Detect( inStr: %s )\n", inStr)
			eng.ShowResults(results)
			fmt.Printf("\toutStr: %s\n", outStr)
			fmt.Println()
		}
		type EmailType string
		// 需要递归的结构体，需要填 `mask:"DEEP"` 才会递归脱敏
		type Foo struct {
			Email         EmailType `mask:"EMAIL"`
			PhoneNumber   string    `mask:"CHINAPHONE"`
			Idcard        string    `mask:"CHINAID"`
			Buffer        string    `mask:"DEIDENTIFY"`
			EmailPtrSlice []*struct {
				Val string `mask:"EMAIL"`
			} `mask:"DEEP"`
			PhoneSlice []string `mask:"CHINAPHONE"`
			Extinfo    *struct {
				Addr string `mask:"ADDRESS"`
			} `mask:"DEEP"`
			EmailArray [2]string   `mask:"EMAIL"`
			NULLPtr    *Foo        `mask:"DEEP"`
			IFace      interface{} `mask:"ExampleTAG"`
		}
		inObj := Foo{
			"abcd@abcd.com",
			"18612341234",
			"110225196403026127",
			"我的邮件是abcd@abcd.com",
			[]*struct {
				Val string `mask:"EMAIL"`
			}{{"3333@4444.com"}, {"5555@6666.com"}},
			[]string{"18612341234", ""},
			&struct {
				Addr string "mask:\"ADDRESS\""
			}{"北京市海淀区北三环西路43号"},
			[2]string{"abcd@abcd.com", "3333@4444.com"},
			nil,
			"abcd@abcd.com",
		}
		inPtr := &inObj
		inObj.NULLPtr = inPtr
		fmt.Printf("\t11. MaskStruct( inPtr: %+v, Extinfo: %+v)\n", inPtr, *(inPtr.Extinfo))
		// MaskStruct 参数必须是pointer, 才能修改struct 内部元素
		if outPtr, err := eng.MaskStruct(inPtr); err == nil {
			fmt.Printf("\toutObj: %+v, Extinfo:%+v\n", outPtr, inObj.Extinfo)
			fmt.Printf("\t\t EmailPtrSlice:\n")
			for i, ePtr := range inObj.EmailPtrSlice {
				fmt.Printf("\t\t\t[%d] = %s\n", i, ePtr.Val)
			}
			fmt.Println()
		} else {
			fmt.Println(err.Error())
		}
		// fmt.Println(eng.GetDefaultConf())
		eng.Close()
	} else {
		fmt.Println("[dlp] NewEngine error: ", err.Error())
	}
}

func main() {
	sensitive := &Sensitive{
		Name:         "刘子豪",
		IdCard:       "530321199204074611",
		FixedPhone:   "01086551122",
		MobilePhone:  "13248765917",
		Address:      "广州市天河区幸福小区102号",
		Email:        "example@gmail.com",
		Password:     "123456",
		CarLicense:   "粤A66666",
		BankCard:     "9988002866797031",
		Ipv4:         "192.0.2.1",
		Ipv6:         "2001:0db8:86a3:08d3:1319:8a2e:0370:7344",
		Base64:       "你好世界!",
		FirstMask:    "123456789",
		ClearToNull:  "Some Data",
		ClearToEmpty: "Some Data",
		URL:          "https://user:123456@www.zishuo.net/live/demo?token=123456&user=zishuo",
	}

	ProcessSensitiveData(sensitive)
	fmt.Printf("%v\n", sensitive)

	data := UrlDesensitize(sensitive.URL, "token", "user")

	fmt.Printf("%v\n", data)

	CustomFilterWords := "卧日，TMD，我去你大爷的，CNM"
	data = StringDesensitize(CustomFilterWords, "TMD", "CNM")

	fmt.Printf("%v\n", data)

	data = CustomizeKeepLengthDesensitize(CustomFilterWords, 3, 4)
	fmt.Printf("%v\n", data)

	dlpDemo()
}
