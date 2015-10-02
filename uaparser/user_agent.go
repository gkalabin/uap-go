package uaparser

import (
	"regexp"
)

type UserAgent struct {
	Family string
	Major  string
	Minor  string
	Patch  string
}

func newUnknownUserAgent() *UserAgent {
	return &UserAgent{Family: familyUnknown}
}

type UserAgentPattern struct {
	Regexp            *regexp.Regexp
	Regex             string
	FamilyReplacement string
	V1Replacement     string
	V2Replacement     string
}

func (uaPattern *UserAgentPattern) Match(line string) (ok bool, ua *UserAgent) {
	matches := uaPattern.Regexp.FindStringSubmatch(line)
	if matches == nil {
		return false, nil
	}
	groupCount := uaPattern.Regexp.NumSubexp()

	ua = &UserAgent{}
	if len(uaPattern.FamilyReplacement) > 0 {
		ua.Family = singleMatchReplacement(uaPattern.FamilyReplacement, matches, 1)
	} else if groupCount >= 1 {
		ua.Family = matches[1]
	}

	if len(uaPattern.V1Replacement) > 0 {
		ua.Major = singleMatchReplacement(uaPattern.V1Replacement, matches, 2)
	} else if groupCount >= 2 {
		ua.Major = matches[2]
	}

	if len(uaPattern.V2Replacement) > 0 {
		ua.Minor = singleMatchReplacement(uaPattern.V2Replacement, matches, 3)
	} else if groupCount >= 3 {
		ua.Minor = matches[3]
		if groupCount >= 4 {
			ua.Patch = matches[4]
		}
	}
	return true, ua
}

func (ua *UserAgent) ToString() string {
	var str string
	if ua.Family != "" {
		str += ua.Family
	}
	version := ua.ToVersionString()
	if version != "" {
		str += " " + version
	}
	return str
}

func (ua *UserAgent) ToVersionString() string {
	var version string
	if ua.Major != "" {
		version += ua.Major
	}
	if ua.Minor != "" {
		version += "." + ua.Minor
	}
	if ua.Patch != "" {
		version += "." + ua.Patch
	}
	return version
}
