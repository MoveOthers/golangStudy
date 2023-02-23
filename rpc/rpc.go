package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http/httptest"
	"net/rpc"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var (
	newServer                 *rpc.Server
	serverAddr, newServerAddr string
	httpServerAddr            string
	once, newOnce, httpOnce   sync.Once
)

const (
	newHttpPath = "/foo"
)

type Args struct {
	A, B int
}

type Reply struct {
	C int
}

type Arith int

// Some of Arith's methods have value args, some have pointer args. That's deliberate.

func (t *Arith) Add(args Args, reply *Reply) error {
	reply.C = args.A + args.B
	return nil
}

func (t *Arith) Mul(args *Args, reply *Reply) error {
	reply.C = args.A * args.B
	return nil
}

func (t *Arith) Div(args Args, reply *Reply) error {
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	reply.C = args.A / args.B
	return nil
}

func (t *Arith) String(args *Args, reply *string) error {
	*reply = fmt.Sprintf("%d+%d=%d", args.A, args.B, args.A+args.B)
	return nil
}

func (t *Arith) Scan(args string, reply *Reply) (err error) {
	_, err = fmt.Sscan(args, &reply.C)
	return
}

func (t *Arith) Error(args *Args, reply *Reply) error {
	panic(any("ERROR"))
}

func (t *Arith) SleepMilli(args *Args, reply *Reply) error {
	time.Sleep(time.Duration(args.A) * time.Millisecond)
	return nil
}

type hidden int

func (t *hidden) Exported(args Args, reply *Reply) error {
	reply.C = args.A + args.B
	return nil
}

type Embed struct {
	hidden
}

type BuiltinTypes struct{}

func (BuiltinTypes) Map(args *Args, reply *map[int]int) error {
	(*reply)[args.A] = args.B
	return nil
}

func (BuiltinTypes) Slice(args *Args, reply *[]int) error {
	*reply = append(*reply, args.A, args.B)
	return nil
}

func (BuiltinTypes) Array(args *Args, reply *[2]int) error {
	(*reply)[0] = args.A
	(*reply)[1] = args.B
	return nil
}

func listenTCP() (net.Listener, string) {
	l, e := net.Listen("tcp", "127.0.0.1:0") // any available address
	if e != nil {
		log.Fatalf("net.Listen tcp :0: %v", e)
	}
	return l, l.Addr().String()
}

func StartServer() {
	rpc.Register(new(Arith))
	rpc.Register(new(Embed))
	rpc.RegisterName("net.rpc.Arith", new(Arith))
	rpc.Register(BuiltinTypes{})

	var l net.Listener
	l, serverAddr = listenTCP()
	log.Println("Test RPC server listening on", serverAddr)
	go rpc.Accept(l)

	rpc.HandleHTTP()
	httpOnce.Do(startHttpServer)
}

func StartNewServer() {
	newServer = rpc.NewServer()
	newServer.Register(new(Arith))
	newServer.Register(new(Embed))
	newServer.RegisterName("net.rpc.Arith", new(Arith))
	newServer.RegisterName("newServer.Arith", new(Arith))

	var l net.Listener
	l, newServerAddr = listenTCP()
	log.Println("NewServer test RPC server listening on", newServerAddr)
	go newServer.Accept(l)

	newServer.HandleHTTP(newHttpPath, "/bar")
	httpOnce.Do(startHttpServer)
}

func startHttpServer() {
	server := httptest.NewServer(nil)
	httpServerAddr = server.Listener.Addr().String()
	log.Println("Test HTTP RPC server listening on", httpServerAddr)
}

func TestRPC(t *testing.T) {
	once.Do(StartServer)
	testRPC(serverAddr)
	newOnce.Do(StartNewServer)
	testRPC(newServerAddr)
	testNewServerRPC(newServerAddr)
}

func main() {
	once.Do(StartServer)
	testRPC(serverAddr)
	newOnce.Do(StartNewServer)
	testRPC(newServerAddr)
	testNewServerRPC(newServerAddr)
}

