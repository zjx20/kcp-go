package kcp

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/net/ipv4"
)

var baseport = uint32(10000)
var key = []byte("testkey")
var pass = pbkdf2.Key(key, []byte("testsalt"), 4096, 32, sha1.New)

func init() {
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()

	log.Println("beginning tests, encryption:salsa20, fec:10/3")
}

func dialEcho(port int) (*UDPSession, error) {
	//block, _ := NewNoneBlockCrypt(pass)
	//block, _ := NewSimpleXORBlockCrypt(pass)
	//block, _ := NewTEABlockCrypt(pass[:16])
	//block, _ := NewAESBlockCrypt(pass)
	block, _ := NewSalsa20BlockCrypt(pass)
	sess, err := DialWithOptions(fmt.Sprintf("127.0.0.1:%v", port), block, 10, 3)
	if err != nil {
		panic(err)
	}

	sess.SetStreamMode(true)
	sess.SetStreamMode(false)
	sess.SetStreamMode(true)
	sess.SetWindowSize(1024, 1024)
	sess.SetReadBuffer(16 * 1024 * 1024)
	sess.SetWriteBuffer(16 * 1024 * 1024)
	sess.SetStreamMode(true)
	sess.SetNoDelay(1, 10, 2, 1)
	sess.SetMtu(1400)
	sess.SetMtu(1600)
	sess.SetMtu(1400)
	sess.SetACKNoDelay(true)
	sess.SetACKNoDelay(false)
	sess.SetDeadline(time.Now().Add(time.Minute))
	return sess, err
}

func dialSink(port int) (*UDPSession, error) {
	sess, err := DialWithOptions(fmt.Sprintf("127.0.0.1:%v", port), nil, 0, 0)
	if err != nil {
		panic(err)
	}

	sess.SetStreamMode(true)
	sess.SetWindowSize(1024, 1024)
	sess.SetReadBuffer(16 * 1024 * 1024)
	sess.SetWriteBuffer(16 * 1024 * 1024)
	sess.SetStreamMode(true)
	sess.SetNoDelay(1, 10, 2, 1)
	sess.SetMtu(1400)
	sess.SetACKNoDelay(false)
	sess.SetDeadline(time.Now().Add(time.Minute))
	return sess, err
}

func dialTinyBufferEcho(port int) (*UDPSession, error) {
	//block, _ := NewNoneBlockCrypt(pass)
	//block, _ := NewSimpleXORBlockCrypt(pass)
	//block, _ := NewTEABlockCrypt(pass[:16])
	//block, _ := NewAESBlockCrypt(pass)
	block, _ := NewSalsa20BlockCrypt(pass)
	sess, err := DialWithOptions(fmt.Sprintf("127.0.0.1:%v", port), block, 10, 3)
	if err != nil {
		panic(err)
	}
	return sess, err
}

//////////////////////////
type listenFn func(port int) (net.Listener, error)

func listenEcho(port int) (net.Listener, error) {
	//block, _ := NewNoneBlockCrypt(pass)
	//block, _ := NewSimpleXORBlockCrypt(pass)
	//block, _ := NewTEABlockCrypt(pass[:16])
	//block, _ := NewAESBlockCrypt(pass)
	block, _ := NewSalsa20BlockCrypt(pass)
	return ListenWithOptions(fmt.Sprintf("127.0.0.1:%v", port), block, 10, 0)
}
func listenTinyBufferEcho(port int) (net.Listener, error) {
	//block, _ := NewNoneBlockCrypt(pass)
	//block, _ := NewSimpleXORBlockCrypt(pass)
	//block, _ := NewTEABlockCrypt(pass[:16])
	//block, _ := NewAESBlockCrypt(pass)
	block, _ := NewSalsa20BlockCrypt(pass)
	return ListenWithOptions(fmt.Sprintf("127.0.0.1:%v", port), block, 10, 3)
}

func listenNoEncryption(port int) (net.Listener, error) {
	return ListenWithOptions(fmt.Sprintf("127.0.0.1:%v", port), nil, 0, 0)
}

