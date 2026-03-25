package logger_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/logger"
)

const debugF = "Debug"
const debuglnF = "Debugln"
const debugfF = "Debugf"
const warningF = "Warning"
const warninglnF = "Warningln"
const warningfF = "Warningf"
const infoF = "Info"
const infolnF = "Infoln"
const infofF = "Infof"

func TestLogger(t *testing.T) {
	testCases := []struct {
		f        string
		expected string
		input    []any
		level    logger.Level
	}{
		{input: []any{"test"}, level: logger.NoneLevel, f: debugF, expected: ""},
		{input: []any{"test"}, level: logger.NoneLevel, f: debuglnF, expected: ""},
		{input: []any{"test%v", 1}, level: logger.NoneLevel, f: debugfF, expected: ""},

		{input: []any{"test"}, level: logger.DebugLevel, f: debugF, expected: "test"},
		{input: []any{"test"}, level: logger.DebugLevel, f: debuglnF, expected: "test\n"},
		{input: []any{"test%v", 1}, level: logger.DebugLevel, f: debugfF, expected: "test1"},

		{input: []any{"test"}, level: logger.WarningLevel, f: debugF, expected: ""},
		{input: []any{"test"}, level: logger.WarningLevel, f: debuglnF, expected: ""},
		{input: []any{"test%v", 1}, level: logger.WarningLevel, f: debugfF, expected: ""},

		{input: []any{"test"}, level: logger.NoneLevel, f: warningF, expected: ""},
		{input: []any{"test"}, level: logger.NoneLevel, f: warninglnF, expected: ""},
		{input: []any{"test%v", 1}, level: logger.NoneLevel, f: warningfF, expected: ""},

		{input: []any{"test"}, level: logger.DebugLevel, f: warningF, expected: "test"},
		{input: []any{"test"}, level: logger.DebugLevel, f: warninglnF, expected: "test\n"},
		{input: []any{"test%v", 1}, level: logger.DebugLevel, f: warningfF, expected: "test1"},

		{input: []any{"test"}, level: logger.DebugLevel, f: warningF, expected: "test"},
		{input: []any{"test"}, level: logger.DebugLevel, f: warninglnF, expected: "test\n"},
		{input: []any{"test%v", 1}, level: logger.DebugLevel, f: warningfF, expected: "test1"},

		{input: []any{"test"}, level: logger.NoneLevel, f: infoF, expected: ""},
		{input: []any{"test"}, level: logger.NoneLevel, f: infolnF, expected: ""},
		{input: []any{"test%v", 1}, level: logger.NoneLevel, f: infofF, expected: ""},

		{input: []any{"test"}, level: logger.WarningLevel, f: infoF, expected: ""},
		{input: []any{"test"}, level: logger.WarningLevel, f: infolnF, expected: ""},
		{input: []any{"test%v", 1}, level: logger.WarningLevel, f: infofF, expected: ""},

		{input: []any{"test"}, level: logger.InfoLevel, f: infoF, expected: "test"},
		{input: []any{"test"}, level: logger.InfoLevel, f: infolnF, expected: "test\n"},
		{input: []any{"test%v", 1}, level: logger.InfoLevel, f: infofF, expected: "test1"},

		{input: []any{"test"}, level: logger.DebugLevel, f: infoF, expected: "test"},
		{input: []any{"test"}, level: logger.DebugLevel, f: infolnF, expected: "test\n"},
		{input: []any{"test%v", 1}, level: logger.DebugLevel, f: infofF, expected: "test1"},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%v %v", i, testCase.f), func(t *testing.T) {
			buf := new(bytes.Buffer)
			log := logger.New(buf, testCase.level)
			var err error
			switch testCase.f {
			case debugF:
				_, err = log.Debug(testCase.input...)
			case debuglnF:
				_, err = log.Debugln(testCase.input...)
			case debugfF:
				_, err = log.Debugf(testCase.input[0].(string), testCase.input[1:]...)
			case warningF:
				_, err = log.Warning(testCase.input...)
			case warninglnF:
				_, err = log.Warningln(testCase.input...)
			case warningfF:
				_, err = log.Warningf(testCase.input[0].(string), testCase.input[1:]...)
			case infoF:
				_, err = log.Info(testCase.input...)
			case infolnF:
				_, err = log.Infoln(testCase.input...)
			case infofF:
				_, err = log.Infof(testCase.input[0].(string), testCase.input[1:]...)
			}
			if err != nil {
				t.Fatal(err)
			}
			got := buf.String()
			if got != testCase.expected {
				t.Fatalf("expected %v got %v", testCase.expected, got)
			}
		})
	}
}

