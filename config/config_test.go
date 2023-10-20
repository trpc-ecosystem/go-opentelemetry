//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package config

import (
	"reflect"
	"testing"
)

func TestLogMode_MarshalText(t *testing.T) {
	tests := []struct {
		name     string
		m        LogMode
		wantText []byte
		wantErr  bool
	}{
		{"", LogModeDefault, []byte("oneline"), false},
		{"", LogModeDisable, []byte("disable"), false},
		{"", LogModeOneLine, []byte("oneline"), false},
		{"", LogModeMultiLine, []byte("multiline"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, err := tt.m.MarshalText()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotText, tt.wantText) {
				t.Errorf("MarshalText() gotText = %v, want %v", string(gotText), string(tt.wantText))
			}
		})
	}
}