func server(
	port int,
	listen listenFn,
	handle func(*UDPSession),
) net.Listener {
	l, err := listen(port)
	if err != nil {
		panic(err)
	}

	go func() {
		kcplistener := l.(*Listener)
		kcplistener.SetReadBuffer(4 * 1024 * 1024)
		kcplistener.SetWriteBuffer(4 * 1024 * 1024)
		kcplistener.SetDSCP(46)
		for {
			s, err := l.Accept()
			if err != nil {
				return
			}

			// coverage test
			s.(*UDPSession).SetReadBuffer(4 * 1024 * 1024)
			s.(*UDPSession).SetWriteBuffer(4 * 1024 * 1024)
			go handle(s.(*UDPSession))
		}
	}()

	return l
}

func echoServer(port int) net.Listener {
	return server(port, listenEcho, handleEcho)
}

func sinkServer(port int) net.Listener {
	return server(port, listenNoEncryption, handleSink)
}

func tinyBufferEchoServer(port int) net.Listener {
	l, err := listenTinyBufferEcho(port)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			s, err := l.Accept()
			if err != nil {
				return
			}
			go handleTinyBufferEcho(s.(*UDPSession))
		}
	}()
	return l
}

///////////////////////////

func handleEcho(conn *UDPSession) {
	conn.SetStreamMode(true)
	conn.SetWindowSize(4096, 4096)
	conn.SetNoDelay(1, 10, 2, 1)
	conn.SetDSCP(46)
	conn.SetMtu(1400)
	conn.SetACKNoDelay(false)
	conn.SetReadDeadline(time.Now().Add(time.Hour))
	conn.SetWriteDeadline(time.Now().Add(time.Hour))
	buf := make([]byte, 65536)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		conn.Write(buf[:n])
	}
}

func handleSink(conn *UDPSession) {
	conn.SetStreamMode(true)
	conn.SetWindowSize(4096, 4096)
	conn.SetNoDelay(1, 10, 2, 1)
	conn.SetDSCP(46)
	conn.SetMtu(1400)
	conn.SetACKNoDelay(false)
	conn.SetReadDeadline(time.Now().Add(time.Hour))
	conn.SetWriteDeadline(time.Now().Add(time.Hour))
	buf := make([]byte, 65536)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			return
		}
	}
}

func handleTinyBufferEcho(conn *UDPSession) {
	conn.SetStreamMode(true)
	buf := make([]byte, 2)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		conn.Write(buf[:n])
	}
}

///////////////////////////

func TestTimeout(t *testing.T) {
	port := int(atomic.AddUint32(&baseport, 1))
	l := echoServer(port)
	defer l.Close()

	cli, err := dialEcho(port)
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 10)

	//timeout
	cli.SetDeadline(time.Now().Add(time.Second))
	<-time.After(2 * time.Second)
	n, err := cli.Read(buf)
	if n != 0 || err == nil {
		t.Fail()
	}
	cli.Close()
}

func TestSendRecv(t *testing.T) {
	port := int(atomic.AddUint32(&baseport, 1))
	l := echoServer(port)
	defer l.Close()

	cli, err := dialEcho(port)
	if err != nil {
		panic(err)
	}
	cli.SetWriteDelay(true)
	cli.SetDUP(1)
	const N = 100
	buf := make([]byte, 10)
	for i := 0; i < N; i++ {
		msg := fmt.Sprintf("hello%v", i)
		cli.Write([]byte(msg))
		if n, err := cli.Read(buf); err == nil {
			if string(buf[:n]) != msg {
				t.Fail()
			}
		} else {
			panic(err)
		}
	}
	cli.Close()
}

func TestSendVector(t *testing.T) {
	port := int(atomic.AddUint32(&baseport, 1))
	l := echoServer(port)
	defer l.Close()

	cli, err := dialEcho(port)
	if err != nil {
		panic(err)
	}
	cli.SetWriteDelay(false)
	const N = 100
	buf := make([]byte, 20)
	v := make([][]byte, 2)
	for i := 0; i < N; i++ {
		v[0] = []byte(fmt.Sprintf("hello%v", i))
		v[1] = []byte(fmt.Sprintf("world%v", i))
		msg := fmt.Sprintf("hello%vworld%v", i, i)
		cli.WriteBuffers(v)
		if n, err := cli.Read(buf); err == nil {
			if string(buf[:n]) != msg {
				t.Error(string(buf[:n]), msg)
			}
		} else {
			panic(err)
		}
	}
	cli.Close()
}