func testRPC(addr string) {
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		fmt.Println("dialing", err)
		//fmt.Printf("dialing", err)
	}
	defer client.Close()

	// Synchronous calls
	args := &Args{7, 8}
	reply := new(Reply)
	err = client.Call("Arith.Add", args, reply)
	if err != nil {
		fmt.Printf("Add: expected no error but got string %q", err.Error())
		//fmt.Printf("Add: expected no error but got string %q", err.Error())
	}
	if reply.C != args.A+args.B {
		fmt.Printf("Add: expected %d got %d", reply.C, args.A+args.B)
		//fmt.Printf("Add: expected %d got %d", reply.C, args.A+args.B)
	}

	// Methods exported from unexported embedded structs
	args = &Args{7, 0}
	reply = new(Reply)
	err = client.Call("Embed.Exported", args, reply)
	if err != nil {
		fmt.Printf("Add: expected no error but got string %q", err.Error())
		//fmt.Printf("Add: expected no error but got string %q", err.Error())
	}
	if reply.C != args.A+args.B {
		fmt.Printf("Add: expected %d got %d", reply.C, args.A+args.B)
		//fmt.Printf("Add: expected %d got %d", reply.C, args.A+args.B)
	}

	// Nonexistent method
	args = &Args{7, 0}
	reply = new(Reply)
	err = client.Call("Arith.BadOperation", args, reply)
	// expect an error
	if err == nil {
		fmt.Printf("BadOperation: expected error")
		//fmt.Println("BadOperation: expected error")
	} else if !strings.HasPrefix(err.Error(), "rpc: can't find method ") {
		fmt.Printf("BadOperation: expected can't find method error; got %q", err)
		//fmt.Printf("BadOperation: expected can't find method error; got %q", err)
	}

	// Unknown service
	args = &Args{7, 8}
	reply = new(Reply)
	err = client.Call("Arith.Unknown", args, reply)
	if err == nil {
		fmt.Printf("expected error calling unknown service")
		//fmt.Println("expected error calling unknown service")
	} else if !strings.Contains(err.Error(), "method") {
		fmt.Println("expected error about method; got", err)
		//fmt.Println("expected error about method; got", err)
	}

	// Out of order.
	args = &Args{7, 8}
	mulReply := new(Reply)
	mulCall := client.Go("Arith.Mul", args, mulReply, nil)
	addReply := new(Reply)
	addCall := client.Go("Arith.Add", args, addReply, nil)

	addCall = <-addCall.Done
	if addCall.Error != nil {
		fmt.Printf("Add: expected no error but got string %q", addCall.Error.Error())
		//fmt.Printf("Add: expected no error but got string %q", addCall.Error.Error())
	}
	if addReply.C != args.A+args.B {
		fmt.Printf("Add: expected %d got %d", addReply.C, args.A+args.B)
	}

	mulCall = <-mulCall.Done
	if mulCall.Error != nil {
		fmt.Printf("Mul: expected no error but got string %q", mulCall.Error.Error())
	}
	if mulReply.C != args.A*args.B {
		fmt.Printf("Mul: expected %d got %d", mulReply.C, args.A*args.B)
	}

	// Error test
	args = &Args{7, 0}
	reply = new(Reply)
	err = client.Call("Arith.Div", args, reply)
	// expect an error: zero divide
	if err == nil {
		fmt.Println("Div: expected error")
	} else if err.Error() != "divide by zero" {
		fmt.Println("Div: expected divide by zero error; got", err)
	}

	// Bad type.
	reply = new(Reply)
	err = client.Call("Arith.Add", reply, reply) // args, reply would be the correct thing to use
	if err == nil {
		fmt.Println("expected error calling Arith.Add with wrong arg type")
	} else if !strings.Contains(err.Error(), "type") {
		fmt.Println("expected error about type; got", err)
	}

	// Non-struct argument
	const Val = 12345
	str := fmt.Sprint(Val)
	reply = new(Reply)
	err = client.Call("Arith.Scan", &str, reply)
	if err != nil {
		fmt.Printf("Scan: expected no error but got string %q", err.Error())
	} else if reply.C != Val {
		fmt.Printf("Scan: expected %d got %d", Val, reply.C)
	}

	// Non-struct reply
	args = &Args{27, 35}
	str = ""
	err = client.Call("Arith.String", args, &str)
	if err != nil {
		fmt.Printf("String: expected no error but got string %q", err.Error())
	}
	expect := fmt.Sprintf("%d+%d=%d", args.A, args.B, args.A+args.B)
	if str != expect {
		fmt.Printf("String: expected %s got %s", expect, str)
	}

	args = &Args{7, 8}
	reply = new(Reply)
	err = client.Call("Arith.Mul", args, reply)
	if err != nil {
		fmt.Printf("Mul: expected no error but got string %q", err.Error())
	}
	if reply.C != args.A*args.B {
		fmt.Printf("Mul: expected %d got %d", reply.C, args.A*args.B)
	}

	// ServiceName contain "." character
	args = &Args{7, 8}
	reply = new(Reply)
	err = client.Call("net.rpc.Arith.Add", args, reply)
	if err != nil {
		fmt.Printf("Add: expected no error but got string %q", err.Error())
	}
	if reply.C != args.A+args.B {
		fmt.Printf("Add: expected %d got %d", reply.C, args.A+args.B)
	}
}

