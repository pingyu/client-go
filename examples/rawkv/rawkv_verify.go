// Copyright 2021 TiKV Authors
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
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/pingcap/log"
	"github.com/tikv/client-go/v2/config"
	"github.com/tikv/client-go/v2/rawkv"
	"go.uber.org/zap"
)

var (
	cmd        string
	pdMain     string
	pdRecovery string
	threads    int
	keyCount   int
	valueBase  int
	sleep      int
)

func init() {
	flag.StringVar(&cmd, "cmd", "BOTH", "PUT/VERIFY/BOTH")
	flag.StringVar(&pdMain, "main", "http://127.0.0.1:2379", "main cluster pd addr, default: http://127.0.0.1:2379")
	flag.StringVar(&pdRecovery, "recovery", "http://127.0.0.1:2379", "secondary cluster pd addr, default: http://127.0.0.1:2379")
	flag.IntVar(&threads, "threads", 10, "# of threads")
	flag.IntVar(&keyCount, "keys", 10, "# of keys")
	flag.IntVar(&valueBase, "value", 0, "value to put / verify")
	flag.IntVar(&sleep, "sleep", 1, "sleep seconds between PUT & Verify")
	flag.Parse()

	conf := &log.Config{Level: "debug", File: log.FileLogConfig{
		Filename: "rawkv_verify.log",
	}}
	logger, props, _ := log.InitLogger(conf)
	log.ReplaceGlobals(logger, props)

	if valueBase == 0 {
		s := rand.NewSource(time.Now().UnixNano())
		r := rand.New(s)
		valueBase = r.Intn(1000) + 1
	}
}

func doPut() {
	if keyCount%threads != 0 {
		threads += 1
	}
	keyCountPerThread := keyCount / threads

	var wg sync.WaitGroup
	for i := 0; i < threads; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithCancel(context.Background())

			cli, err := rawkv.NewClient(ctx, []string{pdMain}, config.DefaultConfig().Security)
			if err != nil {
				panic(err)
			}
			defer cli.Close()

			log.Debug("PUT worker start", zap.Uint64("clusterID", cli.ClusterID()), zap.Int("thread", i))

			end := (i + 1) * keyCountPerThread
			if end > keyCount {
				end = keyCount
			}
			for k := i * keyCountPerThread; k < end; k++ {
				key := fmt.Sprintf("vk%v", k)
				val_1 := fmt.Sprintf("%v", valueBase-1)
				val := fmt.Sprintf("%v", valueBase)
				log.Debug("PUT", zap.String("key", key), zap.String("val-1", val_1), zap.String("val", val))

				err = cli.Put(ctx, []byte(key), []byte(val_1))
				if err != nil {
					panic(err)
				}
				err = cli.Put(ctx, []byte(key), []byte(val))
				if err != nil {
					panic(err)
				}
			}

			cancel()
		}()
	}
	wg.Wait()
}

func doVerify() {
	if keyCount%threads != 0 {
		threads += 1
	}
	keyCountPerThread := keyCount / threads

	var wg sync.WaitGroup
	for i := 0; i < threads; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithCancel(context.Background())

			cli, err := rawkv.NewClient(ctx, []string{pdRecovery}, config.DefaultConfig().Security)
			if err != nil {
				panic(err)
			}
			defer cli.Close()

			log.Debug("VERIFY worker start", zap.Uint64("clusterID", cli.ClusterID()), zap.Int("thread", i))

			end := (i + 1) * keyCountPerThread
			if end > keyCount {
				end = keyCount
			}
			for k := i * keyCountPerThread; k < end; k++ {
				key := fmt.Sprintf("vk%v", k)
				expected := fmt.Sprintf("%v", valueBase)

				val, err := cli.Get(ctx, []byte(key))
				if err != nil {
					panic(err)
				}
				log.Debug("VERIFY", zap.String("key", key), zap.String("val-expected", expected), zap.String("got", string(val)))

				if string(val) != expected {
					log.Error("VERIFY ERROR", zap.String("key", key), zap.String("val-expected", expected), zap.String("got", string(val)))
					fmt.Fprintf(os.Stderr, "VERIFY ERROR: key: %v, value-expect: %v, got: %v\n", key, expected, string(val))
				}
			}

			cancel()
		}()
	}
	wg.Wait()
}

func main() {
	fmt.Printf("Value Base: %v\n", valueBase)

	if cmd == "PUT" {
		doPut()
	} else if cmd == "VERIFY" {
		doVerify()
	} else {
		fmt.Printf("Do PUT now.\n")
		doPut()

		fmt.Printf("PUT finished. Waiting for %v seconds... ", sleep)
		time.Sleep(time.Duration(sleep) * time.Second)
		fmt.Printf("Do VERIFY now.\n")

		doVerify()
		fmt.Printf("DONE.\n")
	}
}
