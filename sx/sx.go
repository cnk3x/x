package sx

import (
	"cmp"
	"context"
	"errors"
	"time"

	"golang.org/x/sync/errgroup"
)

var (
	ErrWaitTimeout = errors.New("wait timeout")
	ErrStopped     = errors.New("stop")
)

func GroupDo(ctx context.Context, stopWait time.Duration, useEg func(eg *errgroup.Group, etx context.Context)) error {
	eg, etx := errgroup.WithContext(ctx)
	errc := make(chan error, 1)
	go func() {
		defer close(errc)
		useEg(eg, etx)
		if err := skipStopped(eg.Wait()); err != nil {
			errc <- err
		}
	}()

	<-etx.Done() // 先等上下文被取消（某个任务返回 err !=nil，或者全部任务完成）

	timer := time.NewTimer(cmp.Or(max(stopWait, 0), time.Second*5))
	select {
	case err := <-errc: // 尝试拿错误
		timer.Stop()
		return err
	case <-timer.C: // 超时，直接退出，放弃等待。
		return ErrWaitTimeout
	}
}

func WrapStopped(f func() error) func() error {
	return func() error {
		if err := f(); err != nil {
			return err
		}
		return ErrStopped
	}
}

func skipStopped(err error) error {
	if errors.Is(err, ErrStopped) {
		return nil
	}
	return err
}