func TestTinyBufferReceiver(t *testing.T) {
	port := int(atomic.AddUint32(&baseport, 1))
	l := tinyBufferEchoServer(port)
	defer l.Close()

	cli, err := dialTinyBufferEcho(port)
	if err != nil {
		panic(err)
	}
	const N = 100
	snd := byte(0)
	fillBuffer := func(buf []byte) {
		for i := 0; i < len(buf); i++ {
			buf[i] = snd
			snd++
		}
	}

	rcv := byte(0)
	check := func(buf []byte) bool {
		for i := 0; i < len(buf); i++ {
			if buf[i] != rcv {
				return false
			}
			rcv++
		}
		return true
	}
	sndbuf := make([]byte, 7)
	rcvbuf := make([]byte, 7)
	for i := 0; i < N; i++ {
		fillBuffer(sndbuf)
		cli.Write(sndbuf)
		if n, err := io.ReadFull(cli, rcvbuf); err == nil {
			if !check(rcvbuf[:n]) {
				t.Fail()
			}
		} else {
			panic(err)
		}
	}
	cli.Close()
}

func TestClose(t *testing.T) {
	var n int
	var err error

	port := int(atomic.AddUint32(&baseport, 1))
	l := echoServer(port)
	defer l.Close()

	cli, err := dialEcho(port)
	if err != nil {
		panic(err)
	}

	// double close
	cli.Close()
	if cli.Close() == nil {
		t.Fatal("double close misbehavior")
	}

	// write after close
	buf := make([]byte, 10)
	n, err = cli.Write(buf)
	if n != 0 || err == nil {
		t.Fatal("write after close misbehavior")
	}

	// write, close, read, read
	cli, err = dialEcho(port)
	if err != nil {
		panic(err)
	}
	if n, err = cli.Write(buf); err != nil {
		t.Fatal("write misbehavior")
	}

	// wait until data arrival
	time.Sleep(2 * time.Second)
	// drain
	cli.Close()
	n, err = io.ReadFull(cli, buf)
	if err != nil {
		t.Fatal("closed conn drain bytes failed", err, n)
	}

	// after drain, read should return error
	n, err = cli.Read(buf)
	if n != 0 || err == nil {
		t.Fatal("write->close->drain->read misbehavior", err, n)
	}
	cli.Close()
}

func TestParallel1024CLIENT_64BMSG_64CNT(t *testing.T) {
	port := int(atomic.AddUint32(&baseport, 1))
	l := echoServer(port)
	defer l.Close()

	var wg sync.WaitGroup
	wg.Add(1024)
	for i := 0; i < 1024; i++ {
		go parallel_client(&wg, port)
	}
	wg.Wait()
}

func parallel_client(wg *sync.WaitGroup, port int) (err error) {
	cli, err := dialEcho(port)
	if err != nil {
		panic(err)
	}

	err = echo_tester(cli, 64, 64)
	cli.Close()
	wg.Done()
	return
}

func BenchmarkEchoSpeed4K(b *testing.B) {
	speedclient(b, 4096)
}

func BenchmarkEchoSpeed64K(b *testing.B) {
	speedclient(b, 65536)
}

func BenchmarkEchoSpeed512K(b *testing.B) {
	speedclient(b, 524288)
}

func BenchmarkEchoSpeed1M(b *testing.B) {
	speedclient(b, 1048576)
}

func speedclient(b *testing.B, nbytes int) {
	port := int(atomic.AddUint32(&baseport, 1))
	l := echoServer(port)
	defer l.Close()

	b.ReportAllocs()
	cli, err := dialEcho(port)
	if err != nil {
		panic(err)
	}

	if err := echo_tester(cli, nbytes, b.N); err != nil {
		b.Fail()
	}
	b.SetBytes(int64(nbytes))
	cli.Close()
}

