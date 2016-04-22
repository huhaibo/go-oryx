// The MIT License (MIT)
//
// Copyright (c) 2013-2016 Oryx(ossrs)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package agent

import (
	"fmt"
	"net"
	"runtime/debug"

	"github.com/ossrs/go-oryx/core"
	"github.com/ossrs/go-oryx/protocol"
)

// the rtmp publish or play agent,
// to listen at RTMP(tcp://1935) and recv data from RTMP publisher or player,
// when identified the client type, redirect to the specified agent.
type Rtmp struct {
	ctx      core.Context
	endpoint string
	wc       core.WorkerContainer
	l        net.Listener
}

func NewRtmp(ctx core.Context, wc core.WorkerContainer) (agent core.OpenCloser) {
	v := &Rtmp{
		ctx: ctx,
		wc:  wc,
	}

	core.Conf.Subscribe(v)

	return v
}

// interface core.Agent
func (v *Rtmp) Open() (err error) {
	return v.applyListen(core.Conf)
}

func (v *Rtmp) Close() (err error) {
	core.Conf.Unsubscribe(v)
	return v.close()
}

func (v *Rtmp) close() (err error) {
	ctx := v.ctx

	if v.l == nil {
		return
	}

	if err = v.l.Close(); err != nil {
		core.Error.Println(ctx, "close rtmp listener failed. err is", err)
		return
	}
	v.l = nil

	core.Trace.Println(ctx, "close rtmp listen", v.endpoint, "ok")
	return
}

func (v *Rtmp) applyListen(c *core.Config) (err error) {
	ctx := v.ctx

	v.endpoint = fmt.Sprintf(":%v", c.Listen)

	ep := v.endpoint
	if v.l, err = net.Listen("tcp", ep); err != nil {
		core.Error.Println(ctx, "rtmp listen at", ep, "failed. err is", err)
		return
	}
	core.Trace.Println(ctx, "rtmp listen at", fmt.Sprintf("tcp://%v", c.Listen))

	// accept cycle
	v.wc.GFork("", func(wc core.WorkerContainer) {
		for v.l != nil {
			var c net.Conn
			if c, err = v.l.Accept(); err != nil {
				if v.l != nil {
					core.Warn.Println(ctx, "accept failed. err is", err)
				}
				return
			}

			v.serve(c)
		}
	})

	// should quit?
	v.wc.GFork("", func(wc core.WorkerContainer) {
		<-wc.QC()
		defer wc.Quit()
		_ = v.close()
	})

	return
}

func (v *Rtmp) serve(c net.Conn) {
	// use gfork to serve the connection.
	v.wc.GFork("", func(wc core.WorkerContainer) {
		// create new context for connection.
		ctx := core.NewContext()

		defer func() {
			if r := recover(); r != nil {
				if !core.IsNormalQuit(r) {
					core.Warn.Println(ctx, "rtmp ignore", r)
				}

				core.Error.Println(ctx, string(debug.Stack()))
			}
		}()
		defer func() {
			if err := c.Close(); err != nil {
				core.Info.Println(ctx, "ignore close failed. err is", err)
			}
		}()

		// for tcp connections.
		if c, ok := c.(*net.TCPConn); ok {
			// set TCP_NODELAY to false for performance issue.
			// TODO: FIXME: config it.
			// TODO: FIXME: refine for the realtime streaming.
			if err := c.SetNoDelay(false); err != nil {
				core.Error.Println(ctx, "set TCP_NODELAY failed. err is", err)
				return
			}
		}
		core.Trace.Println(ctx, "rtmp accept", c.RemoteAddr())

		conn := protocol.NewRtmpConnection(ctx, c, v.wc)
		defer conn.Close()

		if err := v.cycle(conn); err != nil {
			if !core.IsNormalQuit(err) && !IsControlError(err) {
				core.Warn.Println(ctx, "ignore error when cycle rtmp. err is", err)
			} else {
				core.Info.Println(ctx, "rtmp cycle ok.")
			}
			return
		}

		return
	})
}

