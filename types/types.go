package types

import (
	"errors"
	"time"
)

// CheckResult содержит результат проверки прокси.
type CheckResult struct {
	URL  string
	Ping time.Duration
}

// Result содержит результат проверки tg-прокси.
type Result struct {
	Ping time.Duration
}

// CheckResults содержит результаты проверки прокси, разделённые на живые и мёртвые.
type CheckResults struct {
	Alive []string
	Dead  []string
}

// ErrProxyDead возвращается когда прокси недоступен или заблокирован.
var ErrProxyDead = errors.New("proxy is dead")
