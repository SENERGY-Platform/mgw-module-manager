/*
 * Copyright 2022 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1

import (
	"code.cloudfoundry.org/bytefmt"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/fs"
	"module-manager/manager/itf"
	"strconv"
	"strings"
	"time"
)

func (p *Port) IsRange() bool {
	if strings.Contains(string(*p), "-") {
		return true
	}
	return false
}

func (p *Port) Range() (ports []int) {
	parts := strings.Split(string(*p), "-")
	start, _ := strconv.ParseInt(parts[0], 10, 64)
	if len(parts) > 1 {
		end, _ := strconv.ParseInt(parts[1], 10, 64)
		for i := start; i <= end; i++ {
			ports = append(ports, int(i))
		}
	} else {
		ports = append(ports, int(start))
	}
	return
}

func (p *Port) Int() int {
	i, _ := strconv.ParseInt(string(*p), 10, 64)
	return int(i)
}

func (p *Port) UnmarshalYAML(yn *yaml.Node) error {
	var it any
	if err := yn.Decode(&it); err != nil {
		return err
	}
	switch v := it.(type) {
	case int:
		*p = Port(strconv.FormatInt(int64(v), 10))
	case string:
		parts := strings.Split(v, "-")
		if len(parts) > 2 {
			return fmt.Errorf("invalid port range: %s", v)
		}
		for i := 0; i < len(parts); i++ {
			_, err := strconv.ParseInt(parts[i], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid port: %s", v)
			}
		}
		*p = Port(v)
	default:
		return fmt.Errorf("invlid port: %v", v)
	}
	return nil
}

func (fb *ByteFmt) UnmarshalYAML(yn *yaml.Node) error {
	var it any
	if err := yn.Decode(&it); err != nil {
		return err
	}
	switch v := it.(type) {
	case int:
		*fb = ByteFmt(v)
	case string:
		bytes, err := bytefmt.ToBytes(v)
		if err != nil {
			return fmt.Errorf("invalid size: %s", err)
		}
		*fb = ByteFmt(bytes)
	default:
		return fmt.Errorf("invalid size: %v", v)
	}
	return nil
}

func (d *Duration) UnmarshalYAML(yn *yaml.Node) error {
	var s string
	if err := yn.Decode(&s); err != nil {
		return err
	}
	if dur, err := time.ParseDuration(s); err != nil {
		return err
	} else {
		d.Duration = dur
	}
	return nil
}

func (m *FileMode) UnmarshalYAML(yn *yaml.Node) error {
	var s string
	if err := yn.Decode(&s); err != nil {
		return err
	}
	i, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return err
	}
	m.FileMode = fs.FileMode(i)
	return nil
}

func Decode(yn *yaml.Node) (itf.ModFileModule, error) {
	var m Module
	if err := yn.Decode(&m); err != nil {
		return m, err
	}
	return m, nil
}
