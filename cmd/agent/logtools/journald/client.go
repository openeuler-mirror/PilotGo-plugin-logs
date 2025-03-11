/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn> 
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package journald

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/global"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/public"
	"github.com/gorilla/websocket"
)

var (
	FollowLogDefaultOptions = []string{"--quiet", "--utc", "--output=json"}

	UnitListDefaultOptions = []string{"list-units", "--no-legend", "--type=service", "--no-pager"}
)

type JournaldClient struct {
	ID string

	Active bool

	wswriteMutex sync.Mutex
	wsreadMutex  sync.Mutex

	wsconn *websocket.Conn

	defaultOptions []string
	options        *public.JournalctlOptions

	CancelC context.Context
	CancelF context.CancelFunc

	Jcmd *exec.Cmd

	wg   sync.WaitGroup
	once sync.Once

	dataCh          chan *public.StdoutData
	errCh           chan []byte
	CloseReadMsgCh  chan struct{}
	closeWriteMsgCh chan struct{}

	// 打印stderr信息时的超时控制
	timeout time.Duration

	UnitsMap map[string][]string

	PageEntryBuff PageEntryBuffSortByTimestamp
}

func CreateJournaldClient(_conn *websocket.Conn, _timeout time.Duration) *JournaldClient {
	cancelCtx, cancelFunc := context.WithCancel(JournaldCtx)
	return &JournaldClient{
		wsconn:          _conn,
		defaultOptions:  FollowLogDefaultOptions,
		options:         nil,
		CancelC:         cancelCtx,
		CancelF:         cancelFunc,
		Jcmd:            nil,
		dataCh:          make(chan *public.StdoutData, 10),
		errCh:           make(chan []byte, 10),
		CloseReadMsgCh:  make(chan struct{}, 10),
		closeWriteMsgCh: make(chan struct{}, 10),
		timeout:         _timeout,
		UnitsMap:        make(map[string][]string),
	}
}

func (jclient *JournaldClient) ReadMessageFromClient() {
OuterLoop:
	for {
		select {
		case <-jclient.CloseReadMsgCh:
			global.ERManager.ErrorTransmit("journald", "warn", errors.New("jclient.ReadMessageFromClient() exit: cancelctx canceled"), false, false)
			return
		default:
			jclient.wsreadMutex.Lock()
			msgType, jmsgBytes, err := jclient.wsconn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					if jclient.Active {
						global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("websocket client %s closed: %s", jclient.wsconn.RemoteAddr().String(), err.Error()), false, false)
						jclient.Close(true, false, false)
					}
					return
				}
				if jclient.Active {
					global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("error while reading message(msgType: %d): %s, %s", msgType, err.Error(), jmsgBytes), false, false)
					jclient.Close(true, false, false)
				}
				return
			}
			jclient.wsreadMutex.Unlock()

			jmsg := &public.JMessage{}
			if err := json.Unmarshal(jmsgBytes, jmsg); jmsgBytes != nil && err != nil {
				global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("error while unmarshalling json options: %s, jmsg: %+v", err.Error(), string(jmsgBytes)), false, true)
				jclient.Close(true, false, false)
				return
			}

			jclient.once.Do(func() {
				global.ERManager.ErrorTransmit("journald", "debug", errors.Errorf("jmsg.type: %+v, jmsg.joptions:%+v, jmsg.data: %+v", jmsg.Type, jmsg.JOptions, jmsg.Data), false, false)
			})

			switch jmsg.Type {
			case public.UpdateOptionsMsg:
				// TODO: 连续发送相同的查询请求暂时跳过
				if jclient.options == jmsg.JOptions {
					time.Sleep(1 * time.Second)
					continue OuterLoop
				}

				if jclient.Jcmd != nil {
					global.ERManager.ErrorTransmit("journald", "info", errors.Errorf("==========%-50s==========", "reset journalctl options"), false, false)
					global.ERManager.ErrorTransmit("journald", "info", errors.Errorf("jmsg.type: %+v, jmsg.joptions:%+v, jmsg.data: %+v", jmsg.Type, jmsg.JOptions, jmsg.Data), false, false)
					// 释放上一次查询的资源
					jclient.Close(false, false, false)
				}

				cancelCtx, cancelFunc := context.WithCancel(JournaldCtx)
				jclient.CancelC = cancelCtx
				jclient.CancelF = cancelFunc
				jclient.options = jmsg.JOptions
				jclient.PageEntryBuff = nil
				jclient.Jcmd = exec.Command("journalctl", jclient.assembleOptions(jclient.defaultOptions, jmsg.JOptions)...)
				go jclient.WriteMessageToClient()
				jclient.ProcessData(jclient.Jcmd, public.LogEntryData)
			case public.UnitListMsg:
				cmd := exec.Command("systemctl", UnitListDefaultOptions...)
				go jclient.WriteMessageToClient()
				jclient.ProcessData(cmd, public.UnitData)
			case public.UpdatePageMsg:
				if len(jclient.PageEntryBuff) == 0 {
					continue OuterLoop
				}
				jclient.options.From = jmsg.JOptions.From
				jclient.options.Size = jmsg.JOptions.Size
				start_index, end_index := jclient.options.From, jclient.options.From+jclient.options.Size
				if len(jclient.PageEntryBuff) <= end_index {
					end_index = len(jclient.PageEntryBuff)
				}
				if len(jclient.PageEntryBuff) <= start_index {
					start_index = len(jclient.PageEntryBuff)
				}

				jclient.dataCh <- &public.StdoutData{
					Type: public.LogEntryData,
					Data: strings.Join(jclient.PageEntryBuff[start_index:end_index], "\n"),
				}
			default:
				global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("unsupport message type: %+v", jmsg), false, false)
			}
		}
	}
}

