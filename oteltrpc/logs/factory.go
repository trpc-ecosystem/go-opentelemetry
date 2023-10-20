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

package logs

import (
	"errors"

	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

const pluginType = "log"

func init() {
	plugin.Register("default", &Factory{})
}

// Factory logger factory framework read config and init log
type Factory struct {
}

// Type log plugin type
func (f *Factory) Type() string {
	return pluginType
}

// Setup setup for log
func (f *Factory) Setup(name string, configDec plugin.Decoder) error {
	if configDec == nil {
		return errors.New("log config decoder empty")
	}

	conf, callerSkip, err := f.setupConfig(configDec)
	if err != nil {
		return err
	}

	hasOpentelemetry := false

	for _, c := range conf {
		if c.Writer == writerType {
			hasOpentelemetry = true
			break
		}
	}

	if !hasOpentelemetry {
		conf = append(conf, log.OutputConfig{
			Writer: writerType,
		})
	}

	logger := log.NewZapLogWithCallerSkip(conf, callerSkip)
	if logger == nil {
		return errors.New("new zap logger fail")
	}

	log.Register(name, logger)

	if name == "default" {
		log.SetLogger(logger)
	}

	return nil
}

func (f *Factory) setupConfig(configDec plugin.Decoder) (log.Config, int, error) {
	conf := log.Config{}

	err := configDec.Decode(&conf)
	if err != nil {
		return nil, 0, err
	}

	if len(conf) == 0 {
		return nil, 0, errors.New("log config output empty")
	}

	callerSkip := 2
	for i := 0; i < len(conf); i++ {
		if conf[i].CallerSkip != 0 {
			callerSkip = conf[i].CallerSkip
		}
	}
	return conf, callerSkip, nil
}
