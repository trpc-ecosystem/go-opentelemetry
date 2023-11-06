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

package otelzap

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/otel/sdk/resource"
	commonproto "go.opentelemetry.io/proto/otlp/common/v1"
	logsproto "go.opentelemetry.io/proto/otlp/logs/v1"
	resourceproto "go.opentelemetry.io/proto/otlp/resource/v1"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"

	apilog "trpc.group/trpc-go/go-opentelemetry/api/log"
	sdklog "trpc.group/trpc-go/go-opentelemetry/sdk/log"
)

var _ zapcore.Encoder = (*encoder)(nil)

type encoder struct {
	*zapcore.EncoderConfig

	record *logsproto.LogRecord
	kvs    []*commonproto.KeyValue

	buf *buffer.Buffer
}

var bufPool = buffer.NewPool()

// NewEncoder return zapcore.Encoder
func NewEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	return &encoder{
		EncoderConfig: &cfg,
		record:        &logsproto.LogRecord{},
		buf:           bufPool.Get(),
	}
}

var errUnimplemented = errors.New("errUnimplemented")

func (e *encoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	return errUnimplemented
}

func (e *encoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	return errUnimplemented
}

func (e *encoder) AddBinary(key string, value []byte) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_StringValue{
				StringValue: string(value),
			},
		},
	})
}

func (e *encoder) AddByteString(key string, value []byte) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_StringValue{
				StringValue: string(value),
			},
		},
	})
}

func (e *encoder) AddBool(key string, value bool) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_BoolValue{
				BoolValue: value,
			},
		},
	})
}

func (e *encoder) AddComplex128(key string, value complex128) {
}

func (e *encoder) AddComplex64(key string, value complex64) {
}

func (e *encoder) AddDuration(key string, value time.Duration) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_StringValue{
				StringValue: value.String(),
			},
		},
	})
}

func (e *encoder) AddFloat64(key string, value float64) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_DoubleValue{
				DoubleValue: value,
			},
		},
	})
}

func (e *encoder) AddFloat32(key string, value float32) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_DoubleValue{
				DoubleValue: float64(value),
			},
		},
	})
}

func (e *encoder) AddInt(key string, value int) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

func (e *encoder) AddInt64(key string, value int64) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: value,
			},
		},
	})
}

func (e *encoder) AddInt32(key string, value int32) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

func (e *encoder) AddInt16(key string, value int16) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

func (e *encoder) AddInt8(key string, value int8) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

func (e *encoder) AddString(key, value string) {
	if key == "sampled" {
		if value == strconv.FormatBool(true) {
			e.record.Flags = 1
		}
		return
	}
	if key == "traceID" {
		e.record.TraceId, _ = hex.DecodeString(value)
		return
	}
	if key == "spanID" {
		e.record.SpanId, _ = hex.DecodeString(value)
		return
	}
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_StringValue{
				StringValue: value,
			},
		},
	})
}

func (e *encoder) AddTime(key string, value time.Time) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_StringValue{
				StringValue: value.String(),
			},
		},
	})
}

func (e *encoder) AddUint(key string, value uint) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

func (e *encoder) AddUint64(key string, value uint64) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

func (e *encoder) AddUint32(key string, value uint32) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

func (e *encoder) AddUint16(key string, value uint16) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

func (e *encoder) AddUint8(key string, value uint8) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

func (e *encoder) AddUintptr(key string, value uintptr) {

}

func (e *encoder) AddReflected(key string, value interface{}) error {
	return nil
}

func (e *encoder) OpenNamespace(key string) {
}

func (e *encoder) Clone() zapcore.Encoder {
	enc := &encoder{
		EncoderConfig: e.EncoderConfig,
		kvs:           make([]*commonproto.KeyValue, 0, len(e.kvs)),
		record:        e.record,
		buf:           e.buf,
	}
	enc.kvs = append(enc.kvs, e.kvs...)
	return enc
}

func (e *encoder) convertField(f zapcore.Field) {
	fn, ok := convertFuncs[f.Type]
	if !ok {
		panic(fmt.Sprintf("unknown field type: %v", f))
	}
	fn(e, f)
}

