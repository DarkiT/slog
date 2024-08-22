// Package dlp provides dlp sdk api implementaion
package dlp

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/darkit/slog/dlp/conf"
	"github.com/darkit/slog/dlp/detector"
	"github.com/darkit/slog/dlp/dlpheader"
	"github.com/darkit/slog/dlp/mask"
)

//go:embed conf/dlp.db
var DefCfg []byte

const (
	Version           = "v1.2.15"
	DefLimitErr       = "<--[DLP] Log Limit Exceeded-->" // 如果超过限制，则追加到日志
	DefMaxLogItem     = 16                               // 日志的最大输入项
	DefResultSize     = 4                                // 数组分配的默认结果大小
	DefLinebackerSize = 1024                             // 默认行块
	DefMaxCallDeep    = 5                                // MaskStruct的最大调用深度
	DefMaxItem        = 1024 * 4                         // MAP API的最大输入项
	DefMaxInput       = 1024 * 1024                      // 1MB，最大输入字符串长度
	DefCutter         = " /\r\n\\[](){}:=\"',"           // 用于在字符串中查找KV对象的默认切割器
)

var (
	DefMaxLogInput    int32 = 1024 // 默认1KB，日志的最大输入长度，在conf中更改
	DefMaxRegexRuleId int32 = 0    // 默认0，没有正则表达式规则将用于日志默认，在conf中更改它
)

// Engine Object implements all DLP API functions
type Engine struct {
	Version     string
	callerID    string
	endPoint    string
	accessKey   string
	secretKey   string
	isLegal     bool // true: auth is ok, false: auth failed
	isClosed    bool // true: Close() has been called
	isForLog    bool // true: NewLogProcessor() has been called, will not do other API
	isConfiged  bool // true: ApplyConfig* API has been called, false: not been called
	confObj     *conf.DlpConf
	detectorMap map[int32]detector.DetectorAPI
	maskerMap   map[string]mask.MaskAPI
}

// NewEngine creates an Engine Object,不要放在循环中调用
// 	Parameters:
// 		callerID: caller ID at the dlp management system.
//
// 	Return:
// 		EngineAPI Object
//
//	Comment:
//

func NewEngine(callerID string) (dlpheader.EngineAPI, error) {
	defer recoveryImplStatic()
	eng := new(Engine)
	eng.Version = Version
	eng.callerID = callerID
	eng.detectorMap = make(map[int32]detector.DetectorAPI)
	eng.maskerMap = make(map[string]mask.MaskAPI)

	return eng, nil
}

// Close release inner object, such as detector and masker
func (I *Engine) Close() {
	defer I.recoveryImpl()
	for k, v := range I.detectorMap {
		if v != nil {
			v.Close()
			I.detectorMap[k] = nil
		}
	}
	for k, v := range I.maskerMap {
		if v != nil {
			I.maskerMap[k] = nil
		}
	}
	I.detectorMap = nil
	I.confObj = nil
	I.isClosed = true
}

// ShowResults print results in console
// 打印识别结果
func (I *Engine) ShowResults(results []*dlpheader.DetectResult) {
	defer I.recoveryImpl()
	fmt.Println()
	fmt.Printf("\tTotal Results: %d\n", len(results))
	for i, item := range results {
		fmt.Printf("Result[%d]: %+v\n", i, *item)
	}
	fmt.Println()
}

// GetVersion return DLP SDK version
func (I *Engine) GetVersion() string {
	defer I.recoveryImpl()
	return Version
}

// NewLogProcessor create a log processer for the package logs
// 调用过之后，eng只能用于log处理，因为规则会做专门的优化，不适合其他API使用
func (I *Engine) NewLogProcessor() dlpheader.Processor {
	defer I.recoveryImpl()

	I.isForLog = true
	_ = I.selectRulesForLog()
	return func(rawLog string, kvs ...interface{}) (string, []interface{}, bool) {
		// do not call log API in this func
		defer I.recoveryImpl()
		// do not call report at here, because this func will call Deidentify()
		// Do not use logs function inside this function
		newLog := rawLog
		logCutted := false
		if int32(len(newLog)) >= DefMaxLogInput {
			// cut for long log
			newLog = newLog[:DefMaxLogInput]
			logCutted = true
		}
		newLog, _, _ = I.deidentifyImpl(newLog)
		if logCutted {
			newLog += DefLimitErr
		}
		// fmt.Printf("LogProcesser rawLog: %s, kvs: %+v\n", rawLog, kvs)
		sz := len(kvs)
		// k1,v1,k2,v2,...
		if sz%2 != 0 {
			sz--
		}
		kvCutted := false
		if sz >= DefMaxLogItem {
			// cut for too many items
			sz = DefMaxLogItem
			kvCutted = true
		}
		retKvs := make([]interface{}, 0, sz)
		if sz > 0 {
			inMap := make(map[string]string)
			for i := 0; i < sz; i += 2 {
				keyStr := I.interfaceToStr(kvs[i])
				valStr := I.interfaceToStr(kvs[i+1])
				inMap[keyStr] = valStr
			}
			outMap, _, _ := I.deidentifyMapImpl(inMap)
			for k, v := range outMap {
				v, _, _ = I.deidentifyImpl(v)
				retKvs = append(retKvs, k, v)
			}
		}
		if kvCutted {
			retKvs = append(retKvs, "<--[DLP Error]-->", DefLimitErr)
		}
		return newLog, retKvs, true
	}
}

// NewEmptyLogProcesser will new a log processer which will do nothing
// 业务禁止使用
func (I *Engine) NewEmptyLogProcessor() dlpheader.Processor {
	return func(rawLog string, kvs ...interface{}) (string, []interface{}, bool) {
		return rawLog, kvs, true
	}
}

// ShowDlpConf print conf on console
func (I *Engine) ShowDlpConf() error {
	// copy obj
	confObj := *I.confObj
	out, err := json.Marshal(confObj)
	if err == nil {
		fmt.Println("====ngdlp conf start====")
		fmt.Println(string(out))
		fmt.Println("====ngdlp conf end====")
		return nil
	} else {
		return err
	}
}

// GetDefaultConf will return default config string
// 返回默认的conf string
func (I *Engine) GetDefaultConf() []byte {
	return DefCfg
}

// ApplyConfigDefault will use embeded local config, only used for DLP team
// 业务禁止使用
func (I *Engine) DisableAllRules() error {
	for i := range I.detectorMap {
		I.detectorMap[i] = nil
	}
	return nil
}

// private func
// interfaceToStr converts interface to string
func (I *Engine) interfaceToStr(in interface{}) string {
	out := ""
	switch in.(type) {
	case []byte:
		out = string(in.([]byte))
	case string:
		out = in.(string)
	default:
		out = fmt.Sprint(in)
	}
	return out
}

// loadDefCfg from the embeded resources
func (I *Engine) loadDefCfg() error {
	if confObj, err := conf.NewDlpConf(DefCfg); err == nil {
		return I.applyConfigImpl(confObj)
	} else {
		return err
	}
}

// formatEndPoint formats endpoint
func (I *Engine) formatEndPoint(endpoint string) string {
	out := endpoint
	if !strings.HasPrefix(endpoint, "http") { // not( http or https)
		out = "http://" + endpoint // defualt use http
		out = strings.TrimSuffix(out, "/")
	}
	return out
}

func B2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func S2B(s string) (b []byte) {
	/* #nosec G103 */
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	/* #nosec G103 */
	sh := *(*reflect.StringHeader)(unsafe.Pointer(&s))
	bh.Data = sh.Data
	bh.Len = sh.Len
	bh.Cap = sh.Len
	return b
}
