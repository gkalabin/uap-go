package fuzz

import "github.com/gkalabin/uap-go/uaparser"

const unknownFamily = "Other"

var parser uaparser.Parser

func init() {
	loaded, err := uaparser.New("regexes.yaml")
	if err != nil {
		panic(err)
	}
	parser = loaded
}

// Fuzz is a function for usage with https://github.com/dvyukov/go-fuzz
func Fuzz(data []byte) int {
	ua := string(data)
	parsed := parser.Parse(ua)
	if parsed.UserAgent.Family == unknownFamily &&
		parsed.Device.Family == unknownFamily &&
		parsed.Os.Family == unknownFamily {
		return 0
	}
	return 1
}