func BenchmarkSinkSpeed4K(b *testing.B) {
	sinkclient(b, 4096)
}

func BenchmarkSinkSpeed64K(b *testing.B) {
	sinkclient(b, 65536)
}

func BenchmarkSinkSpeed256K(b *testing.B) {
	sinkclient(b, 524288)
}

func BenchmarkSinkSpeed1M(b *testing.B) {
	sinkclient(b, 1048576)
}

func sinkclient(b *testing.B, nbytes int) {
	port := int(atomic.AddUint32(&baseport, 1))
	l := sinkServer(port)
	defer l.Close()

	b.ReportAllocs()
	cli, err := dialSink(port)
	if err != nil {
		panic(err)
	}

	sink_tester(cli, nbytes, b.N)
	b.SetBytes(int64(nbytes))
	cli.Close()
}

func echo_tester(cli net.Conn, msglen, msgcount int) error {
	buf := make([]byte, msglen)
	for i := 0; i < msgcount; i++ {
		// send packet
		if _, err := cli.Write(buf); err != nil {
			return err
		}

		// receive packet
		nrecv := 0
		for {
			n, err := cli.Read(buf)
			if err != nil {
				return err
			} else {
				nrecv += n
				if nrecv == msglen {
					break
				}
			}
		}
	}
	return nil
}

func sink_tester(cli *UDPSession, msglen, msgcount int) error {
	// sender
	buf := make([]byte, msglen)
	for i := 0; i < msgcount; i++ {
		if _, err := cli.Write(buf); err != nil {
			return err
		}
	}
	return nil
}

func TestSNMP(t *testing.T) {
	t.Log(DefaultSnmp.Copy())
	t.Log(DefaultSnmp.Header())
	t.Log(DefaultSnmp.ToSlice())
	DefaultSnmp.Reset()
	t.Log(DefaultSnmp.ToSlice())
}

func TestListenerClose(t *testing.T) {
	port := int(atomic.AddUint32(&baseport, 1))
	l, err := ListenWithOptions(fmt.Sprintf("127.0.0.1:%v", port), nil, 10, 3)
	if err != nil {
		t.Fail()
	}
	l.SetReadDeadline(time.Now().Add(time.Second))
	l.SetWriteDeadline(time.Now().Add(time.Second))
	l.SetDeadline(time.Now().Add(time.Second))
	time.Sleep(2 * time.Second)
	if _, err := l.Accept(); err == nil {
		t.Fail()
	}

	l.Close()
	fakeaddr, _ := net.ResolveUDPAddr("udp6", "127.0.0.1:1111")
	if l.closeSession(fakeaddr) {
		t.Fail()
	}
}

// A wrapper for net.PacketConn that remembers when Close has been called.
type closedFlagPacketConn struct {
	net.PacketConn
	Closed bool
}

func (c *closedFlagPacketConn) Close() error {
	c.Closed = true
	return c.PacketConn.Close()
}

func newClosedFlagPacketConn(c net.PacketConn) *closedFlagPacketConn {
	return &closedFlagPacketConn{c, false}
}

// Listener should close a net.PacketConn that it created.
// https://github.com/xtaci/kcp-go/issues/165
func TestListenerOwnedPacketConn(t *testing.T) {
	// ListenWithOptions creates its own net.PacketConn.
	l, err := ListenWithOptions("127.0.0.1:0", nil, 0, 0)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	// Replace the internal net.PacketConn with one that remembers when it
	// has been closed.
	pconn := newClosedFlagPacketConn(l.conn)
	l.conn = pconn

	if pconn.Closed {
		t.Fatal("owned PacketConn closed before Listener.Close()")
	}

	err = l.Close()
	if err != nil {
		panic(err)
	}

	if !pconn.Closed {
		t.Fatal("owned PacketConn not closed after Listener.Close()")
	}
}

