/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2.
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn> 
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package resourcemanage

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"gitee.com/openeuler/PilotGo/sdk/logger"
	"github.com/pkg/errors"
)

const (
	green string = "\x1b[97;104m"
	reset string = "\x1b[0m"
)

type ResourceReleaseFunction func()

type FinalError struct {
	Err error

	Module string

	Severity string

	Cancel context.CancelFunc

	PrintStack bool

	ExitAfterPrint bool
}

func (e *FinalError) Error() string {
	return fmt.Sprintf("%+v", e.Err)
}

type ErrorReleaseManagement struct {
	ErrChan chan error

	errEndChan chan struct{}

	// cancelCtx: 控制 ERManager 本身资源释放的上下文，子上下文：errortransmit
	cancelCtx  context.Context
	cancelFunc context.CancelFunc

	// GoCancelCtx: ERManager 用于控制项目指定goroutine优雅退出的上下文
	GoCancelCtx  context.Context
	GoCancelFunc context.CancelFunc

	Wg sync.WaitGroup

	// releaseFunc: 项目整体资源释放回调函数
	releaseFunc ResourceReleaseFunction
}

func CreateErrorReleaseManager(_ctx context.Context, _releaseFunc ResourceReleaseFunction) (*ErrorReleaseManagement, error) {
	if _ctx == nil || _releaseFunc == nil {
		return nil, fmt.Errorf("context or closeFunc is nil")
	}

	ErrorM := &ErrorReleaseManagement{
		ErrChan:     make(chan error, 20),
		errEndChan:  make(chan struct{}),
		releaseFunc: _releaseFunc,
	}
	ErrorM.cancelCtx, ErrorM.cancelFunc = context.WithCancel(_ctx)
	ErrorM.GoCancelCtx, ErrorM.GoCancelFunc = context.WithCancel(_ctx)

	go ErrorM.errorFactory()

	return ErrorM, nil
}

func (erm *ErrorReleaseManagement) errorFactory() {
	for {
		select {
		case <-erm.errEndChan:
			logger.Info("error management stopped")
			return
		case _error := <-erm.ErrChan:
			if _error == nil {
				continue
			}

			_terror, ok := _error.(*FinalError)
			if !ok {
				logger.Error("plain error: %s", _error.Error())
				continue
			}

			if _terror.Err != nil {
				if !_terror.PrintStack && !_terror.ExitAfterPrint {
					erm.output(_terror)
				} else if _terror.PrintStack && !_terror.ExitAfterPrint {
					logger.ErrorStack(erm.errorStackMsg(_terror.Module), _terror.Err)
				} else if !_terror.PrintStack && _terror.ExitAfterPrint {
					erm.output(_terror)
					_terror.Cancel()
				} else if _terror.PrintStack && _terror.ExitAfterPrint {
					logger.ErrorStack(erm.errorStackMsg(_terror.Module), _terror.Err)
					_terror.Cancel()
				}
			}
		}
	}
}

func (erm *ErrorReleaseManagement) ResourceRelease() {
	erm.releaseFunc()

	erm.GoCancelFunc()

	erm.Wg.Wait()

	time.Sleep(500 * time.Millisecond)

	close(erm.errEndChan)
	close(erm.ErrChan)
}

/*
@severity: debug info warn error

@err:	最终生成的error

@exit_after_print: 打印完异常日志后是否结束主程序

@print_stack: 是否打印异常日志错误链，打印错误链时默认severity为error
*/
func (erm *ErrorReleaseManagement) ErrorTransmit(_module, _severity string, _err error, _exit_after_print, _print_stack bool) {
	if _exit_after_print {
		ctx, cancel := context.WithCancel(erm.cancelCtx)
		erm.ErrChan <- &FinalError{
			Err:            _err,
			Cancel:         cancel,
			Module:         _module,
			Severity:       _severity,
			PrintStack:     _print_stack,
			ExitAfterPrint: _exit_after_print,
		}
		<-ctx.Done()
		erm.ResourceRelease()
		os.Exit(1)
	}

	erm.ErrChan <- &FinalError{
		Err:            _err,
		Cancel:         nil,
		Module:         _module,
		Severity:       _severity,
		PrintStack:     _print_stack,
		ExitAfterPrint: _exit_after_print,
	}
}

func (erm *ErrorReleaseManagement) logFormat(_err error, _module string) string {
	if len(_module) > 10 {
		_module = _module[:10]
	}
	log := fmt.Sprintf("%v %s %-10s %s %+v",
		time.Now().Format("2006-01-02 15:04:05"),
		green, _module, reset,
		_err.Error(),
	)
	return log
}

func (erm *ErrorReleaseManagement) errorStackMsg(_module string) string {
	if len(_module) > 10 {
		_module = _module[:10]
	}
	return fmt.Sprintf("%v %s %-10s %s",
		time.Now().Format("2006-01-02 15:04:05"),
		green, _module, reset,
	)
}

func (erm *ErrorReleaseManagement) output(_err *FinalError) {
	switch _err.Severity {
	case "debug":
		logger.Debug("%s", erm.logFormat(errors.Cause(_err.Err), _err.Module))
	case "info":
		logger.Info("%s", erm.logFormat(errors.Cause(_err.Err), _err.Module))
	case "warn":
		logger.Warn("%s", erm.logFormat(errors.Cause(_err.Err), _err.Module))
	case "error":
		logger.Error("%s", erm.logFormat(errors.Cause(_err.Err), _err.Module))
	default:
		logger.Error("%s", erm.logFormat(errors.Cause(_err.Err), _err.Module))
	}
}
