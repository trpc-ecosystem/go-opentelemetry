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

package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevel_UnmarshalText(t *testing.T) {
	var s Level
	err := s.UnmarshalText([]byte("trace"))
	assert.NoError(t, err)

	assert.Equal(t, s, TraceLevel)

	err = s.UnmarshalText([]byte("unknown"))
	assert.NoError(t, err)
}
