package uaparser

import (
	"regexp"
	"strings"
)

type Device struct {
	Family string
}

func newUnknownDevice() *Device {
	return &Device{Family: familyUnknown}
}

type DevicePattern struct {
	Regexp            *regexp.Regexp
	Regex             string
	RegexFlag         string
	BrandReplacement  string
	DeviceReplacement string
	ModelReplacement  string
}

func (dvcPattern *DevicePattern) Match(line string) (ok bool, device *Device) {
	matches := dvcPattern.Regexp.FindStringSubmatch(line)
	if matches == nil {
		return false, nil
	}
	device = &Device{}
	groupCount := dvcPattern.Regexp.NumSubexp()
	if len(dvcPattern.DeviceReplacement) > 0 {
		device.Family = allMatchesReplacement(dvcPattern.DeviceReplacement, matches)
	} else if groupCount >= 1 {
		device.Family = matches[1]
	}
	device.Family = strings.TrimSpace(device.Family)
	return true, device
}

func (dvc *Device) ToString() string {
	return dvc.Family
}
