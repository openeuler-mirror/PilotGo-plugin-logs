/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package proxy

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/public"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/global"
	"gitee.com/openeuler/PilotGo/sdk/utils/httputils"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var (
	DefaultUpgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	DefaultDialer = &websocket.Dialer{
		Proxy:            nil,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	err error
)

type WebsocketError struct {
	Code       int
	SrcConn    *websocket.Conn
	DstConn    *websocket.Conn
	SingleConn *websocket.Conn
	Text       string
}

const (
	WebsocketProxyReadError int = iota
	WebsocketProxyWriteError
	WebsocketProxySingleError
)

func (we *WebsocketError) Error() string {
	str := ""
	switch we.Code {
	case WebsocketProxyReadError:
		str = fmt.Sprintf("websocketerror(read): %s", we.Text)
	case WebsocketProxyWriteError:
		str = fmt.Sprintf("websocketerror(write): %s", we.Text)
	case WebsocketProxySingleError:
		str = fmt.Sprintf("websocketerror: %s", we.Text)
	}
	return str
}

type WebsocketForwardProxy struct {
	ID string

	Active bool

	targetURL string

	Upgrader *websocket.Upgrader

	Dialer *websocket.Dialer

	errChan    chan error
	errEndChan chan struct{}

	client_wsconn *websocket.Conn
	target_wsconn *websocket.Conn

	client_closemsg string
	target_closemsg string

	wg   sync.WaitGroup
	once sync.Once

	clientWriteMutex sync.Mutex
	clientReadMutex  sync.Mutex
	targetWriteMutex sync.Mutex
	targetReadMutex  sync.Mutex

	CancelCtx  context.Context
	CancelFunc context.CancelFunc

	request        *http.Request
	responseWriter http.ResponseWriter
}

func NewWebsocketForwardProxy() *WebsocketForwardProxy {
	ctx, cancel := context.WithCancel(WebsocketProxyCtx)
	return &WebsocketForwardProxy{
		Upgrader:        DefaultUpgrader,
		Dialer:          DefaultDialer,
		errChan:         make(chan error, 1),
		errEndChan:      make(chan struct{}, 1),
		client_closemsg: "",
		target_closemsg: "",
		CancelCtx:       ctx,
		CancelFunc:      cancel,
	}
}

// websocket代理：双端websocket连接转发
func (w *WebsocketForwardProxy) ServeHTTP(_w http.ResponseWriter, _r *http.Request) {
	global.ERManager.ErrorTransmit("webserver", "debug", errors.Errorf(
		"r.method: %v, r.proto: %v, r.host: %v, r.header: %+v, r.URL.scheme: %v, r.URL.host: %v, r.URL.path: %v",
		_r.Method, _r.Proto, _r.Host, _r.Header, _r.URL.Scheme, _r.URL.Host, _r.URL.Path), false, false)

	w.responseWriter = _w
	w.request = _r

	w.client_wsconn, err = w.Upgrader.Upgrade(_w, _r, nil)
	if err != nil {
		global.ERManager.ErrorTransmit("webserver", "error", errors.Errorf("failed to upgrade client connection to WebSocket: %s", err.Error()), false, false)
		http.Error(_w, fmt.Sprintf("failed to upgrade client connection to WebSocket: %s", err.Error()), http.StatusBadGateway)
		w.Close(true, false, false)
		return
	}

	if err := w.readMessageAgentAddr(); err != nil {
		global.ERManager.ErrorTransmit("webserver", "error", errors.Wrap(err, " "), false, false)
		w.client_closemsg = errors.Cause(err).Error()
		w.Close(true, false, false)
		return
	}

	if err := w.dialTarget(_r); err != nil {
		global.ERManager.ErrorTransmit("webserver", "error", errors.Wrap(err, " "), false, false)
		w.client_closemsg = errors.Cause(err).Error()
		w.Close(true, false, false)
		return
	}

	w.wg.Add(1)
	go w.writeMessage2Client(public.ConnectedMsg)
	go w.processError()
}

func (w *WebsocketForwardProxy) dialTarget(_r *http.Request) error {
	w.CancelCtx, w.CancelFunc = context.WithCancel(WebsocketProxyCtx)
	w.target_wsconn, _, err = w.Dialer.Dial(w.targetURL, w.targetDirector(_r))
	if err != nil {
		return errors.Errorf("dial to target WebSocket failed: %s", err.Error())
	}

	go w.transferMessages(w.client_wsconn, w.target_wsconn, true)
	go w.transferMessages(w.target_wsconn, w.client_wsconn, false)
	return nil
}

func (w *WebsocketForwardProxy) processError() {
	for {
		select {
		case <-w.errEndChan:
			global.ERManager.ErrorTransmit("webserver", "info", errors.New("processError done"), false, false)
			return
		case err := <-w.errChan:
			if err == nil {
				continue
			}
			global.ERManager.ErrorTransmit("webserver", "error", errors.New(err.Error()), false, false)
			wserr := err.(*WebsocketError)
			w.client_closemsg = wserr.Text
			w.target_closemsg = wserr.Text
		}
	}
}

func (w *WebsocketForwardProxy) targetDirector(_r *http.Request) http.Header {
	header := http.Header{}

	header.Set("Host", _r.Host)

	if clientIP, clientPort, err := net.SplitHostPort(_r.RemoteAddr); err == nil {

		if prior, ok := _r.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		header.Set("X-Forwarded-For", clientIP+":"+clientPort)
	}

	header.Set("X-Forwarded-Proto", "http")
	if _r.TLS != nil {
		header.Set("X-Forwarded-Proto", "https")
	}

	header.Set("clientId", w.ID)
	return header
}

func (w *WebsocketForwardProxy) ResponseDirector(_resp *http.Response, _header *http.Header) {
	if hdr := _resp.Header.Get("Sec-Websocket-Protocol"); hdr != "" {
		_header.Set("Sec-Websocket-Protocol", hdr)
	}
	if hdr := _resp.Header.Get("Set-Cookie"); hdr != "" {
		_header.Set("Set-Cookie", hdr)
	}
}

func (w *WebsocketForwardProxy) transferMessages(_srcConn, _dstConn *websocket.Conn, isC2T bool) {
	defer func() {
		global.ERManager.ErrorTransmit("webserver", "info", errors.Errorf("transferMessages goroutine done(%v->%v), forward: %v", _srcConn.RemoteAddr().String(), _dstConn.RemoteAddr().String(), isC2T), false, false)

		if r := recover(); r != nil {
			global.ERManager.ErrorTransmit("webserver", "warn", errors.Errorf("(transfermessages)send on closed channel: %+v", r), false, false)
		}
	}()

	for {
		select {
		case <-w.CancelCtx.Done():
			w.errChan <- &WebsocketError{
				Code:    WebsocketProxySingleError,
				SrcConn: _srcConn,
				DstConn: _dstConn,
				Text:    fmt.Sprintf("transferMessages goroutine exit(%v->%v), context canceled, forward: %v", _srcConn.RemoteAddr().String(), _dstConn.RemoteAddr().String(), isC2T),
			}
			return
		default:
			if isC2T {
				w.clientReadMutex.Lock()
			} else {
				w.targetReadMutex.Lock()
			}
			messageType, message, err := _srcConn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					w.errChan <- &WebsocketError{
						Code:    WebsocketProxyReadError,
						SrcConn: _srcConn,
						DstConn: _dstConn,
						Text:    fmt.Sprintf("websocket src conn %s closed(%v->%v): %s", _srcConn.RemoteAddr().String(), _srcConn.RemoteAddr().String(), _dstConn.RemoteAddr().String(), err.Error()),
					}
					// 远端关闭连接，释放资源
					w.Close(true, false, false)
					if isC2T {
						w.clientReadMutex.Unlock()
					} else {
						w.targetReadMutex.Unlock()
					}
					return
				}
				w.errChan <- &WebsocketError{
					Code:    WebsocketProxyReadError,
					SrcConn: _srcConn,
					DstConn: _dstConn,
					Text:    fmt.Sprintf("error while reading message(%v->%v, msgType: %d): %s, %s", _srcConn.RemoteAddr().String(), _dstConn.RemoteAddr().String(), messageType, err.Error(), message),
				}
				if isC2T {
					w.clientReadMutex.Unlock()
				} else {
					w.targetReadMutex.Unlock()
				}
				return
			}
			if isC2T {
				w.clientReadMutex.Unlock()
			} else {
				w.targetReadMutex.Unlock()
			}

			if isC2T {
				jmsg := &public.JMessage{}
				if err := json.Unmarshal(message, jmsg); message != nil && err != nil {
					global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("error while unmarshalling json options: %s, jmsg: %+v", err.Error(), string(message)), false, true)
					w.Close(true, false, false)
					return
				}
				switch jmsg.Type {
				case public.AgentAddrMsg:
					w.Close(false, false, true)
					ishttp, err := httputils.ServerIsHttp("http://" + jmsg.Data.(string))
					if err != nil {
						w.writeMessage2Client(public.DialFailedMsg)
						global.ERManager.ErrorTransmit("journald", "error", errors.Errorf("fail to detect remote http/https: %s", err.Error()), false, true)
						w.Close(true, false, false)
						return
					}
					if ishttp {
						w.targetURL = fmt.Sprintf("ws://%s/ws/entry", jmsg.Data.(string))
					}
					if !ishttp {
						w.targetURL = fmt.Sprintf("wss://%s/ws/entry", jmsg.Data.(string))
					}
					if err := w.dialTarget(w.request); err != nil {
						global.ERManager.ErrorTransmit("journald", "error", errors.Wrap(err, " "), false, true)
						w.Close(true, false, false)
					}
					w.wg.Add(1)
					go w.writeMessage2Client(public.ConnectedMsg)
					return
				}
			}

			if isC2T {
				w.targetWriteMutex.Lock()
			} else {
				w.clientWriteMutex.Lock()
			}
			if err := _dstConn.WriteMessage(messageType, message); err != nil {
				w.errChan <- &WebsocketError{
					Code:    WebsocketProxyWriteError,
					SrcConn: _srcConn,
					DstConn: _dstConn,
					Text:    fmt.Sprintf("error while writing message(%v->%v): %s", _srcConn.RemoteAddr().String(), _dstConn.RemoteAddr().String(), err.Error()),
				}
				if isC2T {
					w.targetWriteMutex.Lock()
				} else {
					w.clientWriteMutex.Lock()
				}
				return
			}
			if isC2T {
				w.targetWriteMutex.Unlock()
			} else {
				w.clientWriteMutex.Unlock()
			}
		}
	}
}

