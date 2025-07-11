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

package traces

import (
	"reflect"
	"strconv"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/modern-go/reflect2"
)

var Integer64AsStringConfig = jsoniter.Config{
	EscapeHTML:                    false,
	MarshalFloatWith6Digits:       true, // will lose precession
	ObjectFieldMustBeSimpleString: true, // do not unescape object field
}.Froze()

func init() {
	Integer64AsStringConfig.RegisterExtension(&integer64AsStringExtension{})
}

type wrapCodec struct {
	encodeFunc func(ptr unsafe.Pointer, stream *jsoniter.Stream)
}

func (codec *wrapCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	codec.encodeFunc(ptr, stream)
}

func (codec *wrapCodec) IsEmpty(ptr unsafe.Pointer) bool {
	return false
}

type integer64AsStringExtension struct {
	jsoniter.DummyExtension
}

// CreateEncoder for javascript
func (e *integer64AsStringExtension) CreateEncoder(typ reflect2.Type) jsoniter.ValEncoder {
	encoder := e.int64AsStringEncoder(typ)
	if encoder != nil {
		return encoder
	}

	return nil
}

// CreateMapKeyEncoder specify encoder for int64,uint64 type key avoiding jsoniter using numericMapKeyEncoder
func (e *integer64AsStringExtension) CreateMapKeyEncoder(typ reflect2.Type) jsoniter.ValEncoder {
	encoder := e.int64AsStringEncoder(typ)
	if encoder != nil {
		return encoder
	}

	return nil
}

// int64AsStringEncoder encode int64, uint64 to string
func (e *integer64AsStringExtension) int64AsStringEncoder(typ reflect2.Type) jsoniter.ValEncoder {
	if typ.Kind() == reflect.Int64 {
		return &wrapCodec{
			encodeFunc: func(ptr unsafe.Pointer, stream *jsoniter.Stream) {
				stream.WriteString(strconv.FormatInt(*(*int64)(ptr), 10))
			},
		}
	}

	if typ.Kind() == reflect.Uint64 {
		return &wrapCodec{
			encodeFunc: func(ptr unsafe.Pointer, stream *jsoniter.Stream) {
				stream.WriteString(strconv.FormatUint(*(*uint64)(ptr), 10))
			},
		}
	}

	return nil
}
