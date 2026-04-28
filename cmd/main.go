package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/imbecility/mtproxy-checker/cmd/commands"
)

const usageTemplate = `%s
проверяет MTProto-прокси

использование:
  %[2]s <команда> [флаги]

команды:
  check    проверить один прокси-URL
  filter   проверить файл со списком прокси

для справки по конкретной команде: %[2]s <команда> -help
%s`

func usageString() string {
	cliName := filepath.Base(os.Args[0])
	divider := strings.Repeat("-", 70)
	return fmt.Sprintf(usageTemplate, divider, cliName, divider)
}

func main() {
	log.SetFlags(0)
	usage := usageString()
	if len(os.Args) < 2 {
		log.Fatal(usage)
	}

	switch os.Args[1] {
	case "check":
		commands.RunCheck(os.Args[2:])
	case "filter":
		commands.RunFilter(os.Args[2:])
	case "-h", "-help", "--help", "help":
		fmt.Print(usage)
	default:
		log.Fatal(usage)
	}
}