// Listener should not close a net.PacketConn that it did not create.
// https://github.com/xtaci/kcp-go/issues/165
func TestListenerNonOwnedPacketConn(t *testing.T) {
	// Create a net.PacketConn not owned by the Listener.
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer c.Close()
	// Make it remember when it has been closed.
	pconn := newClosedFlagPacketConn(c)

	l, err := ServeConn(nil, 0, 0, pconn)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	if pconn.Closed {
		t.Fatal("non-owned PacketConn closed before Listener.Close()")
	}

	err = l.Close()
	if err != nil {
		panic(err)
	}

	if pconn.Closed {
		t.Fatal("non-owned PacketConn closed after Listener.Close()")
	}
}

// UDPSession should close a net.PacketConn that it created.
// https://github.com/xtaci/kcp-go/issues/165
func TestUDPSessionOwnedPacketConn(t *testing.T) {
	l := sinkServer(0)
	defer l.Close()

	// DialWithOptions creates its own net.PacketConn.
	client, err := DialWithOptions(l.Addr().String(), nil, 0, 0)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	// Replace the internal net.PacketConn with one that remembers when it
	// has been closed.
	pconn := newClosedFlagPacketConn(client.conn)
	client.conn = pconn

	if pconn.Closed {
		t.Fatal("owned PacketConn closed before UDPSession.Close()")
	}

	err = client.Close()
	if err != nil {
		panic(err)
	}

	if !pconn.Closed {
		t.Fatal("owned PacketConn not closed after UDPSession.Close()")
	}
}

// UDPSession should not close a net.PacketConn that it did not create.
// https://github.com/xtaci/kcp-go/issues/165
func TestUDPSessionNonOwnedPacketConn(t *testing.T) {
	l := sinkServer(0)
	defer l.Close()

	// Create a net.PacketConn not owned by the UDPSession.
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer c.Close()
	// Make it remember when it has been closed.
	pconn := newClosedFlagPacketConn(c)

	client, err := NewConn2(l.Addr(), nil, 0, 0, pconn)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	if pconn.Closed {
		t.Fatal("non-owned PacketConn closed before UDPSession.Close()")
	}

	err = client.Close()
	if err != nil {
		panic(err)
	}

	if pconn.Closed {
		t.Fatal("non-owned PacketConn closed after UDPSession.Close()")
	}
}

type customBatchConn struct {
	*net.UDPConn
	calledWriteBatch bool
	calledReadBatch  bool

	disableWriteBatch     bool
	disableReadBatch      bool
	simulateWriteBatchErr bool
	simulateReadBatchErr  bool
}

func (c *customBatchConn) WriteBatch(ms []ipv4.Message, flags int) (int, error) {
	c.calledWriteBatch = true
	if c.disableWriteBatch {
		return 0, errors.New("unsupported")
	}
	if c.simulateWriteBatchErr {
		return 0, errors.New("unknown err")
	}
	n := 0
	for k := range ms {
		if _, err := c.WriteTo(ms[k].Buffers[0], ms[k].Addr); err == nil {
			n++
		} else {
			return n, err
		}
	}
	return n, nil
}

func (c *customBatchConn) ReadBatch(ms []ipv4.Message, flags int) (int, error) {
	c.calledReadBatch = true
	if c.disableReadBatch {
		return 0, errors.New("unsupported")
	}
	if c.simulateReadBatchErr {
		return 0, errors.New("unknown err")
	}
	succ := 0
	n, addr, err := c.ReadFrom(ms[0].Buffers[0])
	if err != nil {
		return succ, err
	}
	ms[0].N = n
	ms[0].Addr = addr
	succ++
	return succ, nil
}

func (c *customBatchConn) ReadBatchUnavailable(err error) bool {
	return err.Error() == "unsupported"
}

func (c *customBatchConn) WriteBatchUnavailable(err error) bool {
	return err.Error() == "unsupported"
}

func batchListenFn(opt func(pconn *customBatchConn)) listenFn {
	return func(port int) (net.Listener, error) {
		udpaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%v", port))
		if err != nil {
			return nil, err
		}
		conn, err := net.ListenUDP("udp", udpaddr)
		if err != nil {
			return nil, err
		}
		pconn := &customBatchConn{UDPConn: conn}
		opt(pconn)
		return serveConn(nil, 0, 0, pconn, true)
	}
}

