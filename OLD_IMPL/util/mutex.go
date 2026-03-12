/*
 * Copyright 2023 InfAI (CC SES)
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

package util

import (
	"errors"
	"sync"
)

type RWMutex struct {
	mu     sync.RWMutex
	reason string
}

func (m *RWMutex) TryRLock() error {
	if !m.mu.TryRLock() {
		return errors.New(m.reason)
	}
	return nil
}

func (m *RWMutex) RUnlock() {
	m.mu.RUnlock()
}

func (m *RWMutex) TryLock(reason string) error {
	if !m.mu.TryLock() {
		return errors.New(m.reason)
	}
	m.reason = reason
	return nil
}

func (m *RWMutex) Unlock() {
	m.reason = ""
	m.mu.Unlock()
}