// 向客户端发送已与目标服务器建立连接消息
func (w *WebsocketForwardProxy) writeMessage2Client(_jmsg_type int) {
	defer w.wg.Done()
	defer global.ERManager.ErrorTransmit("webserver", "info", errors.Errorf("writeMessage2Client goroutine done, jmsg type: %d", _jmsg_type), false, false)

	for {
		select {
		case <-w.CancelCtx.Done():
			w.errChan <- &WebsocketError{
				Code:    WebsocketProxySingleError,
				DstConn: w.client_wsconn,
				SrcConn: w.target_wsconn,
				Text:    fmt.Sprintf("writeMessage2Client goroutine, jmsg type: %d, context canceled", _jmsg_type),
			}
			return
		default:
			jmsg := &public.JMessage{Type: _jmsg_type}
			jmsgBytes, err := json.Marshal(jmsg)
			if err != nil {
				w.errChan <- &WebsocketError{
					Code:       WebsocketProxySingleError,
					SingleConn: w.target_wsconn,
					Text:       fmt.Sprintf("error while marshalling json jmessage: %s", err.Error()),
				}
				return
			}

			w.clientWriteMutex.Lock()
			if err := w.client_wsconn.WriteMessage(websocket.TextMessage, jmsgBytes); err != nil {
				w.errChan <- &WebsocketError{
					Code:    WebsocketProxyWriteError,
					SrcConn: w.target_wsconn,
					DstConn: w.client_wsconn,
					Text:    fmt.Sprintf("error while writing message %d(%v->%v): %s", _jmsg_type, w.target_wsconn.RemoteAddr().String(), w.client_wsconn.RemoteAddr().String(), err.Error()),
				}
			}
			w.clientWriteMutex.Unlock()
			return
		}
	}
}

