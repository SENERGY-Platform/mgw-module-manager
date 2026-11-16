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

package deployments

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

type secretTypesTestClient struct {
	secretManagerClient
	secrets []external_models.SmSecret
	err     error
}

func (c *secretTypesTestClient) GetSecrets(_ context.Context) ([]external_models.SmSecret, error, int) {
	if c.err != nil {
		return nil, c.err, http.StatusServiceUnavailable
	}
	return c.secrets, nil, http.StatusOK
}

func TestHandler_GetSecretTypes(t *testing.T) {
	t.Run("types by id", func(t *testing.T) {
		h := New(nil, nil, nil, &secretTypesTestClient{secrets: []external_models.SmSecret{
			{ID: "s-1", Name: "cert", SecretType: "certificate"},
			{ID: "s-2", Name: "login", SecretType: "basic-auth"},
		}}, nil, Config{})
		got, err := h.GetSecretTypes(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{"s-1": "certificate", "s-2": "basic-auth"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("want %v, got %v", want, got)
		}
	})
	t.Run("error passed on", func(t *testing.T) {
		h := New(nil, nil, nil, &secretTypesTestClient{err: errors.New("unreachable")}, nil, Config{})
		if got, err := h.GetSecretTypes(context.Background()); err == nil || got != nil {
			t.Errorf("expected an error and no types, got %v, %v", got, err)
		}
	})
}
