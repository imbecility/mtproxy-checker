# mtproxy-checker

Утилита и библиотека для проверки работоспособности MTProto-прокси (Telegram).  
Выполняет реальный MTProto-хендшейк через `gotd` и запрос `HelpGetNearestDC` — без эмуляции, без HTTP-запросов.

## Возможности

- **Одиночная проверка** — пинг одного прокси за секунды
- **Массовая фильтрация** — многопоточная проверка файла со списком прокси:
  - живые (отсортированные по пингу) остаются в исходном файле
  - мёртвые сохраняются рядом в `<имя>_dead<расширение>`
- Поддержка форматов URL: `tg://proxy?...` и `https://t.me/proxy?...`
- Поддержка секретов в hex и base64url
- Привязка исходящих соединений к конкретному IP (обход VPN при проверке)

---


## CLI-утилита

### Скачать готовый бинарник

На странице [релизов](https://github.com/imbecility/mtproxy-checker/releases/latest) можно загрузить архив для нужной платформы:

| скачать                                                                                                                        |
|--------------------------------------------------------------------------------------------------------------------------------|
| [Linux x64](https://github.com/imbecility/mtproxy-checker/releases/latest/download/mtproxy-checker-linux-x64.tar.gz)           |
| [Linux ARM  ](https://github.com/imbecility/mtproxy-checker/releases/latest/download/mtproxy-checker-linux-arm.tar.gz)         |
| [macOS Apple Silicon](https://github.com/imbecility/mtproxy-checker/releases/latest/download/mtproxy-checker-macos-arm.tar.gz) |
| [macOS Intel ](https://github.com/imbecility/mtproxy-checker/releases/latest/download/mtproxy-checker-macos-intel.tar.gz)      |
| [Windows x64 ](https://github.com/imbecility/mtproxy-checker/releases/latest/download/mtproxy-checker-windows-x64.zip)         |
| [Windows ARM](https://github.com/imbecility/mtproxy-checker/releases/latest/download/mtproxy-checker-windows-arm.zip)          |

### Сборка из исходников

<details>
  <summary>👈 развернуть и показать инструкции по сборке</summary>

Установить Go: https://go.dev/dl
Затем:

```bash
git clone --depth 1 https://github.com/imbecility/mtproxy-checker.git
cd mtproxy-checker
go mod tidy
go build -ldflags="-s -w -extldflags '-static'" -trimpath -o mtproxy-checker.exe ./cmd
```

Или через скрипты кросс-компиляции:

### Linux / macOS

```bash
chmod +x build_on_linux.sh
./build_on_linux.sh
```

### Windows (только в PowerShell)

```powershell
.\build_on_windows.ps1
```

Готовые бинарники появятся в папке `build/`, архивы для публикации — в `dist/`.

Поддерживаемые платформы: `linux/x64`, `linux/arm`, `macos/intel`, `macos/apple_silicon`, `windows/x64`, `windows/arm64`.

</details>

---

## Использование

```
mtproxy-checker <команда> [флаги]
```

### `check` — проверить один прокси

```
mtproxy-checker check [флаги] <proxy-url>
```

**Аргументы:**

| Аргумент      | Описание                                               |
|---------------|--------------------------------------------------------|
| `<proxy-url>` | URL вида `tg://proxy?...` или `https://t.me/proxy?...` |

**Флаги:**

| Флаг                | По умолчанию | Описание                                              |
|---------------------|--------------|-------------------------------------------------------|
| `-timeout duration` | `10s`        | Таймаут проверки                                      |
| `-bind string`      | —            | Локальный IP для исходящих соединений (необязательно) |

**Примеры:**

```bash
# Базовая проверка
mtproxy-checker check "tg://proxy?server=1.2.3.4&port=443&secret=abc123"

# С явным таймаутом
mtproxy-checker check -timeout 15s "https://t.me/proxy?server=1.2.3.4&port=443&secret=abc123"

# Проверка через конкретный сетевой интерфейс (например, минуя VPN)
mtproxy-checker check -bind 192.168.1.1 "tg://proxy?server=1.2.3.4&port=443&secret=abc123"
```

**Вывод:**
```
[check] ✓ живой: пинг 142ms
```
```
[check] ✗ мёртвый: proxy is dead
```

Код завершения `2` если прокси недоступен, `0` если живой.

---

### `filter` — фильтровать файл со списком прокси

```
mtproxy-checker filter [флаги] <file>
```

**Аргументы:**

| Аргумент | Описание                                                 |
|----------|----------------------------------------------------------|
| `<file>` | Путь к файлу со списком прокси (по одному URL на строку) |

**Флаги:**

| Флаг                | По умолчанию | Описание                                              |
|---------------------|--------------|-------------------------------------------------------|
| `-workers int`      | `50`         | Количество параллельных проверок                      |
| `-timeout duration` | `10s`        | Таймаут на один прокси                                |
| `-bind string`      | —            | Локальный IP для исходящих соединений (необязательно) |

**Примеры:**

```bash
# Базовая фильтрация
mtproxy-checker filter proxies.txt

# Агрессивная многопоточность с коротким таймаутом
mtproxy-checker filter -workers 200 -timeout 5s proxies.txt

# Проверка через конкретный интерфейс
mtproxy-checker filter -bind 192.168.1.1 proxies.txt
```

**Формат входного файла** — по одному URL на строку, строки без прокси-ссылок игнорируются:

```
# этот комментарий будет проигнорирован
tg://proxy?server=1.2.3.4&port=443&secret=abc123
https://t.me/proxy?server=5.6.7.8&port=8888&secret=def456
любой другой текст тоже игнорируется
```

**Результат:**

```
proxies.txt       ← живые прокси, отсортированные по пингу
proxies_dead.txt  ← мёртвые прокси
```

Запись выполняется атомарно через временный файл — исходный файл не повреждается при сбое.

**Вывод:**
```
готово: живых 312, мёртвых 1688
  живые   → /home/user/proxies.txt
  мёртвые → /home/user/proxies_dead.txt
```

---

## Как работает проверка

1. Парсится URL прокси, извлекаются `server`, `port`, `secret`
2. Через `gotd` устанавливается TCP-соединение с прокси-сервером
3. Выполняется полный MTProto-хендшейк (включая обфускацию)
4. Отправляется запрос `HelpGetNearestDC` — первый реальный запрос к Telegram
5. Измеряется время от старта до получения ответа (пинг)

Прокси считается живым только если все четыре шага завершились успешно в рамках таймаута.

---

## Флаг `-bind`: зачем нужен

Если машина подключена к VPN, исходящие соединения идут через него — прокси будут проверяться «через VPN», что искажает пинг и может давать ложные результаты.

Флаг `-bind` позволяет указать IP физического интерфейса (например, домашнего роутера), чтобы соединения шли напрямую в интернет, минуя VPN.

```bash
# Узнать локальный IP интерфейса (Linux/macOS)
ip addr show eth0
ifconfig en0

# Использовать его при проверке
mtproxy-checker filter -bind 192.168.1.100 proxies.txt
```

---

## Использование как библиотеки

Пакеты `single_check` и `proxylist_filter` можно использовать независимо от CLI.

```bash
go get github.com/imbecility/mtproxy-checker
```

---

### `single_check` — проверка одного прокси

```go
import (
    "fmt"
    "time"

    "github.com/imbecility/mtproxy-checker/single_check"
)

result, err := single_check.CheckProxy(
    "tg://proxy?server=1.2.3.4&port=443&secret=abc123",
    "",           // localAddr: "" — системный диалер, иначе IP конкретного интерфейса
    10*time.Second,
)
if err != nil {
    // types.ErrProxyDead — прокси недоступен или не прошёл хендшейк
    fmt.Println("мёртвый:", err)
    return
}
fmt.Println("живой, пинг:", result.Ping.Round(time.Millisecond))
```

**Сигнатура:**

```go
func CheckProxy(rawURL string, localAddr string, timeout time.Duration) (types.Result, error)
```

| Параметр    | Тип             | Описание                                                       |
|-------------|-----------------|----------------------------------------------------------------|
| `rawURL`    | `string`        | URL прокси: `tg://proxy?...` или `https://t.me/proxy?...`      |
| `localAddr` | `string`        | IP исходящего интерфейса; `""` — системный диалер по умолчанию |
| `timeout`   | `time.Duration` | Общий таймаут на всю проверку включая хендшейк                 |

Возвращает `types.ErrProxyDead` если прокси недоступен, не прошёл хендшейк или истёк таймаут.

---

### `proxylist_filter` — массовая проверка

```go
import (
    "fmt"
    "time"

    "github.com/imbecility/mtproxy-checker/proxylist_filter"
)

proxies := []string{
    "tg://proxy?server=1.2.3.4&port=443&secret=abc123",
    "https://t.me/proxy?server=5.6.7.8&port=8888&secret=def456",
    // ...
}

results := proxylist_filter.CheckAndFilterProxies(
    proxies,
    50,            // maxWorkers: количество параллельных проверок
    "",            // localAddr: "" — системный диалер
    10*time.Second,
)

fmt.Println("живые:", results.Alive) // отсортированы по пингу
fmt.Println("мёртвые:", results.Dead)
```

**Сигнатура:**

```go
func CheckAndFilterProxies(
    proxies    []string,
    maxWorkers int,
    localAddr  string,
    timeout    time.Duration,
) types.CheckResults
```

| Параметр     | Тип             | Описание                                                       |
|--------------|-----------------|----------------------------------------------------------------|
| `proxies`    | `[]string`      | Список URL прокси                                              |
| `maxWorkers` | `int`           | Максимум параллельных проверок                                 |
| `localAddr`  | `string`        | IP исходящего интерфейса; `""` — системный диалер по умолчанию |
| `timeout`    | `time.Duration` | Таймаут на один прокси                                         |

**Возвращаемый тип `types.CheckResults`:**

```go
type CheckResults struct {
    Alive []string // живые прокси, отсортированные по пингу
    Dead  []string // недоступные прокси
}
```

---

### `utils` — парсинг URL прокси

Если нужно только разобрать URL без проверки:

```go
import "github.com/imbecility/mtproxy-checker/utils"

server, port, secret, err := utils.ParseProxyURL(
    "tg://proxy?server=1.2.3.4&port=443&secret=abc123",
)
// server: "1.2.3.4"
// port:   443
// secret: []byte{...}  (hex или base64url декодируется автоматически)
```

---

## Лицензия

MIT