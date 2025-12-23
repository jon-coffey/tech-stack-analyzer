// Copyright 2025 Google LLC (adapted from deps.dev)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package semver

import (
	"fmt"
	"strconv"
	"strings"
)

// pypiSystem implements PEP 440 version parsing
type pypiSystem struct{}

func (s *pypiSystem) Name() string {
	return "PyPI"
}

func (s *pypiSystem) Parse(version string) (Version, error) {
	return parsePyPIVersion(version)
}

// PyPIVersion represents a PEP 440 compliant version
// Based on: https://www.python.org/dev/peps/pep-0440/
type PyPIVersion struct {
	original string
	epoch    int
	release  []int
	pre      *preRelease
	post     *int
	dev      *int
	local    string
}

type preRelease struct {
	phase  string // "a", "b", "rc"
	number int
}

// parsePyPIVersion parses a PEP 440 version string
func parsePyPIVersion(version string) (*PyPIVersion, error) {
	if version == "" {
		return nil, parseError("PyPI", version, "empty version string")
	}

	v := &PyPIVersion{original: version}
	s := strings.ToLower(strings.TrimSpace(version))

	// Parse epoch (e.g., "1!")
	if idx := strings.IndexByte(s, '!'); idx > 0 {
		epochStr := s[:idx]
		epoch, err := strconv.Atoi(epochStr)
		if err != nil {
			return nil, parseError("PyPI", version, fmt.Sprintf("invalid epoch: %s", epochStr))
		}
		v.epoch = epoch
		s = s[idx+1:]
	}

	// Parse local version (e.g., "+local.version")
	if idx := strings.IndexByte(s, '+'); idx >= 0 {
		v.local = s[idx+1:]
		s = s[:idx]
	}

	// Parse dev release (e.g., ".dev0")
	if idx := strings.Index(s, ".dev"); idx >= 0 {
		devStr := s[idx+4:]
		if devStr != "" {
			dev, err := strconv.Atoi(devStr)
			if err != nil {
				return nil, parseError("PyPI", version, fmt.Sprintf("invalid dev number: %s", devStr))
			}
			v.dev = &dev
		} else {
			zero := 0
			v.dev = &zero
		}
		s = s[:idx]
	} else if idx := strings.Index(s, "dev"); idx >= 0 {
		// Handle "dev" without dot
		devStr := s[idx+3:]
		if devStr != "" {
			dev, err := strconv.Atoi(devStr)
			if err != nil {
				return nil, parseError("PyPI", version, fmt.Sprintf("invalid dev number: %s", devStr))
			}
			v.dev = &dev
		} else {
			zero := 0
			v.dev = &zero
		}
		s = s[:idx]
	}

	// Parse post release (e.g., ".post0" or "-0")
	if idx := strings.Index(s, ".post"); idx >= 0 {
		postStr := s[idx+5:]
		if postStr != "" {
			post, err := strconv.Atoi(postStr)
			if err != nil {
				return nil, parseError("PyPI", version, fmt.Sprintf("invalid post number: %s", postStr))
			}
			v.post = &post
		} else {
			zero := 0
			v.post = &zero
		}
		s = s[:idx]
	} else if idx := strings.Index(s, "post"); idx >= 0 {
		postStr := s[idx+4:]
		if postStr != "" {
			post, err := strconv.Atoi(postStr)
			if err != nil {
				return nil, parseError("PyPI", version, fmt.Sprintf("invalid post number: %s", postStr))
			}
			v.post = &post
		} else {
			zero := 0
			v.post = &zero
		}
		s = s[:idx]
	} else if idx := strings.LastIndexByte(s, '-'); idx >= 0 {
		// Handle post release with dash (but not underscore, which is a separator)
		postStr := s[idx+1:]
		if postStr != "" && isAllDigits(postStr) {
			post, err := strconv.Atoi(postStr)
			if err == nil {
				v.post = &post
				s = s[:idx]
			}
		}
	}

	// Parse pre-release (alpha, beta, rc)
	preIdx := -1
	prePhase := ""

	for _, phase := range []string{"rc", "c", "beta", "b", "alpha", "a"} {
		if idx := strings.Index(s, phase); idx >= 0 {
			if preIdx == -1 || idx < preIdx {
				preIdx = idx
				prePhase = phase
			}
		}
	}

	if preIdx >= 0 {
		preNumStr := s[preIdx+len(prePhase):]
		s = s[:preIdx]

		// Normalize phase names
		switch prePhase {
		case "alpha", "a":
			prePhase = "a"
		case "beta", "b":
			prePhase = "b"
		case "c", "rc":
			prePhase = "rc"
		}

		preNum := 0
		if preNumStr != "" {
			var err error
			preNum, err = strconv.Atoi(preNumStr)
			if err != nil {
				return nil, parseError("PyPI", version, fmt.Sprintf("invalid pre-release number: %s", preNumStr))
			}
		}

		v.pre = &preRelease{phase: prePhase, number: preNum}
	}

	// Parse release numbers (e.g., "1.2.3")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return nil, parseError("PyPI", version, "no release numbers found")
	}

	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '.' || r == '_' || r == '-'
	})

	for _, part := range parts {
		if part == "" {
			continue
		}
		num, err := strconv.Atoi(part)
		if err != nil {
			return nil, parseError("PyPI", version, fmt.Sprintf("invalid release number: %s", part))
		}
		v.release = append(v.release, num)
	}

	if len(v.release) == 0 {
		return nil, parseError("PyPI", version, "no valid release numbers")
	}

	return v, nil
}