func (jclient *JournaldClient) ProcessData(_cmd *exec.Cmd, _data_type public.StdoutDataType) {
	cmd_stdout, err := _cmd.StdoutPipe()
	if err != nil {
		global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("cannot get stdout pipe: %s", err), false, false)
		jclient.Close(true, false, false)
		return
	}
	cmd_stderr, err := _cmd.StderrPipe()
	if err != nil {
		global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("cannot get stderr pipe: %s", err), false, false)
		jclient.Close(true, false, false)
		return
	}

	jclient.wg.Add(1)
	go jclient.readFromStderr(cmd_stderr)
	jclient.wg.Add(1)
	go jclient.readFromStdout(_data_type, cmd_stdout)
	global.ERManager.Wg.Add(1)
	go func(__cmd *exec.Cmd) {
		defer global.ERManager.Wg.Done()
		__cmd.WaitDelay = time.Second * 2
		err = __cmd.Run()
		if err != nil {
			global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("err while running cmd: %d, %s", __cmd.ProcessState.ExitCode(), err.Error()), false, false)
			// 主动kill journalctl process，exitcode: -1, 不释放资源
			if __cmd.ProcessState.ExitCode() != -1 {
				jclient.Close(false, false, true)
				return
			}
		}
		// 分页查询模式下command执行完成时不调用cancelFunc
		// if jclient.options == nil || !jclient.options.Notail {
		// 	jclient.Close(false, false, false)
		// }
	}(_cmd)
}