func (v *Rtmp) cycle(conn *protocol.RtmpConnection) (err error) {
	ctx := conn.Ctx()
	r := conn.Req

	// handshake with client.
	if err = conn.Handshake(); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println(ctx, "rtmp handshake failed. err is", err)
		}
		return
	}
	core.Info.Println(ctx, "rtmp handshake ok.")

	// expoect connect app.
	if err = conn.ExpectConnectApp(r); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println(ctx, "rtmp connnect app failed. err is", err)
		}
		return
	}
	core.Info.Println(ctx, "rtmp connect app ok, tcUrl is", r.TcUrl)

	if err = conn.SetWindowAckSize(uint32(2.5 * 1000 * 1000)); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println(ctx, "rtmp set ack size failed. err is", err)
		}
		return
	}
	core.Info.Println(ctx, "set window ack ok.")

	if err = conn.SetPeerBandwidth(uint32(2.5*1000*1000), uint8(2)); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println(ctx, "rtmp set peer bandwidth failed. err is", err)
		}
		return
	}
	core.Info.Println(ctx, "set peer bandwidth ok.")

	// do bandwidth test if connect to the vhost which is for bandwidth check.
	// TODO: FIXME: support bandwidth check.

	// do token traverse before serve it.
	// @see https://github.com/ossrs/srs/pull/239
	// TODO: FIXME: support edge token tranverse.

	// set chunk size to larger.
	// set the chunk size before any larger response greater than 128,
	// to make OBS happy, @see https://github.com/ossrs/srs/issues/454
	if err = conn.SetChunkSize(core.Conf.ChunkSize); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println(ctx, "rtmp set chunk size failed. err is", err)
		}
		return
	}
	core.Info.Println(ctx, "set chunk size to", core.Conf.ChunkSize)

	// response the client connect ok and onBWDone.
	if err = conn.ResponseConnectApp(); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println(ctx, "response connect app failed. err is", err)
		}
		return
	}
	core.Info.Println(ctx, "response connect app ok.")
	if err = conn.OnBwDone(); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println(ctx, "response onBWDone failed. err is", err)
		}
		return
	}
	core.Info.Println(ctx, "onBWDone ok.")

	// identify the client, publish or play.
	if r.Type, r.Stream, r.Duration, err = conn.Identify(); err != nil {
		if !core.IsNormalQuit(err) {
			core.Error.Println(ctx, "identify client failed. err is", err)
		}
		return
	}
	core.Trace.Println(ctx, fmt.Sprintf(
		"client identified, type=%s, stream_name=%s, duration=%.2f",
		r.Type, r.Stream, r.Duration))

	// reparse the request by connect and play/publish.
	if err = r.Reparse(); err != nil {
		core.Error.Println(ctx, "reparse request failed. err is", err)
		return
	}
	if err = conn.OnUrlParsed(); err != nil {
		core.Error.Println(ctx, "notify url parsed failed. err is", err)
		return
	}

	// security check
	// TODO: FIXME: implements it.

	// set the TCP_NODELAY to false for high performance.
	// or set tot true for realtime stream.
	// TODO: FIXME: implements it.

	// check vhost.
	// for standard rtmp, the vhost specified in connectApp(tcUrl),
	// while some new client specifies the vhost in stream.
	// for example,
	//		connect("rtmp://vhost/app"), specified in tcUrl.
	//		connect("rtmp://ip/app?vhost=vhost"), specified in tcUrl.
	//		connect("rtmp://ip/app") && play("stream?vhost=vhost"), specified in stream.
	var vhost *core.Vhost
	if vhost, err = core.Conf.Vhost(r.Vhost); err != nil {
		core.Error.Println(ctx, "check vhost failed, vhost is", r.Vhost, "and err is", err)
		return
	} else if r.Vhost != vhost.Name {
		core.Trace.Println(ctx, "redirect vhost", r.Vhost, "to", vhost.Name)
		r.Vhost = vhost.Name
	}

	// set chunk_size on vhost level.
	// TODO: FIXME: support set chunk size.

	var agent core.Agent
	if conn.Req.Type.IsPlay() {
		if agent, err = Manager.NewRtmpPlayAgent(ctx, conn, v.wc); err != nil {
			core.Error.Println(ctx, "create play agent failed. err is", err)
			return
		}
	} else if conn.Req.Type.IsPublish() {
		if agent, err = Manager.NewRtmpPublishAgent(ctx, conn, v.wc); err != nil {
			core.Error.Println(ctx, "create publish agent failed. err is", err)
			return
		}
	} else {
		core.Warn.Println(ctx, "close invalid", conn.Req.Type, "client")
		return
	}

	// always create the agent when work done.
	defer func() {
		if err := agent.Close(); err != nil {
			core.Warn.Println(ctx, "ignore agent close failed. err is", err)
		}
	}()

	if err = agent.Pump(); err != nil {
		if !core.IsNormalQuit(err) && !IsControlError(err) {
			core.Warn.Println(ctx, "ignore rtmp agent work failed. err is", err)
		}
		return
	}

	return
}

