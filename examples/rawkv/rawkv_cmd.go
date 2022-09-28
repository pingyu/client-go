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
	"fmt"
	"os"
	"strconv"

	"github.com/pingcap/log"
	"github.com/tikv/client-go/v2/config"
	"github.com/tikv/client-go/v2/rawkv"
)

func main() {
	conf := &log.Config{Level: "debug", File: log.FileLogConfig{}}
	logger, props, _ := log.InitLogger(conf)
	log.ReplaceGlobals(logger, props)

	if len(os.Args) < 5 {
		fmt.Printf("%s <pd-url> <get/put/put-ttl/del> <key> <value>\n", os.Args[0])
		return
	}

	pd := os.Args[1]
	oper := os.Args[2]
	key := []byte(os.Args[3])
	value := []byte(os.Args[4])

	// cli, err := rawkv.NewClient(context.TODO(), []string{pd}, config.DefaultConfig().Security)
	cli, err := rawkv.NewClientV2(context.TODO(), []string{pd}, config.DefaultConfig().Security)
	// cli, err := rawkv.NewClientV2WithPrefix(context.TODO(), []string{pd}, config.DefaultConfig().Security)
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	fmt.Printf("cluster ID: %d\n", cli.ClusterID())

	if oper == "put" {
		// put key into tikv
		err = cli.Put(context.TODO(), key, value)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Successfully put %s:%s to tikv\n", key, value)
	} else if oper == "put-ttl" {
		ttl, err := strconv.Atoi(os.Args[5])
		if err != nil {
			panic(err)
		}
		err = cli.PutWithTTL(context.TODO(), key, value, uint64(ttl))
		if err != nil {
			panic(err)
		}
		fmt.Printf("Successfully put-ttl %s:%s (ttl: %v) to tikv\n", key, value, ttl)
	} else if oper == "get" {
		// get key from tikv
		val, err := cli.Get(context.TODO(), key)
		if err != nil {
			panic(err)
		}
		if val == nil {
			fmt.Printf("not found for key: %s\n", key)
		} else {
			fmt.Printf("found val: %s for key: %s\n", string(val), key)
		}
	} else if oper == "del" {
		// delete key from tikv
		err := cli.Delete(context.TODO(), key)
		if err != nil {
			panic(err)
		}
		fmt.Printf("key: %s deleted\n", key)
	} else if oper == "scan" {
		endKey := []byte(os.Args[4])
		limit, err := strconv.Atoi(os.Args[5])
		if err != nil {
			panic(err)
		}
		keys, values, err := cli.Scan(context.TODO(), key, endKey, limit)
		if err != nil {
			panic(err)
		}
		for i := range keys {
			fmt.Printf("(%v, %v)\n", hex.EncodeToString(keys[i]), hex.EncodeToString(values[i]))
		}
		fmt.Println("")
	} else if oper == "del-range" {
		endKey := []byte(os.Args[4])
		err := cli.DeleteRange(context.TODO(), key, endKey)
		if err != nil {
			panic(err)
		}
		fmt.Println("success")
	} else if oper == "cas" {
		newValue := []byte(os.Args[5])
		oldValue := []byte(nil)
		if len(value) != 0 {
			oldValue = value
		}
		var succeed bool
		cli.SetAtomicForCAS(true)
		oldValue, succeed, err = cli.CompareAndSwap(context.TODO(), key, oldValue, newValue)
		if err != nil {
			panic(err)
		}
		fmt.Printf("succeed: %v, oldValue: %v\n", succeed, oldValue)
	}
}
