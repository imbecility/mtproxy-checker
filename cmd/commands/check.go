package commands

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/imbecility/mtproxy-checker/single_check"
)

const usageTemplate = `%s%s
проверяет один MTProto-прокси и выводит пинг или сообщение об ошибке.

использование:
  %[3]s check [флаги] <proxy-url>

аргументы:
  <proxy-url>   URL прокси вида tg://proxy?... или https://t.me/proxy?...

флаги:
%s
%s`

// RunCheck — точка входа для подкоманды "check".
func RunCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	var (
		localAddr = fs.String("bind", "", "локальный IP-адрес роутера для исходящих соединений,\n"+
			"позволяет проводить проверку минуя VPN (необязательно)")
		timeout = fs.Duration("timeout", 10*time.Second, "таймаут проверки")
	)

	fs.Usage = func() {
		fmt.Print(usageString(fs, usageTemplate, ""))
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() != 1 {
		log.Fatal(usageString(fs, usageTemplate, "укажите ровно один proxy-url"))
	}
	proxyURL := fs.Arg(0)
	if !isProxyLine(proxyURL) {
		log.Fatal(usageString(fs, usageTemplate, "proxy-url должны быть вида:\ntg://proxy?server=*&port=*&secret=*\nhttps://t.me/proxy?server=*&port=*&secret=*"))
	}
	log.Printf("[check] проверка...")
	log.Printf("[check] таймаут: %s", *timeout)
	if *localAddr != "" {
		log.Printf("[check] bind: %s", *localAddr)
	}
	result, err := single_check.CheckProxy(proxyURL, *localAddr, *timeout)
	if err != nil {
		log.Printf("[check] ✗ мёртвый: %v", err)
		os.Exit(2)
	}
	fmt.Printf("[check] ✓ живой: пинг %s\n", result.Ping.Round(time.Millisecond))
}
