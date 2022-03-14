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
	"encoding/hex"
	"flag"
	"fmt"
	"strings"

	"github.com/pingcap/log"
	"github.com/tikv/client-go/v2/config"
	"github.com/tikv/client-go/v2/rawkv"
)

var (
	pd        string
	cmdCheck  bool
	cmdDelete bool
)

func init() {
	flag.BoolVar(&cmdCheck, "check", true, "check empty key")
	flag.BoolVar(&cmdDelete, "delete", false, "delete empty key entry")
	flag.StringVar(&pd, "pd", "http://127.0.0.1:2379", "pd addr, seperated by comma, default: http://127.0.0.1:2379")
	flag.Parse()

	conf := &log.Config{Level: "error"}
	logger, props, _ := log.InitLogger(conf)
	log.ReplaceGlobals(logger, props)
}

func doCheck() {
	ctx, cancel := context.WithCancel(context.Background())

	cli, err := rawkv.NewClient(ctx, strings.Split(pd, ","), config.DefaultConfig().Security)
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	key := make([]byte, 0)
	val, err := cli.Get(ctx, key)
	if err != nil {
		panic(err)
	}

	if val == nil {
		fmt.Printf("No empty entry\n")
	} else {
		fmt.Printf("Found empty entry, value: 0x%s\n", hex.EncodeToString(val))
	}

	cancel()
}

func doDelete() {
	ctx, cancel := context.WithCancel(context.Background())

	cli, err := rawkv.NewClient(ctx, strings.Split(pd, ","), config.DefaultConfig().Security)
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	key := make([]byte, 0)
	err = cli.Delete(ctx, key)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Emptry entry deleted\n")

	cancel()
}

func main() {
	if cmdDelete {
		doDelete()
	} else if cmdCheck {
		doCheck()
	}
}