func TestPackage(t *testing.T) {
	testCases := []struct {
		f        string
		expected string
		input    []any
		level    logger.Level
	}{
		{input: []any{"test"}, level: logger.NoneLevel, f: debugF, expected: ""},
		{input: []any{"test"}, level: logger.NoneLevel, f: debuglnF, expected: ""},
		{input: []any{"test%v", 1}, level: logger.NoneLevel, f: debugfF, expected: ""},

		{input: []any{"test"}, level: logger.DebugLevel, f: debugF, expected: "test"},
		{input: []any{"test"}, level: logger.DebugLevel, f: debuglnF, expected: "test\n"},
		{input: []any{"test%v", 1}, level: logger.DebugLevel, f: debugfF, expected: "test1"},

		{input: []any{"test"}, level: logger.WarningLevel, f: debugF, expected: ""},
		{input: []any{"test"}, level: logger.WarningLevel, f: debuglnF, expected: ""},
		{input: []any{"test%v", 1}, level: logger.WarningLevel, f: debugfF, expected: ""},

		{input: []any{"test"}, level: logger.NoneLevel, f: warningF, expected: ""},
		{input: []any{"test"}, level: logger.NoneLevel, f: warninglnF, expected: ""},
		{input: []any{"test%v", 1}, level: logger.NoneLevel, f: warningfF, expected: ""},

		{input: []any{"test"}, level: logger.DebugLevel, f: warningF, expected: "test"},
		{input: []any{"test"}, level: logger.DebugLevel, f: warninglnF, expected: "test\n"},
		{input: []any{"test%v", 1}, level: logger.DebugLevel, f: warningfF, expected: "test1"},

		{input: []any{"test"}, level: logger.DebugLevel, f: warningF, expected: "test"},
		{input: []any{"test"}, level: logger.DebugLevel, f: warninglnF, expected: "test\n"},
		{input: []any{"test%v", 1}, level: logger.DebugLevel, f: warningfF, expected: "test1"},

		{input: []any{"test"}, level: logger.NoneLevel, f: infoF, expected: ""},
		{input: []any{"test"}, level: logger.NoneLevel, f: infolnF, expected: ""},
		{input: []any{"test%v", 1}, level: logger.NoneLevel, f: infofF, expected: ""},

		{input: []any{"test"}, level: logger.WarningLevel, f: infoF, expected: ""},
		{input: []any{"test"}, level: logger.WarningLevel, f: infolnF, expected: ""},
		{input: []any{"test%v", 1}, level: logger.WarningLevel, f: infofF, expected: ""},

		{input: []any{"test"}, level: logger.InfoLevel, f: infoF, expected: "test"},
		{input: []any{"test"}, level: logger.InfoLevel, f: infolnF, expected: "test\n"},
		{input: []any{"test%v", 1}, level: logger.InfoLevel, f: infofF, expected: "test1"},

		{input: []any{"test"}, level: logger.DebugLevel, f: infoF, expected: "test"},
		{input: []any{"test"}, level: logger.DebugLevel, f: infolnF, expected: "test\n"},
		{input: []any{"test%v", 1}, level: logger.DebugLevel, f: infofF, expected: "test1"},
	}

	oldWriter := logger.Writer()
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%v %v", i, testCase.f), func(t *testing.T) {
			buf := new(bytes.Buffer)
			logger.SetWriter(buf)
			logger.SetLevel(testCase.level)
			var err error
			switch testCase.f {
			case debugF:
				_, err = logger.Debug(testCase.input...)
			case debuglnF:
				_, err = logger.Debugln(testCase.input...)
			case debugfF:
				_, err = logger.Debugf(testCase.input[0].(string), testCase.input[1:]...)
			case warningF:
				_, err = logger.Warning(testCase.input...)
			case warninglnF:
				_, err = logger.Warningln(testCase.input...)
			case warningfF:
				_, err = logger.Warningf(testCase.input[0].(string), testCase.input[1:]...)
			case infoF:
				_, err = logger.Info(testCase.input...)
			case infolnF:
				_, err = logger.Infoln(testCase.input...)
			case infofF:
				_, err = logger.Infof(testCase.input[0].(string), testCase.input[1:]...)
			}
			if err != nil {
				t.Fatal(err)
			}
			got := buf.String()
			if got != testCase.expected {
				t.Fatalf("expected %v got %v", testCase.expected, got)
			}
		})
	}
	logger.SetLevel(logger.InfoLevel)
	logger.SetWriter(oldWriter)
}
