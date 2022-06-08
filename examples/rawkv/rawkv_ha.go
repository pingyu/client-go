// Copyright 2022 TiKV Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/log"
	"github.com/tikv/client-go/v2/config"
	"github.com/tikv/client-go/v2/rawkv"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var (
	flagPDAddrs       []string
	flagPDAddrStr     = flag.String("pd", "127.0.0.1:2379", "pd address, separated by comma")
	flagThreads       = flag.Int64("threads", 1000, "threads")
	flagKeysPerThread = flag.Int64("keys", 1000000, "keys per threads")
	flagDuration      = flag.Duration("duration", 1*time.Hour, "duration of tests")
	flagLogLevel      = flag.String("log_level", "info", "log level")
	flagRetryCnt      = flag.Int("retry", 10, "max retry count")
)

func Init() {
	flag.Parse()

	flagPDAddrs = strings.Split(*flagPDAddrStr, ",")

	fileConf := log.FileLogConfig{Filename: "rawkv_ha.log", MaxSize: 300}
	conf := &log.Config{Level: *flagLogLevel, File: fileConf}
	logger, props, _ := log.InitLogger(conf)
	log.ReplaceGlobals(logger, props)
}

func workload(ctx context.Context, cli *rawkv.Client, idx int64) (err error) {
	ctxCli := context.Background()
	begin := idx * *flagKeysPerThread
	end := begin + *flagKeysPerThread

	withRetry := func(f func() error) (err error) {
	Loop:
		for i := 0; i < *flagRetryCnt; i++ {
			if e := f(); e == nil {
				return nil
			} else {
				err = multierr.Append(err, errors.Annotatef(e, "retry %d", i))
			}

			select {
			case <-ctx.Done():
				break Loop
			default:
			}
		}
		log.Warn("witryRetry final error", zap.Error(err))
		return
	}

	for {
		for k := begin; k < end; k++ {
			key := []byte(fmt.Sprintf("k%016d", k))
			value0 := []byte(fmt.Sprintf("%v_0", k))

			if err = withRetry(
				func() error { return cli.Put(ctxCli, key, value0) },
			); err != nil {
				return errors.Trace(err)
			}

			select {
			case <-ctx.Done():
				return nil
			default:
			}
		}
	}
}

func runWorkload(ctx context.Context, cli *rawkv.Client) error {
	eg, ctx1 := errgroup.WithContext(ctx)
	ctx, cancel := context.WithTimeout(ctx1, *flagDuration)
	defer cancel()

	for idx := int64(0); idx < *flagThreads; idx++ {
		idx := idx
		eg.Go(func() error {
			return workload(ctx, cli, idx)
		})
	}

	return eg.Wait()
}

func main() {
	Init()

	cli, err := rawkv.NewClient(context.Background(), flagPDAddrs, config.DefaultConfig().Security)
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	fmt.Printf("cluster ID: %d\n", cli.ClusterID())
	log.Info("start runWorkload", zap.Uint64("clusterID", cli.ClusterID()), zap.Any("pd", flagPDAddrs),
		zap.Int64("threads", *flagThreads), zap.Int64("keys", *flagKeysPerThread), zap.Duration("duration", *flagDuration))

	if err := runWorkload(context.Background(), cli); err != nil {
		log.Error("runWorkload error", zap.Error(err))
	} else {
		log.Info("runWorkload finished")
	}
}
