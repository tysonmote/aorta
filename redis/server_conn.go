package redis

import (
	"github.com/stvp/resp"
	"net"
	"time"
)

type ServerConn struct {
	LastUsed time.Time
	address  string
	password string
	RESPConn
}

func NewServerConn(address, password string, timeout time.Duration) *ServerConn {
	server := &ServerConn{
		LastUsed: time.Now(),
		address:  address,
		password: password,
		RESPConn: RESPConn{
			timeout: timeout,
		},
	}
	return server
}

func (s *ServerConn) Do(command resp.Command) (response resp.Object, err error) {
	s.Lock()
	defer s.Unlock()
	s.LastUsed = time.Now()

	if s.conn == nil {
		err = s.dial()
		if err != nil {
			return nil, err
		}
	}

	return s.do(command)
}

func (s *ServerConn) Send(command resp.Command) (err error) {
	s.Lock()
	defer s.Unlock()
	s.LastUsed = time.Now()

	if s.conn == nil {
		err = s.dial()
		if err != nil {
			return err
		}
	}

	return s.write(command)
}

func (s *ServerConn) Address() string {
	return s.address
}

func (s *ServerConn) Password() string {
	return s.password
}

func (s *ServerConn) dial() (err error) {
	s.close()

	conn, err := net.DialTimeout("tcp", s.address, s.timeout)
	if err != nil {
		return wrapErr(err)
	}

	s.conn = conn
	s.reader = resp.NewReaderSize(s.conn, 8192)
	if len(s.password) > 0 {
		_, err = s.do(resp.NewCommand("AUTH", s.password))
		if err != nil {
			s.close()
			return err
		}
	}

	return nil
}

func (s *ServerConn) do(command resp.Command) (response resp.Object, err error) {
	err = s.write(command)
	if err != nil {
		return nil, err
	}
	response, err = s.readObject()
	if err == nil {
		if e, ok := response.(resp.Error); ok {
			err = e
		}
	}
	return response, err
}
