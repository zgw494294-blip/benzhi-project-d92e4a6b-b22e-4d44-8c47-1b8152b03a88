package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultAddress = "127.0.0.1:19081"

type config struct {
	Address          string
	DataDirectory    string
	SelfCheck        bool
	SelfCheckTimeout time.Duration
}

func parseConfig(arguments []string) (config, error) {
	set := flag.NewFlagSet("server", flag.ContinueOnError)
	address := set.String("addr", defaultAddress, "HTTP 监听地址")
	dataDirectory := set.String("data", "data", "账本数据目录")
	selfCheck := set.Bool("selfcheck", false, "运行真实 HTTP 全流程自检并退出")
	selfCheckTimeout := set.Duration("selfcheck-timeout", 20*time.Second, "自检超时")
	if err := set.Parse(arguments); err != nil {
		return config{}, err
	}
	if set.NArg() != 0 {
		return config{}, errors.New("不接受位置参数")
	}
	explicitAddress := false
	set.Visit(func(value *flag.Flag) {
		if value.Name == "addr" {
			explicitAddress = true
		}
	})
	resolvedAddress := *address
	if !explicitAddress {
		if rawPort, ok := os.LookupEnv("PORT"); ok && strings.TrimSpace(rawPort) != "" {
			port, err := strconv.Atoi(rawPort)
			if err != nil || port < 1 || port > 65535 {
				return config{}, errors.New("PORT 必须是 1 到 65535 的端口号")
			}
			resolvedAddress = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		}
	}
	if err := validateAddress(resolvedAddress); err != nil {
		return config{}, err
	}
	if strings.TrimSpace(*dataDirectory) == "" {
		return config{}, errors.New("数据目录不能为空")
	}
	if *selfCheckTimeout < 3*time.Second || *selfCheckTimeout > 2*time.Minute {
		return config{}, errors.New("selfcheck-timeout 必须在 3 秒到 2 分钟之间")
	}
	return config{Address: resolvedAddress, DataDirectory: *dataDirectory, SelfCheck: *selfCheck, SelfCheckTimeout: *selfCheckTimeout}, nil
}

func validateAddress(address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("addr 必须采用 IP:port 格式: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("addr 必须显式绑定回环 IP，不能使用 0.0.0.0 或外部地址")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("addr 端口必须在 1 到 65535 之间")
	}
	if port < 1024 || port == 3000 || port == 8080 {
		return errors.New("addr 不能使用低位或常见开发端口")
	}
	return nil
}