func (jclient *JournaldClient) WriteMessageToClient() {
	for {
		select {
		case <-jclient.closeWriteMsgCh:
			global.ERManager.ErrorTransmit("journald", "warn", errors.New("cancelctx canceled, jclient.WriteMessageToClient() exit"), false, false)
			return
		case data, open := <-jclient.dataCh:
			if !open {
				global.ERManager.ErrorTransmit("journald", "warn", errors.New("journalctl stdout pipe closed"), false, false)
				jclient.Close(true, false, false)
				return
			}

			var err error
			jdata := &public.StdoutData{}
			switch data.Type {
			// 日志条目查询
			case public.LogEntryData:
				jdata.Type = public.LogEntryData
				if jclient.options.Notail {
					// 分页查询
					var raw_page_entries []string
					var ok bool

					buffdata, ok := data.Data.(string)
					if !ok {
						global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("fail to assert page enties json to string: %+v(%T)", data.Data, data.Data), false, true)
						jclient.Close(true, true, false)
						return
					}

					if buffdata != "abnormal" {
						if len(jclient.PageEntryBuff) == 0 {
							jclient.PageEntryBuff = strings.Split(buffdata, "\n")
							jclient.PageEntryBuff = jclient.PageEntryBuff[:len(jclient.PageEntryBuff)-1]
							sort.Sort(jclient.PageEntryBuff)
							start_index, end_index := jclient.options.From, jclient.options.Size
							if len(jclient.PageEntryBuff) <= jclient.options.Size {
								end_index = len(jclient.PageEntryBuff)
							}
							raw_page_entries = jclient.PageEntryBuff[start_index:end_index]
						} else {
							raw_page_entries = strings.Split(buffdata, "\n")
							if !ok {
								global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("fail to assert page enties json to []string: %+v(%T)", data.Data, data.Data), false, true)
								jclient.Close(true, true, false)
								return
							}
						}

						page_entries := make([]map[string]interface{}, 0)
						for _, entry_json := range raw_page_entries {
							if len(entry_json) == 0 {
								continue
							}
							raw_entry := map[string]interface{}{}
							if err := json.Unmarshal([]byte(entry_json), &raw_entry); err != nil {
								global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("fail to unmarshal Journald JSON: %s; raw data: %+v(%d)", err, entry_json, len(entry_json)), false, true)
								jclient.Close(true, true, false)
								return
							}
							page_entries = append(page_entries, jclient.generateEntry(raw_entry))
						}

						jdata.Data = &public.PageData{
							Total: len(jclient.PageEntryBuff),
							Hits:  page_entries,
						}
					} else {
						jdata.Data = nil
					}
				} else {
					// 实时查询
					raw_entry := map[string]interface{}{}
					jsondata, ok := data.Data.(string)
					if !ok {
						global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("fail to assert follow mode json to string: %+v(%T)", data.Data, data.Data), false, true)
						jclient.Close(true, true, false)
						return
					}
					if jsondata != "abnormal" {
						if err := json.Unmarshal([]byte(jsondata), &raw_entry); err != nil {
							global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("fail to unmarshal Journald JSON: %s; raw data: %s(%d)", err, data.Data.(string), len(data.Data.(string))), false, true)
							jclient.Close(true, true, false)
							return
						}
						jdata.Data = jclient.generateEntry(raw_entry)
					} else {
						jdata.Data = nil
					}
				}
			// 服务单元查询
			case public.UnitData:
				jdata.Type = public.UnitData
				if data.Data.(string) != "abnormal" {
					systemd_units_raw := strings.Split(data.Data.(string), "\n")
					if err := jclient.generateUnitsMap(systemd_units_raw); err != nil {
						global.ERManager.ErrorTransmit("journald", "error", errors.Wrap(err, " "), false, true)
					}
					jdata.Data = jclient.UnitsMap
				} else {
					jdata.Data = nil
				}
			}

			jmsg := &public.JMessage{
				Type: public.DataMsg,
				Data: jdata,
			}
			jmsgBytes, err := json.Marshal(jmsg)
			if err != nil {
				global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("fail to marshal message: %s", err.Error()), false, true)
			}

			if jclient.wsconn == nil {
				break
			}

			jclient.wswriteMutex.Lock()
			if err := jclient.wsconn.WriteMessage(websocket.TextMessage, jmsgBytes); err != nil {
				global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("error while writing message to ws client: %s", err.Error()), false, true)
			}
			jclient.wswriteMutex.Unlock()
		}
	}
}

func (jclient *JournaldClient) generateEntry(_raw_entry map[string]interface{}) map[string]interface{} {
	entry := map[string]interface{}{}
	if _raw_entry["__REALTIME_TIMESTAMP"].(string) != "" {
		timestamp_int64, err := strconv.ParseInt(_raw_entry["__REALTIME_TIMESTAMP"].(string), 10, 64)
		if err != nil {
			global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("fail to parse timestamp %s: %s", _raw_entry["__REALTIME_TIMESTAMP"].(string), err.Error()), false, true)
			entry["timestamp"] = _raw_entry["__REALTIME_TIMESTAMP"].(string)[:13]
		}
		entry["timestamp"] = strconv.Itoa(int(timestamp_int64 / 1000))
	}
	if _raw_entry["PRIORITY"] != nil && _raw_entry["PRIORITY"].(string) != "" {
		entry["level"] = _raw_entry["PRIORITY"].(string)
	}
	if _raw_entry["MESSAGE"].(string) != "" {
		entry["message"] = _raw_entry["MESSAGE"].(string)
	}
	if _raw_entry["_TRANSPORT"].(string) != "" {
		switch _raw_entry["_TRANSPORT"].(string) {
		case "journal":
			if _raw_entry["UNIT"] != nil {
				entry["targetname"] = _raw_entry["UNIT"].(string)
			} else if _raw_entry["SYSLOG_IDENTIFIER"] != nil {
				entry["targetname"] = _raw_entry["SYSLOG_IDENTIFIER"].(string)
			}
		// TODO: 暂时无法提供syslog日志
		case "syslog", "kernel", "audit":
			if _raw_entry["SYSLOG_IDENTIFIER"] != nil {
				entry["targetname"] = _raw_entry["SYSLOG_IDENTIFIER"].(string)
			}
		}
	}
	return entry
}