// 从客户端读取目标服务器地址
func (w *WebsocketForwardProxy) readMessageAgentAddr() error {
	w.clientReadMutex.Lock()
	_, jmsgBytes, err := w.client_wsconn.ReadMessage()
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
			w.clientReadMutex.Unlock()
			return errors.Errorf("error while reading message: %s", err.Error())
		}
		w.clientReadMutex.Unlock()
		return errors.Errorf("error while reading message: %s", err.Error())
	}
	w.clientReadMutex.Unlock()

	jmsg := &public.JMessage{}
	if err := json.Unmarshal(jmsgBytes, jmsg); jmsgBytes != nil && err != nil {
		return errors.Errorf("error while unmarshalling json jmessage: %s, %s", err.Error(), string(jmsgBytes))
	}
	// 确保与client建立websocket连接之后client发送的第一条消息为AgentAddrMsg
	if jmsg.Type != public.AgentAddrMsg {
		return errors.Errorf("the first message must be the agent addr: %d", jmsg.Type)
	}

	target_addr := jmsg.Data.(string)
	ishttp, err := httputils.ServerIsHttp("http://" + target_addr)
	if err != nil {
		w.writeMessage2Client(public.DialFailedMsg)
		return errors.Errorf("fail to detect remote http/https: %s", err.Error())
	}
	if ishttp {
		w.targetURL = fmt.Sprintf("ws://%s/ws/entry", target_addr)
	}
	if !ishttp {
		w.targetURL = fmt.Sprintf("wss://%s/ws/entry", target_addr)
	}
	return nil
}

