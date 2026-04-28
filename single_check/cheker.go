package single_check

import (
	"context"
	"fmt"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/tg"
	"github.com/imbecility/mtproxy-checker/types"
	"github.com/imbecility/mtproxy-checker/utils"
)

// CheckProxy проверяет один MTProto-прокси через gotd, выполняя реальный MTProto-хендшейк
// и запрос HelpGetNearestDC для подтверждения работоспособности.
func CheckProxy(rawURL string, localAddr string, timeout time.Duration) (types.Result, error) {
	server, port, secret, err := utils.ParseProxyURL(rawURL)
	if err != nil {
		return types.Result{}, fmt.Errorf("parse error: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", server, port)

	resolver, err := dcs.MTProxy(addr, secret, dcs.MTProxyOptions{Dial: utils.BuildDialer(localAddr)})
	if err != nil {
		return types.Result{}, fmt.Errorf("invalid proxy config: %w", err)
	}

	client := telegram.NewClient(telegram.TestAppID, telegram.TestAppHash, telegram.Options{
		Resolver:        resolver,
		SessionStorage:  &session.StorageMemory{},
		NoUpdates:       true,
		DialTimeout:     timeout / 2,     // 50% на TCP
		ExchangeTimeout: timeout * 3 / 4, // 75% на хендшейк
	})

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	var latency time.Duration
	var runErr error

	// проверка в дочерней горутине
	done := make(chan struct{})
	go func() {
		defer close(done)
		runErr = client.Run(ctx, func(runCtx context.Context) error {
			_, err := tg.NewClient(client).HelpGetNearestDC(runCtx)
			latency = time.Since(start)
			return err
		})
	}()

	// ожидание реального завершения, либо по таймеру
	select {
	case <-done:
		// результат до истечения `timeout`
		if runErr != nil {
			return types.Result{}, types.ErrProxyDead
		}
		return types.Result{Ping: latency}, nil
	case <-ctx.Done():
		// таймаут, но gotd почему-то завис, придется просто бросить горутину на произвол OS
		return types.Result{}, types.ErrProxyDead
	}
}