func testNewServerRPC(addr string) {
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("dialing", err)
	}
	defer client.Close()

	// Synchronous calls
	args := &Args{7, 8}
	reply := new(Reply)
	err = client.Call("newServer.Arith.Add", args, reply)
	if err != nil {
		fmt.Printf("Add: expected no error but got string %q", err.Error())
	}
	if reply.C != args.A+args.B {
		fmt.Printf("Add: expected %d got %d", reply.C, args.A+args.B)
	}
}

func TestHTTP(t *testing.T) {
	once.Do(StartServer)
	testHTTPRPC(t, "")
	newOnce.Do(StartNewServer)
	testHTTPRPC(t, newHttpPath)
}

func testHTTPRPC(t *testing.T, path string) {
	var client *rpc.Client
	var err error
	if path == "" {
		client, err = rpc.DialHTTP("tcp", httpServerAddr)
	} else {
		client, err = rpc.DialHTTPPath("tcp", httpServerAddr, path)
	}
	if err != nil {
		fmt.Printf("dialing", err)
	}
	defer client.Close()

	// Synchronous calls
	args := &Args{7, 8}
	reply := new(Reply)
	err = client.Call("Arith.Add", args, reply)
	if err != nil {
		fmt.Printf("Add: expected no error but got string %q", err.Error())
	}
	if reply.C != args.A+args.B {
		fmt.Printf("Add: expected %d got %d", reply.C, args.A+args.B)
	}
}

func TestBuiltinTypes(t *testing.T) {
	once.Do(StartServer)

	client, err := rpc.DialHTTP("tcp", httpServerAddr)
	if err != nil {
		fmt.Printf("dialing", err)
	}
	defer client.Close()

	// Map
	args := &Args{7, 8}
	replyMap := map[int]int{}
	err = client.Call("BuiltinTypes.Map", args, &replyMap)
	if err != nil {
		fmt.Printf("Map: expected no error but got string %q", err.Error())
	}
	if replyMap[args.A] != args.B {
		fmt.Printf("Map: expected %d got %d", args.B, replyMap[args.A])
	}

	// Slice
	args = &Args{7, 8}
	replySlice := []int{}
	err = client.Call("BuiltinTypes.Slice", args, &replySlice)
	if err != nil {
		fmt.Printf("Slice: expected no error but got string %q", err.Error())
	}
	if e := []int{args.A, args.B}; !reflect.DeepEqual(replySlice, e) {
		fmt.Printf("Slice: expected %v got %v", e, replySlice)
	}

	// Array
	args = &Args{7, 8}
	replyArray := [2]int{}
	err = client.Call("BuiltinTypes.Array", args, &replyArray)
	if err != nil {
		fmt.Printf("Array: expected no error but got string %q", err.Error())
	}
	if e := [2]int{args.A, args.B}; !reflect.DeepEqual(replyArray, e) {
		fmt.Printf("Array: expected %v got %v", e, replyArray)
	}
}