// Canon returns the canonical string representation of the version
func (v *PyPIVersion) Canon(includeEpoch bool) string {
	var b strings.Builder

	// Epoch
	if includeEpoch && v.epoch > 0 {
		b.WriteString(strconv.Itoa(v.epoch))
		b.WriteByte('!')
	}

	// Release
	for i, num := range v.release {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(strconv.Itoa(num))
	}

	// Pre-release
	if v.pre != nil {
		b.WriteString(v.pre.phase)
		b.WriteString(strconv.Itoa(v.pre.number))
	}

	// Post-release
	if v.post != nil {
		b.WriteString(".post")
		b.WriteString(strconv.Itoa(*v.post))
	}

	// Dev-release
	if v.dev != nil {
		b.WriteString(".dev")
		b.WriteString(strconv.Itoa(*v.dev))
	}

	// Local version
	if v.local != "" {
		b.WriteByte('+')
		b.WriteString(v.local)
	}

	return b.String()
}

// String returns the original version string
func (v *PyPIVersion) String() string {
	return v.original
}

// Compare compares this version with another version
func (v *PyPIVersion) Compare(other Version) int {
	o, ok := other.(*PyPIVersion)
	if !ok {
		// Can't compare different version types
		return 0
	}

	// Compare epoch
	if v.epoch != o.epoch {
		if v.epoch < o.epoch {
			return -1
		}
		return 1
	}

	// Compare release
	minLen := len(v.release)
	if len(o.release) < minLen {
		minLen = len(o.release)
	}

	for i := 0; i < minLen; i++ {
		if v.release[i] != o.release[i] {
			if v.release[i] < o.release[i] {
				return -1
			}
			return 1
		}
	}

	// If all compared parts are equal, longer version is greater
	if len(v.release) != len(o.release) {
		if len(v.release) < len(o.release) {
			return -1
		}
		return 1
	}

	// Compare pre-release (no pre-release > has pre-release)
	if v.pre == nil && o.pre != nil {
		return 1
	}
	if v.pre != nil && o.pre == nil {
		return -1
	}
	if v.pre != nil && o.pre != nil {
		// Compare phase (a < b < rc)
		phaseOrder := map[string]int{"a": 1, "b": 2, "rc": 3}
		vPhase := phaseOrder[v.pre.phase]
		oPhase := phaseOrder[o.pre.phase]
		if vPhase != oPhase {
			if vPhase < oPhase {
				return -1
			}
			return 1
		}
		// Compare number
		if v.pre.number != o.pre.number {
			if v.pre.number < o.pre.number {
				return -1
			}
			return 1
		}
	}

	// Compare post-release (no post < has post)
	if v.post == nil && o.post != nil {
		return -1
	}
	if v.post != nil && o.post == nil {
		return 1
	}
	if v.post != nil && o.post != nil {
		if *v.post != *o.post {
			if *v.post < *o.post {
				return -1
			}
			return 1
		}
	}

	// Compare dev-release (no dev > has dev)
	if v.dev == nil && o.dev != nil {
		return 1
	}
	if v.dev != nil && o.dev == nil {
		return -1
	}
	if v.dev != nil && o.dev != nil {
		if *v.dev != *o.dev {
			if *v.dev < *o.dev {
				return -1
			}
			return 1
		}
	}

	// Local versions are not compared in PEP 440
	return 0
}

// isAllDigits returns true if the string contains only digits
func isAllDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return len(s) > 0
}