func (jclient *JournaldClient) generateUnitsMap(_systemd_units_raw []string) error {
	user_units := []string{}
	user_, err := user.Lookup("root")
	if err != nil {
		return errors.Errorf("fail to get user: %s", err)
	}
	user_units = append(user_units, fmt.Sprintf("%v:%v", user_.Username, user_.Uid))

	user_, err = user.Current()
	if err != nil {
		return errors.Errorf("fail to get user: %s", err)
	}
	if user_.Username != "root" {
		user_units = append(user_units, fmt.Sprintf("%v:%v", user_.Username, user_.Uid))
	}

	jclient.UnitsMap["user"] = user_units

	jclient.UnitsMap["transport"] = []string{"audit", "kernel"}

	systemd_units := []string{}
	for _, line := range _systemd_units_raw {
		if line != "" {
			switch global.OsName {
			case "openEuler":
				unit_name := ""
				line_split_by_space := strings.Split(strings.TrimLeft(line, " "), " ")
				for _, e := range line_split_by_space {
					if strings.Contains(e, ".service") {
						unit_name = strings.Split(e, ".service")[0]
						break
					}
				}
				systemd_units = append(systemd_units, unit_name)
			case "Kylin":
				unit_name := ""
				line_split_by_space := strings.Split(line, " ")
				for _, e := range line_split_by_space {
					if strings.Contains(e, ".service") {
						unit_name = strings.Split(e, ".service")[0]
						break
					}
				}
				systemd_units = append(systemd_units, unit_name)
			}
		}
	}
	jclient.UnitsMap["systemd"] = systemd_units
	return nil
}

func (jclient *JournaldClient) readFromStdout(_type public.StdoutDataType, _stdout io.ReadCloser) {
	defer jclient.wg.Done()
	defer _stdout.Close()

	reader := bufio.NewReader(_stdout)
	for {
		select {
		case <-jclient.CancelC.Done():
			global.ERManager.ErrorTransmit("journald", "warn", errors.New("jclient.readFromStdout() exit, cancelctx canceled"), false, false)
			return
		default:
			dataT := &public.StdoutData{}
			switch _type {
			case public.LogEntryData:
				dataT.Type = public.LogEntryData
				if jclient.options.Notail {
					text, err := jclient.readAllOnce(_stdout)
					if err != nil {
						if strings.Contains(err.Error(), "EOF") {
							global.ERManager.ErrorTransmit("journald", "error", errors.Wrap(err, "jclient.readFromStdout() exit: "), false, false)
							dataT.Data = "abnormal"
							jclient.dataCh <- dataT
							return
						}
						global.ERManager.ErrorTransmit("journald", "error", errors.Wrap(err, "jclient.readFromStdout() exit: "), false, true)
						dataT.Data = "abnormal"
						jclient.dataCh <- dataT
						return
					}
					dataT.Data = text
				} else {
					text := jclient.readOneLineOnce(reader)
					if text == "" {
						dataT.Data = "abnormal"
						jclient.dataCh <- dataT
						global.ERManager.ErrorTransmit("journald", "debug", errors.New("jclient.readFromStdout() exit: EOF"), false, false)
						return
					}
					dataT.Data = text
				}
			case public.UnitData:
				dataT.Type = public.UnitData
				text, err := jclient.readAllOnce(_stdout)
				if text == "" && err != nil {
					if strings.Contains(err.Error(), "EOF") {
						global.ERManager.ErrorTransmit("journald", "error", errors.Wrap(err, "jclient.readFromStdout() exit: "), false, false)
						dataT.Data = "abnormal"
						jclient.dataCh <- dataT
						return
					}
					global.ERManager.ErrorTransmit("journald", "error", errors.Wrap(err, "jclient.readFromStdout() exit: "), false, true)
					dataT.Data = "abnormal"
					jclient.dataCh <- dataT
					return
				}
				dataT.Data = text
			}
			jclient.dataCh <- dataT
		}
	}
}

func (jclient *JournaldClient) readOneLineOnce(_reader *bufio.Reader) string {
	bytes, err := _reader.ReadBytes('\n')
	if err != nil {
		return ""
	}

	return string(bytes)
}

func (jclient *JournaldClient) readAllOnce(_reader io.ReadCloser) (string, error) {
	bytes, err := io.ReadAll(_reader)
	if len(bytes) == 0 {
		return "", errors.New("cannot read from cmd stdout: EOF")
	}
	if err != nil {
		return string(bytes), errors.Errorf("cannot read from cmd stdout(bytes: %s): %s", string(bytes), err)
	}
	return string(bytes), nil
}

