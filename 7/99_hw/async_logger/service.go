package __

import (
	"context"
	"encoding/json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"net"
	"strings"
	"sync"
	"time"
)

type bizServer struct {
	UnimplementedBizServer
}

func (b *bizServer) Check(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (b *bizServer) Add(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (b *bizServer) Test(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func NewBizServer() *bizServer {
	return &bizServer{}
}

type adminServer struct {
	UnimplementedAdminServer
	subsManager *SubEvent
}

func (a *adminServer) Logging(n *Nothing, ad Admin_LoggingServer) error {
	id, events := a.subsManager.NewSub()
	defer a.subsManager.RemoveSub(id)

	for event := range events {
		if err := ad.Send(event); err != nil {
			return err
		}
	}
	return nil
}

type StatCollector struct {
	stat Stat
}

func NewCollector() *StatCollector {
	return &StatCollector{
		Stat{
			ByMethod:   make(map[string]uint64),
			ByConsumer: make(map[string]uint64),
		},
	}
}

func (s *StatCollector) Update(event *Event) {
	s.stat.ByConsumer[event.Consumer] += 1
	s.stat.ByMethod[event.Method] += 1
}

func (s *StatCollector) reset() {
	s.stat = Stat{
		ByMethod:   make(map[string]uint64),
		ByConsumer: make(map[string]uint64),
	}
}

func (s *StatCollector) Collect() *Stat {
	newStat := &Stat{
		Timestamp:  time.Now().Unix(),
		ByMethod:   s.stat.ByMethod,
		ByConsumer: s.stat.ByConsumer,
	}
	s.reset()

	return newStat
}

func (a *adminServer) Statistics(s *StatInterval, ad Admin_StatisticsServer) error {
	id, events := a.subsManager.NewSub()
	defer a.subsManager.RemoveSub(id)
	statCollector := NewCollector()
	ticker := time.NewTicker(time.Duration(s.IntervalSeconds))
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-events:
			if ok {
				statCollector.Update(event)
			} else {
				return nil
			}
		case <-ticker.C:
			if err := ad.Send(statCollector.Collect()); err != nil {
				return err
			}
		}
	}
}

func NewAdminServer(subs *SubEvent) *adminServer {
	return &adminServer{subsManager: subs}
}

// []string{"url components"}
type ACLMethods [][]string

type ACL struct {
	allowed map[string]ACLMethods
}

func (acl *ACL) IsAllowed(consumer, method string) bool {
	splitted := strings.Split(method, "/")
	if methods, ok := acl.allowed[consumer]; ok {
		for _, rule := range methods {
			match := true
			for i := 0; i < len(rule); i++ {
				if i >= len(splitted) {
					match = false
					break
				}
				if rule[i] != "*" && rule[i] != splitted[i] {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

func NewACL(data string) (*ACL, error) {
	// map{key:[]values}
	aclTempMap := make(map[string][]string)
	if err := json.Unmarshal([]byte(data), &aclTempMap); err != nil {
		return nil, err
	}

	auth := &ACL{
		allowed: make(map[string]ACLMethods),
	}

	for consumer, methods := range aclTempMap {
		auth.allowed[consumer] = make([][]string, len(methods))
		for i, method := range methods {
			auth.allowed[consumer][i] = strings.Split(method, "/")
		}
	}

	return auth, nil
}

type SubEvent struct {
	UUID   int
	Events map[int]chan *Event
	Mtx    *sync.Mutex
}

func (s *SubEvent) NewSub() (int, chan *Event) {
	s.Mtx.Lock()
	defer s.Mtx.Unlock()

	s.UUID++
	s.Events[s.UUID] = make(chan *Event)
	return s.UUID, s.Events[s.UUID]
}

func (s *SubEvent) RemoveSub(id int) {
	s.Mtx.Lock()
	defer s.Mtx.Unlock()

	if sub, ok := s.Events[id]; ok {
		close(sub)
		delete(s.Events, id)
	}
}

func (s *SubEvent) RemoveAll() {
	s.Mtx.Lock()
	defer s.Mtx.Unlock()

	for key, _ := range s.Events {
		s.RemoveSub(key)
	}
}

// send this message for all subscribers
func (sub *SubEvent) Notify(event *Event) {
	sub.Mtx.Lock()
	defer sub.Mtx.Unlock()

	for _, subscriber := range sub.Events {
		subscriber <- event
	}
}

func NewEventSub() *SubEvent {
	return &SubEvent{
		Events: map[int]chan *Event{},
		Mtx:    &sync.Mutex{},
	}
}

type middleware struct {
	Options []grpc.ServerOption
	Auth    *ACL
	subs    *SubEvent
}

func (m *middleware) process(ctx context.Context, method string) error {
	md, _ := metadata.FromIncomingContext(ctx)
	str := strings.Join(md.Get("consumer"), "")
	host := ""
	if h, ok := peer.FromContext(ctx); ok {
		host = h.Addr.String()
	}
	// отправка сформированного лога(Event) всем подписчикам
	m.subs.Notify(
		&Event{
			Timestamp: time.Now().Unix(),
			Consumer:  str,
			Method:    method,
			Host:      host,
		},
	)

	// проверка (consumer, method) в ACL
	if !m.Auth.IsAllowed(str, method) {
		return status.Errorf(codes.Unauthenticated, "Inappropriate method for this consumer")
	}
	return nil
}

func (m *middleware) middlewareUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if err := m.process(ctx, info.FullMethod); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

func (m *middleware) middlewareStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := m.process(ss.Context(), info.FullMethod); err != nil {
		return err
	}
	return handler(srv, ss)
}

// перехватчик запросов
func NewMiddleware(acl *ACL, sub *SubEvent) (*middleware, error) {
	m := &middleware{Auth: acl, subs: sub}
	m.Options = []grpc.ServerOption{
		grpc.UnaryInterceptor(m.middlewareUnaryInterceptor),
		grpc.StreamInterceptor(m.middlewareStreamInterceptor),
	}
	return m, nil
}

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные
func StartMyMicroservice(ctx context.Context, addr, ACLData string) error {
	// создание таблицы ACL
	acl, err := NewACL(ACLData)
	if err != nil {
		return err
	}
	//net.Listener
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	// написание middleware , в который передаётся ACL таблица, который обрабатывает события до вызова
	sub := NewEventSub()
	mid, err := NewMiddleware(acl, sub)
	if mid == nil || err != nil {
		return err
	}

	// передача опций из middleware в параметры сервера

	// mid в параметрах
	server := grpc.NewServer(mid.Options...)

	RegisterAdminServer(server, NewAdminServer(sub))
	RegisterBizServer(server, NewBizServer())

	go server.Serve(lis)
	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()
	return nil
}
