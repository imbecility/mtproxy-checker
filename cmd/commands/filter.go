package commands

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/imbecility/mtproxy-checker/proxylist_filter"
)

const filterUsageTemplate = `%s%s
проверяет список MTProto-прокси из файла.

живые прокси (отсортированные по пингу) перезаписывают исходный файл,
мёртвые сохраняются рядом в файле "<имя>_dead<расширение>".
строки, не являющиеся прокси-URI, игнорируются.

использование:
  %[3]s filter [флаги] <file>

аргументы:
  <file>   путь к файлу со списком прокси (по одному URL на строку)

флаги:
%s
%s`

// RunFilter — точка входа для подкоманды "filter".
func RunFilter(args []string) {
	fs := flag.NewFlagSet("filter", flag.ExitOnError)
	var (
		workers   = fs.Int("workers", 50, "количество параллельных проверок")
		localAddr = fs.String("bind", "", "локальный IP-адрес роутера для исходящих соединений,\n"+
			"позволяет проводить проверку минуя VPN (необязательно)")
		timeout = fs.Duration("timeout", 10*time.Second, "таймаут на один прокси")
	)

	fs.Usage = func() {
		fmt.Print(usageString(fs, filterUsageTemplate, ""))
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() != 1 {
		log.Fatal(usageString(fs, filterUsageTemplate, "укажите ровно один файл"))
	}

	inputFile, err := ResolvePath(fs.Arg(0))
	if err != nil {
		log.Fatal(usageString(fs, filterUsageTemplate, fmt.Sprintf("ошибка разрешения пути %q: %v", fs.Arg(0), err)))
	}
	if err := CheckFile(inputFile); err != nil {
		log.Fatal(usageString(fs, filterUsageTemplate, err.Error()))
	}

	proxies, err := readLines(inputFile)
	if err != nil {
		log.Fatalf("ошибка чтения файла %q: %v\n", inputFile, err)
	}
	if len(proxies) == 0 {
		log.Fatal(usageString(fs, filterUsageTemplate, "файл не содержит прокси"))
	}

	log.Printf("[filter] файл: %s", inputFile)
	log.Printf("[filter] прокси в файле: %d", len(proxies))
	log.Printf("[filter] потоков: %d, таймаут: %s", *workers, *timeout)

	results := proxylist_filter.CheckAndFilterProxies(proxies, *workers, *localAddr, *timeout)

	if err := writeLines(inputFile, results.Alive); err != nil {
		log.Fatalf("ошибка записи живых прокси в %q: %v\n", inputFile, err)
	}
	log.Printf("[filter] живые прокси записаны в: %s (%d шт.)", inputFile, len(results.Alive))

	deadFile := WithStemSuffix(inputFile, "_dead")
	if err := writeLines(deadFile, results.Dead); err != nil {
		log.Fatalf("ошибка записи мёртвых прокси в %q: %v\n", deadFile, err)
	}
	log.Printf("[filter] мёртвые прокси записаны в: %s (%d шт.)", deadFile, len(results.Dead))

	fmt.Printf("\nготово: живых %d, мёртвых %d\n", len(results.Alive), len(results.Dead))
	fmt.Printf("  живые   → %s\n", inputFile)
	fmt.Printf("  мёртвые → %s\n", deadFile)
}