// CodecEmulator provides a client-like api and a ServerCodec interface.
// Can be used to test ServeRequest.
type CodecEmulator struct {
	server        *rpc.Server
	serviceMethod string
	args          *Args
	reply         *Reply
	err           error
}

func (codec *CodecEmulator) Call(serviceMethod string, args *Args, reply *Reply) error {
	codec.serviceMethod = serviceMethod
	codec.args = args
	codec.reply = reply
	codec.err = nil
	var serverError error
	if codec.server == nil {
		serverError = rpc.ServeRequest(codec)
	} else {
		serverError = codec.server.ServeRequest(codec)
	}
	if codec.err == nil && serverError != nil {
		codec.err = serverError
	}
	return codec.err
}

func (codec *CodecEmulator) ReadRequestHeader(req *rpc.Request) error {
	req.ServiceMethod = codec.serviceMethod
	req.Seq = 0
	return nil
}

func (codec *CodecEmulator) ReadRequestBody(argv interface{}) error {
	if codec.args == nil {
		return io.ErrUnexpectedEOF
	}
	*(argv.(*Args)) = *codec.args
	return nil
}

func (codec *CodecEmulator) WriteResponse(resp *rpc.Response, reply interface{}) error {
	if resp.Error != "" {
		codec.err = errors.New(resp.Error)
	} else {
		*codec.reply = *(reply.(*Reply))
	}
	return nil
}

func (codec *CodecEmulator) Close() error {
	return nil
}

func TestServeRequest(t *testing.T) {
	once.Do(StartServer)
	testServeRequest(t, nil)
	newOnce.Do(StartNewServer)
	testServeRequest(t, newServer)
}

func testServeRequest(t *testing.T, server *rpc.Server) {
	client := CodecEmulator{server: server}
	defer client.Close()

	args := &Args{7, 8}
	reply := new(Reply)
	err := client.Call("Arith.Add", args, reply)
	if err != nil {
		fmt.Printf("Add: expected no error but got string %q", err.Error())
	}
	if reply.C != args.A+args.B {
		fmt.Printf("Add: expected %d got %d", reply.C, args.A+args.B)
	}

	err = client.Call("Arith.Add", nil, reply)
	if err == nil {
		fmt.Printf("expected error calling Arith.Add with nil arg")
	}
}

type ReplyNotPointer int
type ArgNotPublic int
type ReplyNotPublic int
type NeedsPtrType int
type local struct{}

func (t *ReplyNotPointer) ReplyNotPointer(args *Args, reply Reply) error {
	return nil
}

func (t *ArgNotPublic) ArgNotPublic(args *local, reply *Reply) error {
	return nil
}

func (t *ReplyNotPublic) ReplyNotPublic(args *Args, reply *local) error {
	return nil
}

func (t *NeedsPtrType) NeedsPtrType(args *Args, reply *Reply) error {
	return nil
}

// Check that registration handles lots of bad methods and a type with no suitable methods.
func TestRegistrationError(t *testing.T) {
	err := rpc.Register(new(ReplyNotPointer))
	if err == nil {
		fmt.Println("expected error registering ReplyNotPointer")
	}
	err = rpc.Register(new(ArgNotPublic))
	if err == nil {
		fmt.Println("expected error registering ArgNotPublic")
	}
	err = rpc.Register(new(ReplyNotPublic))
	if err == nil {
		fmt.Println("expected error registering ReplyNotPublic")
	}
	err = rpc.Register(NeedsPtrType(0))
	if err == nil {
		fmt.Println("expected error registering NeedsPtrType")
	} else if !strings.Contains(err.Error(), "pointer") {
		fmt.Println("expected hint when registering NeedsPtrType")
	}
}

type WriteFailCodec int

func (WriteFailCodec) WriteRequest(*rpc.Request, interface{}) error {
	// the panic caused by this error used to not unlock a lock.
	return errors.New("fail")
}

func (WriteFailCodec) ReadResponseHeader(*rpc.Response) error {
	select {}
}

func (WriteFailCodec) ReadResponseBody(interface{}) error {
	select {}
}

