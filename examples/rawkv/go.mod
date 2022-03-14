module rawkv

go 1.16

require (
	github.com/pingcap/log v0.0.0-20211215031037-e024ba4eb0ee
	github.com/tikv/client-go/v2 v2.0.0
	go.uber.org/zap v1.20.0
)

replace github.com/tikv/client-go/v2 => ../../
