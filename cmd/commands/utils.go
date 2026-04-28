package commands

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ResolvePath преобразует путь в абсолютный вид с использованием разделителей в стиле POSIX,
// раскрывая домашнюю директорию пользователя и разрешая символические ссылки.
func ResolvePath(path string) (string, error) {
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	}

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved, err = filepath.Abs(path)
		if err != nil {
			return "", err
		}
	}

	return filepath.ToSlash(resolved), nil
}

// WithStemSuffix добавляет указанный суффикс к базовому имени файла перед его расширением,
// сохраняя исходный путь к директории и расширение файла.
func WithStemSuffix(path, suffix string) string {
	resolved, err := ResolvePath(path)
	if err != nil {
		log.Printf("  внимание, не удалось разрешить путь %q: %v", path, err)
		resolved = path
	}
	dir := filepath.Dir(resolved)
	ext := filepath.Ext(resolved)
	base := filepath.Base(resolved)
	stem := base[:len(base)-len(ext)]

	newName := stem + suffix + ext
	return filepath.Join(dir, newName)
}

// IsTextLine проверяет, состоит ли строка из печатных символов и стандартных управляющих кодов:
// возвращает false, если в строке обнаружены непечатные управляющие символы.
func IsTextLine(s string) bool {
	for _, b := range []byte(s) {
		if b < 9 || (b > 13 && b < 32) {
			return false
		}
	}
	return true
}

// CheckFile проверяет файл по указанному пути на соответствие формату списка Telegram-прокси:
// файл должен быть текстовым и содержать хотя бы одну прокси-ссылку.
func CheckFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Printf("не удалось корректно закрыть файл '%s': %v", path, cerr)
		}
	}()

	scanner := bufio.NewScanner(f)
	hasProxy := false

	for scanner.Scan() {
		line := scanner.Text()

		if !IsTextLine(line) {
			return fmt.Errorf("передан не текстовый файл '%s'", path)
		}

		if isProxyLine(line) {
			hasProxy = true
			// Дальнейшее сканирование не нужно — файл валиден.
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения файла '%s': %w", path, err)
	}

	if !hasProxy {
		return fmt.Errorf("передан текстовый файл '%s' который не содержит ссылок tg-прокси", path)
	}

	return nil
}

// readLines читает файл и возвращает строки являющиеся прокси-ссылками.
func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Printf("ошибка закрытия файла %s: %v", path, cerr)
		}
	}()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !isProxyLine(line) {
			continue
		}
		lines = append(lines, line)
	}
	return lines, scanner.Err()
}

func isProxyLine(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "tg://proxy") ||
		strings.HasPrefix(line, "https://t.me/proxy")
}

// writeLines атомарно записывает строки в файл (через временный файл).
func writeLines(path string, lines []string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".mtproxy-tmp-*")
	if err != nil {
		return fmt.Errorf("создание временного файла: %w", err)
	}
	tmpName := tmp.Name()

	// При любой ошибке до Rename удаляем временный файл.
	success := false
	defer func() {
		if !success {
			if rerr := os.Remove(tmpName); rerr != nil {
				log.Printf("ошибка удаления временного файла %s: %v", tmpName, rerr)
			}
		}
	}()

	w := bufio.NewWriter(tmp)
	for _, line := range lines {
		if _, werr := fmt.Fprintln(w, line); werr != nil {
			if cerr := tmp.Close(); cerr != nil {
				log.Printf("ошибка закрытия временного файла %s: %v", tmpName, cerr)
			}
			return fmt.Errorf("запись строки: %w", werr)
		}
	}

	if err := w.Flush(); err != nil {
		if cerr := tmp.Close(); cerr != nil {
			log.Printf("ошибка закрытия временного файла %s: %v", tmpName, cerr)
		}
		return fmt.Errorf("flush: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("закрытие временного файла: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("переименование %q → %q: %w", tmpName, path, err)
	}

	success = true
	return nil
}

func usageString(fs *flag.FlagSet, template string, msg string) string {
	divider := strings.Repeat("-", 70)
	if msg != "" {
		msg = fmt.Sprintf("\n%s! ОШИБКА !\n%s\n%s\n",
			strings.Repeat(" ", 29), msg, divider)
	}
	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.PrintDefaults()
	cliName := filepath.Base(os.Args[0])
	return fmt.Sprintf(template, divider, msg, cliName, buf.String(), divider)
}