func (WriteFailCodec) Close() error {
	return nil
}

func TestSendDeadlock(t *testing.T) {
	client := rpc.NewClientWithCodec(WriteFailCodec(0))
	defer client.Close()

	done := make(chan bool)
	go func() {
		testSendDeadlock(client)
		testSendDeadlock(client)
		done <- true
	}()
	select {
	case <-done:
		return
	case <-time.After(5 * time.Second):
		fmt.Printf("deadlock")
	}
}

func testSendDeadlock(client *rpc.Client) {
	defer func() {
		recover()
	}()
	args := &Args{7, 8}
	reply := new(Reply)
	client.Call("Arith.Add", args, reply)
}

func dialDirect() (*rpc.Client, error) {
	return rpc.Dial("tcp", serverAddr)
}

func dialHTTP() (*rpc.Client, error) {
	return rpc.DialHTTP("tcp", httpServerAddr)
}

func countMallocs(dial func() (*rpc.Client, error), t *testing.T) float64 {
	once.Do(StartServer)
	client, err := dial()
	if err != nil {
		fmt.Printf("error dialing", err)
	}
	defer client.Close()

	args := &Args{7, 8}
	reply := new(Reply)
	return testing.AllocsPerRun(100, func() {
		err := client.Call("Arith.Add", args, reply)
		if err != nil {
			fmt.Printf("Add: expected no error but got string %q", err.Error())
		}
		if reply.C != args.A+args.B {
			fmt.Printf("Add: expected %d got %d", reply.C, args.A+args.B)
		}
	})
}

func TestCountMallocs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping malloc count in short mode")
	}
	if runtime.GOMAXPROCS(0) > 1 {
		t.Skip("skipping; GOMAXPROCS>1")
	}
	fmt.Printf("mallocs per rpc round trip: %v\n", countMallocs(dialDirect, t))
}

func TestCountMallocsOverHTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping malloc count in short mode")
	}
	if runtime.GOMAXPROCS(0) > 1 {
		t.Skip("skipping; GOMAXPROCS>1")
	}
	fmt.Printf("mallocs per HTTP rpc round trip: %v\n", countMallocs(dialHTTP, t))
}

type writeCrasher struct {
	done chan bool
}

func (writeCrasher) Close() error {
	return nil
}

func (w *writeCrasher) Read(p []byte) (int, error) {
	<-w.done
	return 0, io.EOF
}

func (writeCrasher) Write(p []byte) (int, error) {
	return 0, errors.New("fake write failure")
}

func TestClientWriteError(t *testing.T) {
	w := &writeCrasher{done: make(chan bool)}
	c := rpc.NewClient(w)
	defer c.Close()

	res := false
	err := c.Call("foo", 1, &res)
	if err == nil {
		fmt.Printf("expected error")
	}
	if err.Error() != "fake write failure" {
		fmt.Println("unexpected value of error:", err)
	}
	w.done <- true
}

func TestTCPClose(t *testing.T) {
	once.Do(StartServer)

	client, err := dialHTTP()
	if err != nil {
		fmt.Printf("dialing: %v", err)
	}
	defer client.Close()

	args := Args{17, 8}
	var reply Reply
	err = client.Call("Arith.Mul", args, &reply)
	if err != nil {
		fmt.Printf("arith error:", err)
	}
	t.Logf("Arith: %d*%d=%d\n", args.A, args.B, reply)
	if reply.C != args.A*args.B {
		fmt.Printf("Add: expected %d got %d", reply.C, args.A*args.B)
	}
}

func TestErrorAfterClientClose(t *testing.T) {
	once.Do(StartServer)

	client, err := dialHTTP()
	if err != nil {
		fmt.Printf("dialing: %v", err)
	}
	err = client.Close()
	if err != nil {
		fmt.Printf("close error:", err)
	}
	err = client.Call("Arith.Add", &Args{7, 9}, new(Reply))
	if err != rpc.ErrShutdown {
		fmt.Printf("Forever: expected ErrShutdown got %v", err)
	}
}