func (jclient *JournaldClient) readFromStderr(_stderr io.ReadCloser) {
	defer jclient.wg.Done()
	defer _stderr.Close()
	reader := bufio.NewReader(_stderr)
	for {
		select {
		case <-jclient.CancelC.Done():
			global.ERManager.ErrorTransmit("journald", "warn", errors.New("jclient.readFromStderr() exit: cancelctx canceled"), false, false)
			return
		default:
			data, err := reader.ReadBytes('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					global.ERManager.ErrorTransmit("journald", "error", errors.New("jclient.readFromStderr() exit: EOF"), false, false)
					return
				}
				global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("jclient.readFromStderr() exit: %s", err), false, false)
				return
			}
			jclient.errCh <- data
		}
	}
}

func (jclient *JournaldClient) assembleOptions(_initOptions []string, _options *public.JournalctlOptions) []string {
	if _options.Notail {
		_initOptions = append(_initOptions, "--no-tail")
	} else {
		_initOptions = append(_initOptions, "--follow")
	}
	if _options.Notail && _options.Since != "" && _options.Until != "" {
		_initOptions = append(_initOptions, "--since", _options.Since, "--until", _options.Until)
	}
	if _options.Unit != "" {
		_initOptions = append(_initOptions, "--unit", _options.Unit)
	}
	if _options.Identifier != "" {
		_initOptions = append(_initOptions, "--identifier", _options.Identifier)
	}
	if _options.Severity != "" {
		_initOptions = append(_initOptions, "--priority", _options.Severity)
	}
	if _options.Transport != "" {
		_initOptions = append(_initOptions, fmt.Sprintf("_TRANSPORT=%s", _options.Transport))
	}
	if _options.User != "" {
		uid := strings.Split(_options.User, ":")[1]
		if uid == "" {
			global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("user field in options is invalid: %s", _options.User), false, false)
		}
		_initOptions = append(_initOptions, "_UID="+uid)
	}
	return _initOptions
}

func (jclient *JournaldClient) ReturnJournalctlOptions() *public.JournalctlOptions {
	return jclient.options
}

/*
_closeconn: 是否关闭websocket连接

_closechan: 是否关闭dataCh和errCh

_printstderr: 因journalctl command执行异常而调用releasesource，且打印stderr错误信息
*/
func (jclient *JournaldClient) Close(_closeconn, _closechan, _printstderr bool) {
	global.ERManager.ErrorTransmit("journald", "info", errors.Errorf("==========%-50s==========", fmt.Sprintf("journald client %s call close", jclient.ID)), false, false)

	if _closeconn && jclient.wsconn != nil {
		jclient.Active = false
		jclient.wswriteMutex.Lock()
		if err := jclient.wsconn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("write close message to wsconn failed: %s", err.Error()), false, false)
		}
		jclient.wswriteMutex.Unlock()
		jclient.wsconn.Close()
	}

	// 是否打印journalctl command stderr错误信息
	if _printstderr {
		readStderrTimeout, cancel := context.WithTimeout(JournaldCtx, jclient.timeout)
		defer cancel()
	OuterLoop:
		for {
			select {
			case <-readStderrTimeout.Done():
				global.ERManager.ErrorTransmit("journald", "warn", errors.New("read stderr timeout, kill journalctl process"), false, false)
				break OuterLoop
			case stderrLine, isOpen := <-jclient.errCh:
				if string(stderrLine) != "" {
					global.ERManager.ErrorTransmit("journald", "error", errors.New(string(stderrLine)), false, false)
				}
				if !isOpen {
					break OuterLoop
				}
			}
		}
	}

	if jclient.Jcmd != nil && jclient.Jcmd.Process != nil {
		if err := jclient.Jcmd.Process.Kill(); err != nil {
			global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("cannot kill journalctl process: %s", err), false, false)
		}
	}

	/*
		jclient.readFromStderr()
		jclient.readFromStdout()
	*/
	jclient.CancelF()
	/*
		jclient.readFromStderr()
		jclient.readFromStdout()
	*/
	jclient.wg.Wait()
	global.ERManager.ErrorTransmit("journald", "info", errors.Errorf("journald client:%s all goroutines done", jclient.ID), false, false)

	jclient.closeWriteMsgCh <- struct{}{}

	if _closechan {
		jclient.once.Do(func() {
			close(jclient.dataCh)
			close(jclient.errCh)
		})
	}

	time.Sleep(1000 * time.Millisecond)
	global.ERManager.ErrorTransmit("journald", "info", errors.Errorf("==========%-50s==========", fmt.Sprintf("journald client %s call close done", jclient.ID)), false, false)
}