func (e *encoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	e.record.SeverityText = entry.Level.String()
	for _, f := range fields {
		e.convertField(f)
	}
	e.record.Attributes = append(e.record.Attributes, e.kvs...)
	e.record.Body = &commonproto.AnyValue{
		Value: &commonproto.AnyValue_StringValue{
			StringValue: entry.Message,
		},
	}
	e.record.TimeUnixNano = uint64(entry.Time.UnixNano())

	data, err := proto.Marshal(e.record)
	if err != nil {
		return nil, err
	}

	_, err = e.buf.Write(data)
	if err != nil {
		return nil, err
	}

	return e.buf, nil
}

var _ zapcore.WriteSyncer = (*writeSyncer)(nil)
var _ zapcore.WriteSyncer = (*jsonWriteSyncer)(nil)

type jsonWriteSyncer struct {
}

func (jw *jsonWriteSyncer) Sync() error {
	return nil
}

func (jw *jsonWriteSyncer) Write(p []byte) (n int, err error) {
	raw := make(map[string]interface{})
	err = jsoniter.ConfigFastest.Unmarshal(p, &raw)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

type writeSyncer struct {
	rs        *resource.Resource
	processor *sdklog.BatchProcessor
}

func (w *writeSyncer) Write(p []byte) (n int, err error) {
	record := &logsproto.LogRecord{}
	err = proto.Unmarshal(p, record)
	if err != nil {
		return 0, err
	}

	rs := &resourceproto.Resource{}
	for _, kv := range w.rs.Attributes() {
		rs.Attributes = append(rs.Attributes, &commonproto.KeyValue{
			Key: string(kv.Key),
			Value: &commonproto.AnyValue{
				Value: &commonproto.AnyValue_StringValue{StringValue: kv.Value.Emit()},
			},
		})
	}

	rl := &logsproto.ResourceLogs{
		Resource: rs,
		ScopeLogs: []*logsproto.ScopeLogs{
			{
				LogRecords: []*logsproto.LogRecord{record},
			},
		},
	}
	w.processor.Enqueue(rl)
	return len(p), nil
}

func (w *writeSyncer) Sync() error {
	return nil
}

// NewWriteSyncer create writeSyncer instance
func NewWriteSyncer(p *sdklog.BatchProcessor, rs *resource.Resource) zapcore.WriteSyncer {
	return &writeSyncer{
		processor: p,
		rs:        rs,
	}
}

// NewJSONWriteSyncer get jsonWriteSyncer instance
func NewJSONWriteSyncer() zapcore.WriteSyncer {
	return &jsonWriteSyncer{}
}

// NewCore create zap core instance
func NewCore(opts ...sdklog.LoggerOption) zapcore.Core {
	o := &sdklog.LoggerOptions{
		LevelEnabled: apilog.DebugLevel,
	}
	for _, opt := range opts {
		opt(o)
	}
	return zapcore.NewCore(NewEncoder(zap.NewProductionEncoderConfig()),
		NewWriteSyncer(o.Processor, o.Resource), toLevelEnabler(o.LevelEnabled))
}

// NewBatchCore batch create zap core instances
func NewBatchCore(syncer *BatchWriteSyncer, opts ...sdklog.LoggerOption) zapcore.Core {
	o := &sdklog.LoggerOptions{
		LevelEnabled: apilog.DebugLevel,
	}
	for _, opt := range opts {
		opt(o)
	}
	return zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig()),
		syncer, toLevelEnabler(o.LevelEnabled))
}

// NewBatchCoreAndLevel NewBatchCore with log level returned
func NewBatchCoreAndLevel(syncer *BatchWriteSyncer, opts ...sdklog.LoggerOption) (zapcore.Core, zap.AtomicLevel) {
	o := &sdklog.LoggerOptions{
		LevelEnabled: apilog.DebugLevel,
	}
	for _, opt := range opts {
		opt(o)
	}
	lvl := zap.NewAtomicLevelAt(toLevelEnabler(o.LevelEnabled))
	return zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig()),
		syncer, lvl), lvl
}

func encoderConfig() zapcore.EncoderConfig {
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeCaller = zapcore.FullCallerEncoder
	return cfg
}

// NewJSONCore get zap core with JSON encoder
func NewJSONCore() zapcore.Core {
	return zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		NewJSONWriteSyncer(), zapcore.DebugLevel)
}

func toLevelEnabler(level apilog.Level) zapcore.Level {
	switch level {
	case apilog.TraceLevel:
		return zap.DebugLevel
	case apilog.DebugLevel:
		return zap.DebugLevel
	case apilog.InfoLevel:
		return zap.InfoLevel
	case apilog.WarnLevel:
		return zap.WarnLevel
	case apilog.ErrorLevel:
		return zap.ErrorLevel
	case apilog.FatalLevel:
		return zap.FatalLevel
	default:
		return zap.ErrorLevel
	}
}