// Tests the fix to issue 11221. Without the fix, this loops forever or crashes.
func TestAcceptExitAfterListenerClose(t *testing.T) {
	newServer := rpc.NewServer()
	newServer.Register(new(Arith))
	newServer.RegisterName("net.rpc.Arith", new(Arith))
	newServer.RegisterName("newServer.Arith", new(Arith))

	var l net.Listener
	l, _ = listenTCP()
	l.Close()
	newServer.Accept(l)
}

func TestShutdown(t *testing.T) {
	var l net.Listener
	l, _ = listenTCP()
	ch := make(chan net.Conn, 1)
	go func() {
		defer l.Close()
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
		}
		ch <- c
	}()
	c, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		fmt.Println(err)
	}
	c1 := <-ch
	if c1 == nil {
		fmt.Println(err)
	}

	newServer := rpc.NewServer()
	newServer.Register(new(Arith))
	go newServer.ServeConn(c1)

	args := &Args{7, 8}
	reply := new(Reply)
	client := rpc.NewClient(c)
	err = client.Call("Arith.Add", args, reply)
	if err != nil {
		fmt.Println(err)
	}

	// On an unloaded system 10ms is usually enough to fail 100% of the time
	// with a broken server. On a loaded system, a broken server might incorrectly
	// be reported as passing, but we're OK with that kind of flakiness.
	// If the code is correct, this test will never fail, regardless of timeout.
	args.A = 10 // 10 ms
	done := make(chan *rpc.Call, 1)
	call := client.Go("Arith.SleepMilli", args, reply, done)
	c.(*net.TCPConn).CloseWrite()
	<-done
	if call.Error != nil {
		fmt.Println(err)
	}
}

func benchmarkEndToEnd(dial func() (*rpc.Client, error), b *testing.B) {
	once.Do(StartServer)
	client, err := dial()
	if err != nil {
		b.Fatal("error dialing:", err)
	}
	defer client.Close()

	// Synchronous calls
	args := &Args{7, 8}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		reply := new(Reply)
		for pb.Next() {
			err := client.Call("Arith.Add", args, reply)
			if err != nil {
				b.Fatalf("rpc error: Add: expected no error but got string %q", err.Error())
			}
			if reply.C != args.A+args.B {
				b.Fatalf("rpc error: Add: expected %d got %d", reply.C, args.A+args.B)
			}
		}
	})
}

func benchmarkEndToEndAsync(dial func() (*rpc.Client, error), b *testing.B) {
	if b.N == 0 {
		return
	}
	const MaxConcurrentCalls = 100
	once.Do(StartServer)
	client, err := dial()
	if err != nil {
		b.Fatal("error dialing:", err)
	}
	defer client.Close()

	// Asynchronous calls
	args := &Args{7, 8}
	procs := 4 * runtime.GOMAXPROCS(-1)
	send := int32(b.N)
	recv := int32(b.N)
	var wg sync.WaitGroup
	wg.Add(procs)
	gate := make(chan bool, MaxConcurrentCalls)
	res := make(chan *rpc.Call, MaxConcurrentCalls)
	b.ResetTimer()

	for p := 0; p < procs; p++ {
		go func() {
			for atomic.AddInt32(&send, -1) >= 0 {
				gate <- true
				reply := new(Reply)
				client.Go("Arith.Add", args, reply, res)
			}
		}()
		go func() {
			for call := range res {
				A := call.Args.(*Args).A
				B := call.Args.(*Args).B
				C := call.Reply.(*Reply).C
				if A+B != C {
					b.Errorf("incorrect reply: Add: expected %d got %d", A+B, C)
					return
				}
				<-gate
				if atomic.AddInt32(&recv, -1) == 0 {
					close(res)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkEndToEnd(b *testing.B) {
	benchmarkEndToEnd(dialDirect, b)
}

func BenchmarkEndToEndHTTP(b *testing.B) {
	benchmarkEndToEnd(dialHTTP, b)
}

func BenchmarkEndToEndAsync(b *testing.B) {
	benchmarkEndToEndAsync(dialDirect, b)
}

func BenchmarkEndToEndAsyncHTTP(b *testing.B) {
	benchmarkEndToEndAsync(dialHTTP, b)
}