func (w *WebsocketForwardProxy) Close(_close_A, _close_C, _close_T bool) {
	/*
		w.writeMessage2Client()
		w.transferMessages()
	*/
	w.CancelFunc()

	if _close_A {
		w.once.Do(func() {
			if w.client_wsconn != nil {
				w.clientWriteMutex.Lock()
				if err := w.client_wsconn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, w.client_closemsg)); err != nil {
					global.ERManager.ErrorTransmit("webserver", "error", errors.Errorf("write close message to client_wsconn error: %s", err.Error()), false, false)
				}
				w.clientWriteMutex.Unlock()
				w.client_wsconn.Close()
			}
			if w.target_wsconn != nil {
				w.targetWriteMutex.Lock()
				if err := w.target_wsconn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, w.target_closemsg)); err != nil {
					global.ERManager.ErrorTransmit("webserver", "error", errors.Errorf("write close message to target_wsconn error: %s", err.Error()), false, false)
				}
				w.targetWriteMutex.Unlock()
				w.target_wsconn.Close()
			}

			/*
				w.writeMessageConnectedWithTarget()
			*/
			w.wg.Wait()
			global.ERManager.ErrorTransmit("webserver", "info", errors.New("WebsocketForwardProxy all waitgroup goroutines done"), false, false)

			close(w.errEndChan)
			close(w.errChan)

			w.Active = false

			time.Sleep(100 * time.Millisecond)
		})
	}

	if _close_T {
		if w.target_wsconn != nil {
			w.targetWriteMutex.Lock()
			if err := w.target_wsconn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, w.target_closemsg)); err != nil {
				global.ERManager.ErrorTransmit("webserver", "error", errors.Errorf("write close message to target_wsconn error: %s", err.Error()), false, false)
			}
			w.targetWriteMutex.Unlock()
			w.target_wsconn.Close()
		}
	}
}
