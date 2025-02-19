package noodlog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

// logLevel represents the log level flag
var logLevel int = infoLevel

// logWriter is the io.Writer where messages get written
var logWriter io.Writer = os.Stdout

// JSONPrettyPrint represents the pretty printing flag
var JSONPrettyPrint bool = false

// obscureSensitiveDataEnabled represents the sensitive data obscuration flag
var obscureSensitiveDataEnabled bool = false

var logLevels = map[string]int{
	traceLabel: traceLevel,
	debugLabel: debugLevel,
	infoLabel:  infoLevel,
	warnLabel:  warnLevel,
	errorLabel: errorLevel,
	panicLabel: panicLevel,
	fatalLabel: fatalLevel,
}

var sensitiveParams = []string{}

// SetConfigs function allows you to add all the configs at once
func SetConfigs(configs Configs) {
	if configs.LogLevel != nil {
		LogLevel(*configs.LogLevel)
	}
	if configs.JSONPrettyPrint != nil {
		JSONPrettyPrint = *configs.JSONPrettyPrint
	}
	if configs.TraceCaller != nil {
		traceCallerEnabled = *configs.TraceCaller
	}
	if configs.SinglePointTracing != nil {
		if *(configs.SinglePointTracing) {
			EnableSinglePointTracing()
		} else {
			DisableSinglePointTracing()
		}
	}
	if configs.Colors != nil {
		colorEnabled = *configs.Colors
	}
	if configs.CustomColors != nil {
		setCustomColors(*configs.CustomColors)
	}
	if configs.ObscureSensitiveData != nil {
		obscureSensitiveDataEnabled = *configs.ObscureSensitiveData
	}
	if configs.SensitiveParams != nil {
		SetSensitiveParams(configs.SensitiveParams)
	}

}

// LogLevel function sets the log level
func LogLevel(level string) {
	logLevel = logLevels[level]
	if logLevel == 0 {
		logLevel = infoLevel
	}
}

// LogWriter function sets the new writer
// TODO: if handle is a file, disable color and indentation ?
func LogWriter(w io.Writer) {
	logWriter = w
}

// EnableJSONPrettyPrint enables JSON pretty printing
func EnableJSONPrettyPrint() {
	JSONPrettyPrint = true
}

// DisableJSONPrettyPrint diables JSON pretty printing
func DisableJSONPrettyPrint() {
	JSONPrettyPrint = false
}

// EnableObscureSensitiveData enables sensitive data obscuration from json logs
func EnableObscureSensitiveData(params []string) {
	obscureSensitiveDataEnabled = true
	SetSensitiveParams(params)
}

// DisableObscureSensitiveData disables sensitive data obscuration from json logs
func DisableObscureSensitiveData() {
	obscureSensitiveDataEnabled = false
}

// SetSensitiveParams sets sensitive data obscuration from json logs
func SetSensitiveParams(params []string) {
	sensitiveParams = params
}

// Trace function prints a log with trace log level
func Trace(message ...interface{}) {
	printLog(traceLabel, message)
}

// Debug function prints a log with debug log level
func Debug(message ...interface{}) {
	printLog(debugLabel, message)
}

// Info function prints a log with info log level
func Info(message ...interface{}) {
	printLog(infoLabel, message)
}

// Warn function prints a log with warn log level
func Warn(message ...interface{}) {
	printLog(warnLabel, message)
}

// Error function prints a log with error log level
func Error(message ...interface{}) {
	printLog(errorLabel, message)
}

// Panic function prints a log with panic log level
func Panic(message ...interface{}) {
	panic(composeLog(panicLabel, message))
}

// Fatal function prints a log with fatal log level
func Fatal(message ...interface{}) {
	printLog(fatalLabel, message)
	os.Exit(1)
}

func printLog(label string, message []interface{}) {
	if logLevels[label] >= logLevel {
		fmt.Fprintf(logWriter, composeLog(label, message))
	}
}

func composeLog(level string, message []interface{}) string {

	logMsg := record{
		Level:   level,
		Message: composeMessage(message),
		Time:    strings.Split(time.Now().String(), "m")[0],
	}

	if traceCallerEnabled {
		f, fx := traceCaller()
		logMsg.File = &f
		logMsg.Function = &fx
	}

	var jsn []byte
	if JSONPrettyPrint {
		jsn, _ = json.MarshalIndent(logMsg, "", "   ")
	} else {
		jsn, _ = json.Marshal(logMsg)
	}

	logRecord := string(jsn)
	if colorEnabled {
		logRecord = fmt.Sprintf("%s%s%s", colorMap[level], logRecord, colorReset)
	}

	return logRecord
}

func composeMessage(message []interface{}) interface{} {
	switch len(message) {
	case 0:
		return ""
	case 1:
		return adaptMessage(message[0])
	default:
		switch message[0].(type) {
		case string:
			msg0 := message[0].(string)
			if strings.Contains(msg0, "%") {
				return fmt.Sprintf(msg0, message[1:]...)
			}
		}
		return stringify(message)
	}
}

func stringify(message []interface{}) string {
	var b strings.Builder
	for _, m := range message {
		if m != nil {
			fmt.Fprintf(&b, "%v ", m)
		}
	}
	msg := b.String()
	return msg[:len(msg)-1]
}

func adaptMessage(message interface{}) interface{} {
	switch message.(type) {
	case string:
		strMsg := message.(string)
		if obscureSensitiveDataEnabled && len(sensitiveParams) != 0 {
			return strToObj(obscureSensitiveData(strMsg))
		}
		return strToObj(strMsg)
	default:
		if obscureSensitiveDataEnabled && len(sensitiveParams) != 0 {
			jsn, _ := json.Marshal(message)
			strMsg := obscureSensitiveData(string(jsn))
			return strToObj(strMsg)
		}
	}
	return message
}

func strToObj(strMsg string) interface{} {
	if byteMsg := []byte(strMsg); json.Valid(byteMsg) {
		var obj interface{}
		_ = json.Unmarshal(byteMsg, &obj)
		return obj
	}
	return strMsg
}

func obscureSensitiveData(jsn string) string {
	for _, param := range sensitiveParams {
		jsn = obscureParam(jsn, param)
	}
	return jsn
}

func obscureParam(jsn string, param string) string {
	rWithSlash := *regexp.MustCompile(`\\"` + param + `\\":.*?"(.*?)\\"`)
	jsn = rWithSlash.ReplaceAllString(jsn, `\"`+param+`\": \"**********\"`)

	rWithoutSlash := *regexp.MustCompile(`"` + param + `":.*?"(.*?)"`)
	return rWithoutSlash.ReplaceAllString(jsn, `"`+param+`": "**********"`)
}