// interface ReloadHandler
func (v *Rtmp) OnReloadGlobal(scope int, cc, pc *core.Config) (err error) {
	if scope != core.ReloadListen {
		return
	}

	if err = v.close(); err != nil {
		return
	}

	if err = v.applyListen(cc); err != nil {
		return
	}

	return
}

func (v *Rtmp) OnReloadVhost(vhost string, scope int, cc, pc *core.Config) (err error) {
	return
}

// rtmp play agent, to serve the player or edge.
type RtmpPlayAgent struct {
	ctx       core.Context
	conn      *protocol.RtmpConnection
	wc        core.WorkerContainer
	upstream  core.Agent
	jitter    *Jitter
	nbDropped uint32
}

func NewRtmpPlayAgent(ctx core.Context, conn *protocol.RtmpConnection, wc core.WorkerContainer) *RtmpPlayAgent {
	return &RtmpPlayAgent{
		ctx:    ctx,
		conn:   conn,
		wc:     wc,
		jitter: NewJitter(ctx),
	}
}

func (v *RtmpPlayAgent) Open() (err error) {
	ctx := v.ctx

	if err = v.conn.FlashStartPlay(); err != nil {
		core.Error.Println(ctx, "start play failed. err is", err)
		return
	}

	// check refer.
	// TODO: FIXME: implements it.

	return
}

func (v *RtmpPlayAgent) Close() (err error) {
	return v.UnTie(v.upstream)
}

func (v *RtmpPlayAgent) Pump() (err error) {
	return v.conn.Cycle(func(m *protocol.RtmpMessage) (err error) {
		// message from player.
		if m != nil {
			// TODO: FIXME: implements it.
			return
		}
		return
	})
}

func (v *RtmpPlayAgent) Write(m core.Message) (err error) {
	var ok bool
	var om *protocol.OryxRtmpMessage
	if om, ok = m.(*protocol.OryxRtmpMessage); !ok {
		return
	}

	// load the jitter algorithm.
	// TODO: FIXME: implements it.
	ag := Full

	// correct message timestamp.
	om.SetTimestamp(v.jitter.Correct(om.Timestamp(), ag))

	// cache message.
	if err = v.conn.CacheMessage(om.Payload()); err != nil {
		return
	}

	return
}

func (v *RtmpPlayAgent) Tie(sink core.Agent) (err error) {
	v.upstream = sink
	return sink.Flow(v)
}

func (v *RtmpPlayAgent) UnTie(sink core.Agent) (err error) {
	v.upstream = nil
	return sink.UnFlow(v)
}

func (v *RtmpPlayAgent) Flow(source core.Agent) (err error) {
	ctx := v.ctx

	core.Error.Println(ctx, "play agent not support flow.")
	return AgentNotSupportError
}

func (v *RtmpPlayAgent) UnFlow(source core.Agent) (err error) {
	ctx := v.ctx

	core.Error.Println(ctx, "play agent not support flow.")
	return AgentNotSupportError
}

