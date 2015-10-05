package uaparser

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"gopkg.in/yaml.v2"
)

const familyUnknown = "Other"

type Parser struct {
	DataFileVersion   string
	UserAgentPatterns []UserAgentPattern
	OsPatterns        []OsPattern
	DevicePatterns    []DevicePattern
}

type Client struct {
	ID        string // some synthetic string we can search by later
	UserAgent *UserAgent
	Os        *Os
	Device    *Device
}

var exportedNameRegex = regexp.MustCompile("[0-9A-Za-z]+")

func GetExportedName(src string) string {
	byteSrc := []byte(src)
	chunks := exportedNameRegex.FindAll(byteSrc, -1)
	for idx, val := range chunks {
		chunks[idx] = bytes.Title(val)
	}
	return string(bytes.Join(chunks, nil))
}

func ToStruct(interfaceArr []map[string]string, typeInterface interface{}, returnVal *[]interface{}) {
	structArr := make([]interface{}, 0)
	for _, interfaceMap := range interfaceArr {
		structValPtr := reflect.New(reflect.TypeOf(typeInterface))
		structVal := structValPtr.Elem()
		for key, value := range interfaceMap {
			structVal.FieldByName(GetExportedName(key)).SetString(value)
		}
		structArr = append(structArr, structVal.Interface())
	}
	*returnVal = structArr
}

func New(regexFile string) (*Parser, error) {
	parser := new(Parser)

	data, err := ioutil.ReadFile(regexFile)
	if nil != err {
		return nil, err
	}

	return parser.newFromBytes(data)
}

func NewWithVersion(regexFile, fileVersion string) (*Parser, error) {
	if strings.Contains(fileVersion, " ") {
		return nil, fmt.Errorf("File version '%s' contains a space, which is used while constructing ID", fileVersion)
	}
	parser, err := New(regexFile)
	if err != nil {
		return nil, err
	}
	parser.DataFileVersion = fileVersion
	return parser, nil
}

func NewFromBytes(regexBytes []byte) (*Parser, error) {
	parser := new(Parser)

	return parser.newFromBytes(regexBytes)
}

func (parser *Parser) newFromBytes(data []byte) (*Parser, error) {
	m := make(map[string][]map[string]string)
	err := yaml.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup

	uaPatternType := new(UserAgentPattern)
	var uaInterfaces []interface{}
	var uaPatterns []UserAgentPattern

	wg.Add(1)
	go func() {
		ToStruct(m["user_agent_parsers"], *uaPatternType, &uaInterfaces)
		uaPatterns = make([]UserAgentPattern, len(uaInterfaces))
		for i, inter := range uaInterfaces {
			uaPatterns[i] = inter.(UserAgentPattern)
			uaPatterns[i].Regexp = regexp.MustCompile(uaPatterns[i].Regex)
		}
		wg.Done()
	}()

	osPatternType := new(OsPattern)
	var osInterfaces []interface{}
	var osPatterns []OsPattern

	wg.Add(1)
	go func() {
		ToStruct(m["os_parsers"], *osPatternType, &osInterfaces)
		osPatterns = make([]OsPattern, len(osInterfaces))
		for i, inter := range osInterfaces {
			osPatterns[i] = inter.(OsPattern)
			osPatterns[i].Regexp = regexp.MustCompile(osPatterns[i].Regex)
		}
		wg.Done()
	}()

	dvcPatternType := new(DevicePattern)
	var dvcInterfaces []interface{}
	var dvcPatterns []DevicePattern

	wg.Add(1)
	go func() {
		ToStruct(m["device_parsers"], *dvcPatternType, &dvcInterfaces)
		dvcPatterns = make([]DevicePattern, len(dvcInterfaces))
		for i, inter := range dvcInterfaces {
			dvcPatterns[i] = inter.(DevicePattern)
			flags := ""
			if strings.Contains(dvcPatterns[i].RegexFlag, "i") {
				flags = "(?i)"
			}
			regexString := fmt.Sprintf("%s%s", flags, dvcPatterns[i].Regex)
			dvcPatterns[i].Regexp = regexp.MustCompile(regexString)
		}
		wg.Done()
	}()

	wg.Wait()

	parser.UserAgentPatterns = uaPatterns
	parser.OsPatterns = osPatterns
	parser.DevicePatterns = dvcPatterns

	return parser, nil
}

func (parser *Parser) ParseUserAgent(line string) *UserAgent {
	ua, _ := parser.doParseUserAgent(line)
	return ua
}

func (parser *Parser) ParseOs(line string) *Os {
	os, _ := parser.doParseOs(line)
	return os
}

func (parser *Parser) ParseDevice(line string) *Device {
	os, _ := parser.doParseDevice(line)
	return os
}

