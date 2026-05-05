/*
 * Copyright 2026 InfAI (CC SES)
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

package mutex_map

import "sync"

type RWMutexMap struct {
	muMap map[string]*sync.RWMutex
	mu    sync.Mutex
}

func New() *RWMutexMap {
	return &RWMutexMap{
		muMap: make(map[string]*sync.RWMutex),
	}
}

func (m *RWMutexMap) Get(key string) *sync.RWMutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	mu, ok := m.muMap[key]
	if !ok {
		mu = &sync.RWMutex{}
		m.muMap[key] = mu
	}
	return mu
}

func (m *RWMutexMap) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.muMap, key)
}