func TestCustomBatchConn(t *testing.T) {
	listen := batchListenFn(func(pconn *customBatchConn) {})
	l := server(0, listen, handleEcho)
	defer l.Close()

	// Create a net.PacketConn not owned by the UDPSession.
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer c.Close()
	pconn := &customBatchConn{UDPConn: c.(*net.UDPConn)}
	defer pconn.Close()

	client, err := NewConn2(l.Addr(), nil, 0, 0, pconn)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	wBuf := []byte("hello")
	_, err = client.Write(wBuf)
	if err != nil {
		t.Fatalf("Write() should not fail, err: %v", err)
	}

	buf := make([]byte, 100)
	n, err := client.Read(buf)
	if err != nil {
		t.Fatalf("Read() should not fail, err: %v", err)
	}
	if n != len(wBuf) {
		t.Fatalf("should read %d bytes, actual n: %d", len(wBuf), n)
	}
	if string(wBuf) != string(buf[:n]) {
		t.Fatalf("read content should be '%s', actual: '%s'", string(wBuf), string(buf[:n]))
	}

	if !pconn.calledWriteBatch {
		t.Fatalf("expect to call WriteBatch()")
	}
	if !pconn.calledReadBatch {
		t.Fatalf("expect to call ReadBatch()")
	}
}

func TestCustomBatchConnFallback(t *testing.T) {
	// should fallback to defaultMonitor()
	listen := batchListenFn(func(pconn *customBatchConn) {
		pconn.disableReadBatch = true
		pconn.disableWriteBatch = true
	})
	l := server(0, listen, handleEcho)
	defer l.Close()

	// Create a net.PacketConn not owned by the UDPSession.
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer c.Close()
	pconn := &customBatchConn{UDPConn: c.(*net.UDPConn)}
	defer pconn.Close()

	// disabled batch ops, it should fallback to normal Read()/Write()
	pconn.disableReadBatch = true
	pconn.disableWriteBatch = true

	client, err := NewConn2(l.Addr(), nil, 0, 0, pconn)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	wBuf := []byte("hello")
	_, err = client.Write(wBuf)
	if err != nil {
		t.Fatalf("Write() should not fail, err: %v", err)
	}

	buf := make([]byte, 100)
	n, err := client.Read(buf)
	if err != nil {
		t.Fatalf("Read() should not fail, err: %v", err)
	}
	if n != len(wBuf) {
		t.Fatalf("should read %d bytes, actual n: %d", len(wBuf), n)
	}
	if string(wBuf) != string(buf[:n]) {
		t.Fatalf("read content should be '%s', actual: '%s'", string(wBuf), string(buf[:n]))
	}

	if !pconn.calledWriteBatch {
		t.Fatalf("expect to call WriteBatch()")
	}
	if !pconn.calledReadBatch {
		t.Fatalf("expect to call ReadBatch()")
	}
}

func TestBatchErrDetectorForRealErr(t *testing.T) {
	l := server(0, listenNoEncryption, handleEcho)
	defer l.Close()

	// Create a net.PacketConn not owned by the UDPSession.
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer c.Close()
	pconn := &customBatchConn{UDPConn: c.(*net.UDPConn)}
	defer pconn.Close()

	pconn.simulateReadBatchErr = true
	pconn.simulateWriteBatchErr = true

	client, err := NewConn2(l.Addr(), nil, 0, 0, pconn)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	client.SetWriteDelay(false)

	wBuf := []byte("hello")

	// no error for the first time
	_, err = client.Write(wBuf)
	if err != nil {
		t.Fatalf("Write() should not fail, err: %v", err)
	}

	// wait for the notification
	time.Sleep(2 * time.Duration(client.kcp.interval) * time.Millisecond)

	// error for the second time
	_, err = client.Write(wBuf)
	if err == nil {
		t.Fatalf("Write() should fail")
	}

	buf := make([]byte, 100)
	_, err = client.Read(buf)
	if err == nil {
		t.Fatalf("Read() should fail")
	}
}