func (v *RtmpPlayAgent) TiedSink() (sink core.Agent) {
	return v.upstream
}

// rtmp publish agent, to serve the FMLE or flash publisher/encoder.
type RtmpPublishAgent struct {
	ctx  core.Context
	conn *protocol.RtmpConnection
	wc   core.WorkerContainer
	flow core.Agent
}

func NewRtmpPublishAgent(ctx core.Context, conn *protocol.RtmpConnection, wc core.WorkerContainer) *RtmpPublishAgent {
	return &RtmpPublishAgent{
		ctx:  ctx,
		conn: conn,
		wc:   wc,
	}
}

func (v *RtmpPublishAgent) Open() (err error) {
	ctx := v.ctx

	if v.conn.Req.Type == protocol.RtmpFmlePublish {
		if err = v.conn.FmleStartPublish(); err != nil {
			core.Error.Println(ctx, "fmle start publish failed. err is", err)
			return
		}
	} else {
		if err = v.conn.FlashStartPublish(); err != nil {
			core.Error.Println(ctx, "flash start publish failed. err is", err)
			return
		}
	}

	// check refer.
	// TODO: FIXME: implements it.

	return
}

func (v *RtmpPublishAgent) Close() (err error) {
	if err = v.flow.UnTie(v); err != nil {
		return
	}

	// release publisher.
	// TODO: FIXME: implements it.

	return
}

func (v *RtmpPublishAgent) Pump() (err error) {
	ctx := v.ctx

	tm := protocol.PublishRecvTimeout

	err = v.conn.RecvMessage(tm, func(m *protocol.RtmpMessage) (err error) {
		if m.MessageType.IsCommand() {
			// for flash, any packet is republish.
			if v.conn.Req.Type == protocol.RtmpFlashPublish {
				// flash unpublish.
				// TODO: maybe need to support republish.
				core.Trace.Println(ctx, "flash publish finished.")
				return AgentControlRepublishError
			}

			// for fmle, drop others except the fmle start packet.
			var p protocol.RtmpPacket
			if p, err = v.conn.DecodeMessage(m); err != nil {
				return
			}

			if p, ok := p.(*protocol.RtmpFMLEStartPacket); ok {
				if err = v.conn.FmleUnpublish(p); err != nil {
					return
				}

				core.Trace.Println(ctx, "fmle publish finished.")
				return AgentControlRepublishError
			}

			core.Trace.Println(ctx, "fmle ignore AMF0/AMF3 command message.")
			return
		}

		var msg core.Message
		if msg, err = m.ToMessage(); err != nil {
			return
		}

		return v.flow.Write(msg)
	})

	// when republish, we expect more one more message
	// to ensure client got the unpublish response.
	// TODO: FIXME: support republish over same connection.
	if err == AgentControlRepublishError {
		return v.conn.RecvMessage(tm, func(m *protocol.RtmpMessage) error {
			core.Info.Println(ctx, "publish drop message", m)
			return AgentControlRepublishError
		})
	}
	return
}

func (v *RtmpPublishAgent) Write(m core.Message) (err error) {
	ctx := v.ctx

	core.Error.Println(ctx, "publish agent not support write message.")
	return AgentNotSupportError
}

func (v *RtmpPublishAgent) Tie(sink core.Agent) (err error) {
	ctx := v.ctx

	core.Error.Println(ctx, "publish agent has no upstream.")
	return AgentNotSupportError
}

func (v *RtmpPublishAgent) UnTie(sink core.Agent) (err error) {
	ctx := v.ctx

	core.Error.Println(ctx, "publish agent has no upstream.")
	return AgentNotSupportError
}

func (v *RtmpPublishAgent) Flow(source core.Agent) (err error) {
	v.flow = source
	return
}

func (v *RtmpPublishAgent) UnFlow(source core.Agent) (err error) {
	v.flow = nil
	return
}

func (v *RtmpPublishAgent) TiedSink() (sink core.Agent) {
	return
}