func (parser *Parser) doParseUserAgent(line string) (ua *UserAgent, idx int) {
	for idx = range parser.UserAgentPatterns {
		if ua, ok := parser.UserAgentPatterns[idx].Match(line); ok {
			return ua, idx
		}
	}
	return newUnknownUserAgent(), -1
}

func (parser *Parser) doParseOs(line string) (os *Os, idx int) {
	for idx = range parser.OsPatterns {
		if os, ok := parser.OsPatterns[idx].Match(line); ok {
			return os, idx
		}
	}
	return newUnknownOs(), -1
}

func (parser *Parser) doParseDevice(line string) (device *Device, idx int) {
	for idx = range parser.DevicePatterns {
		if device, ok := parser.DevicePatterns[idx].Match(line); ok {
			return device, idx
		}
	}
	return newUnknownDevice(), -1
}

func (parser *Parser) Parse(line string) *Client {
	ua, uaIdx := parser.doParseUserAgent(line)
	os, osIdx := parser.doParseOs(line)
	dev, devIdx := parser.doParseDevice(line)
	return &Client{
		ID:        fmt.Sprintf("%s %d %d %d", parser.DataFileVersion, uaIdx, osIdx, devIdx),
		UserAgent: ua,
		Os:        os,
		Device:    dev,
	}
}

func (parser *Parser) FindByID(id, line string) *Client {
	idParts := strings.Split(id, " ")
	if len(idParts) != 4 {
		return nil
	}
	dataFileVersion := idParts[0]
	if dataFileVersion != parser.DataFileVersion {
		return nil
	}
	// restore ua
	uaIdx, err := strconv.Atoi(idParts[1])
	if err != nil || uaIdx < -1 || uaIdx >= len(parser.UserAgentPatterns) {
		return nil
	}
	var ua *UserAgent
	if uaIdx == -1 {
		ua = newUnknownUserAgent()
	} else {
		matched, ok := parser.UserAgentPatterns[uaIdx].Match(line)
		if !ok {
			return nil
		}
		ua = matched
	}
	// restore os
	osIdx, err := strconv.Atoi(idParts[2])
	if err != nil || osIdx < -1 || osIdx >= len(parser.OsPatterns) {
		return nil
	}
	var os *Os
	if osIdx == -1 {
		os = newUnknownOs()
	} else {
		matched, ok := parser.OsPatterns[osIdx].Match(line)
		if !ok {
			return nil
		}
		os = matched
	}
	// restore device
	deviceIdx, err := strconv.Atoi(idParts[3])
	if err != nil || deviceIdx < -1 || deviceIdx >= len(parser.DevicePatterns) {
		return nil
	}
	var device *Device
	if deviceIdx == -1 {
		device = newUnknownDevice()
	} else {
		matched, ok := parser.DevicePatterns[deviceIdx].Match(line)
		if !ok {
			return nil
		}
		device = matched
	}
	return &Client{
		ID:        id,
		UserAgent: ua,
		Os:        os,
		Device:    device,
	}
}

func singleMatchReplacement(replacement string, matches []string, idx int) string {
	token := "$" + strconv.Itoa(idx)
	if strings.Contains(replacement, token) {
		return strings.Replace(replacement, token, matches[idx], -1)
	}
	return replacement
}

// allMatchesReplacement replaces all tokens in format $<digit> (like $1 or $12) with values
// at corresponding indexes (NOT POSITIONS, so $1 will be replaced with v[1], NOT v[0]) in the provided array.
// If array doesn't have value at the index (when array length is less than the value), it remains unchanged in the string
func allMatchesReplacement(pattern string, matches []string) string {
	var output bytes.Buffer
	readingToken := false
	var readToken bytes.Buffer
	writeTokenValue := func() {
		if !readingToken {
			return
		}
		if readToken.Len() == 0 {
			output.WriteRune('$')
			return
		}
		idx, err := strconv.Atoi(readToken.String())
		// index is out of range when value is too big for int or when it's zero (or less) or greater than array length
		indexOutOfRange := (err != nil && err.(*strconv.NumError).Err != strconv.ErrRange) || idx <= 0 || idx >= len(matches)
		if indexOutOfRange {
			output.WriteRune('$')
			output.Write(readToken.Bytes())
			readToken.Reset()
			return
		}
		if err != nil {
			// should never happen
			panic(err)
		}
		output.WriteString(matches[idx])
		readToken.Reset()
	}
	for _, r := range pattern {
		if !readingToken && r == '$' {
			readingToken = true
			continue
		}
		if !readingToken {
			output.WriteRune(r)
			continue
		}
		if unicode.IsDigit(r) {
			readToken.WriteRune(r)
			continue
		}
		writeTokenValue()
		readingToken = (r == '$')
		if !readingToken {
			output.WriteRune(r)
		}
	}
	writeTokenValue()
	return output.String()
}
