//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package codes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodeTypeMappingDescription_UnmarshalText(t *testing.T) {
	type fields struct {
		CodeType    string
		Description string
	}
	type args struct {
		text []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "length exceeds maximum length",
			args: args{
				text: []byte("success|Length exceeds maximum length"),
			},
			fields: fields{
				CodeType:    CodeTypeSuccess.String(),
				Description: "Length exceeds maximum length",
			},
		},
		{
			name: "codeType default value",
			args: args{
				text: []byte("123|备注"),
			},
			fields: fields{
				CodeType:    CodeTypeSuccess.String(),
				Description: "备注",
			},
		},
		{
			name: "empty description",
			args: args{
				text: []byte("success"),
			},
			fields: fields{
				CodeType:    CodeTypeSuccess.String(),
				Description: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &CodeTypeMappingDescription{}
			if err := d.UnmarshalText(tt.args.text); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.EqualValues(t, &CodeTypeMappingDescription{
				CodeType:    tt.fields.CodeType,
				Description: tt.fields.Description,
			}, d)
		})
	}
}