var convertFuncs = map[zapcore.FieldType]func(e *encoder, f zapcore.Field){
	zapcore.ArrayMarshalerType: func(e *encoder, f zapcore.Field) {
		_ = e.AddArray(f.Key, f.Interface.(zapcore.ArrayMarshaler))
	},
	zapcore.ObjectMarshalerType: func(e *encoder, f zapcore.Field) {
		_ = e.AddObject(f.Key, f.Interface.(zapcore.ObjectMarshaler))
	},
	zapcore.BinaryType: func(e *encoder, f zapcore.Field) {
		e.AddBinary(f.Key, f.Interface.([]byte))
	},
	zapcore.BoolType: func(e *encoder, f zapcore.Field) {
		e.AddBool(f.Key, f.Integer == 1)
	},
	zapcore.ByteStringType: func(e *encoder, f zapcore.Field) {
		e.AddByteString(f.Key, f.Interface.([]byte))
	},
	zapcore.Complex128Type: func(e *encoder, f zapcore.Field) {

		e.AddComplex128(f.Key, f.Interface.(complex128))
	},
	zapcore.Complex64Type: func(e *encoder, f zapcore.Field) {
		e.AddComplex64(f.Key, f.Interface.(complex64))
	},
	zapcore.DurationType: func(e *encoder, f zapcore.Field) {
		e.AddDuration(f.Key, time.Duration(f.Integer))
	},
	zapcore.Float64Type: func(e *encoder, f zapcore.Field) {
		e.AddFloat64(f.Key, math.Float64frombits(uint64(f.Integer)))
	},
	zapcore.Float32Type: func(e *encoder, f zapcore.Field) {
		e.AddFloat32(f.Key, math.Float32frombits(uint32(f.Integer)))
	},
	zapcore.Int64Type: func(e *encoder, f zapcore.Field) {
		e.AddInt64(f.Key, f.Integer)
	},
	zapcore.Int32Type: func(e *encoder, f zapcore.Field) {
		e.AddInt32(f.Key, int32(f.Integer))
	},
	zapcore.Int16Type: func(e *encoder, f zapcore.Field) {
		e.AddInt16(f.Key, int16(f.Integer))
	},
	zapcore.Int8Type: func(e *encoder, f zapcore.Field) {
		e.AddInt8(f.Key, int8(f.Integer))
	},
	zapcore.StringType: func(e *encoder, f zapcore.Field) {
		e.AddString(f.Key, f.String)
	},
	zapcore.TimeType: func(e *encoder, f zapcore.Field) {
		if f.Interface != nil {
			e.AddTime(f.Key, time.Unix(0, f.Integer).In(f.Interface.(*time.Location)))
		} else {
			// Fall back to UTC if location is nil.
			e.AddTime(f.Key, time.Unix(0, f.Integer))
		}
	},
	zapcore.TimeFullType: func(e *encoder, f zapcore.Field) {
		e.AddTime(f.Key, f.Interface.(time.Time))
	},
	zapcore.Uint64Type: func(e *encoder, f zapcore.Field) {
		e.AddUint64(f.Key, uint64(f.Integer))
	},
	zapcore.Uint32Type: func(e *encoder, f zapcore.Field) {
		e.AddUint32(f.Key, uint32(f.Integer))
	},
	zapcore.Uint16Type: func(e *encoder, f zapcore.Field) {
		e.AddUint16(f.Key, uint16(f.Integer))
	},
	zapcore.Uint8Type: func(e *encoder, f zapcore.Field) {
		e.AddUint8(f.Key, uint8(f.Integer))
	},
	zapcore.UintptrType: func(e *encoder, f zapcore.Field) {
		e.AddUintptr(f.Key, uintptr(f.Integer))
	},
	zapcore.ReflectType: func(e *encoder, f zapcore.Field) {
		_ = e.AddReflected(f.Key, f.Interface)
	},
	zapcore.NamespaceType: func(e *encoder, f zapcore.Field) {
		e.OpenNamespace(f.Key)
	},
	zapcore.StringerType: func(e *encoder, f zapcore.Field) {
	},
	zapcore.ErrorType: func(e *encoder, f zapcore.Field) {
		e.AddString(f.Key, f.Interface.(error).Error())
	},
	zapcore.SkipType: func(e *encoder, f zapcore.Field) {
	},
}
