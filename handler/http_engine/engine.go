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

package http_engine

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Gen0P2RHandler[RT any](hf func(context.Context) (RT, error)) func(*gin.Context) {
	return func(gc *gin.Context) {
		r, err := hf(gc.Request.Context())
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, r)
	}
}

func Gen1P1RHandler[PT any](hf func(context.Context, PT) error, pf func(*gin.Context) (PT, error)) func(*gin.Context) {
	return func(gc *gin.Context) {
		p, err := pf(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		err = hf(gc.Request.Context(), p)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func Gen1P2RHandler[PT any, RT any](hf func(context.Context, PT) (RT, error), pf func(*gin.Context) (PT, error)) func(*gin.Context) {
	return func(gc *gin.Context) {
		p, err := pf(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		r, err := hf(gc.Request.Context(), p)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, r)
	}
}

func Gen2P1RHandler[P1T any, P2T any](hf func(context.Context, P1T, P2T) error, p1f func(*gin.Context) (P1T, error), p2f func(*gin.Context) (P2T, error)) func(*gin.Context) {
	return func(gc *gin.Context) {
		p1, err := p1f(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		p2, err := p2f(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		err = hf(gc.Request.Context(), p1, p2)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func GetUrlParam(gc *gin.Context, p string) (s string, err error) {
	s = gc.Param(p)
	if s == "" {
		err = fmt.Errorf("parameter '%s' not defined", p)
		gc.Status(http.StatusBadRequest)
	}
	return
}

func GetJsonBody[T any](gc *gin.Context) (r T, err error) {
	err = gc.ShouldBindJSON(&r)
	if err != nil {
		gc.Status(http.StatusBadRequest)
	}
	return
}

func GetQuery[T any](gc *gin.Context) (r T, err error) {
	err = gc.ShouldBindQuery(&r)
	if err != nil {
		gc.Status(http.StatusBadRequest)
	}
	return
}
