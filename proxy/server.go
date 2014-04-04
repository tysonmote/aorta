package proxy

import (
	"fmt"
	"github.com/stvp/aorta/redis"
	"github.com/stvp/resp"
	"net"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	// Settings
	password      string
	clientTimeout time.Duration
	serverTimeout time.Duration

	bind     string
	listener net.Listener
	pool     *redis.ServerConnPool
	// cache *Cache
}

func NewServer(bind, password string, clientTimeout, serverTimeout time.Duration) *Server {
	return &Server{
		password:      password,
		clientTimeout: clientTimeout,
		serverTimeout: serverTimeout,
		bind:          bind,
		pool:          redis.NewServerConnPool(),
	}
}

func (s *Server) Listen() error {
	listener, err := net.Listen("tcp", s.bind)
	if err != nil {
		return err
	}
	s.listener = listener

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// TODO surface error to logger or something. Also, closing the
				// listener returns an error here, so we should ignore that.
				return
			}
			go s.handle(conn)
		}
	}()

	return nil
}

func (s *Server) Close() {
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *Server) handle(conn net.Conn) {
	client := redis.NewClientConn(conn, s.clientTimeout)
	defer client.Close()

	// State
	var authenticated bool
	var server *redis.ServerConn

	for {
		// Read command
		command, err := client.ReadCommand()
		if err == redis.ErrTimeout || err == redis.ErrConnClosed {
			return
		} else if err == resp.ErrSyntaxError {
			client.WriteError("ERR syntax error")
			return
		} else if err != nil {
			client.WriteError("aorta: " + err.Error())
			return
		}

		// Parse command
		args, err := command.Strings()
		if err != nil {
			client.WriteError("ERR syntax error")
			return
		}
		commandName := strings.ToUpper(args[0])

		if commandName == "QUIT" {
			return
		}

		// Require authentication
		if commandName == "AUTH" {
			if len(args) != 2 {
				client.WriteError("ERR wrong number of arguments for 'auth' command")
			} else if args[1] == s.password {
				authenticated = true
				client.Write(resp.OK)
			} else {
				authenticated = false
				client.WriteError("ERR invalid password")
			}
			continue
		}
		if !authenticated {
			// Redis returns the period even though thats inconsistent with all other
			// error messages. We include it here for correctness.
			client.WriteError("NOAUTH Authentication required.")
			return
		}

		// Require destination server
		if commandName == "PROXY" {
			server = nil
			if len(args) != 4 {
				client.WriteError("ERR wrong number of arguments for 'proxy' command")
				continue
			}
			address := fmt.Sprintf("%s:%s", args[1], args[2])
			server = s.pool.Get(address, args[3], s.serverTimeout)
			client.Write(resp.OK)
			continue
		}

		if server == nil {
			client.WriteError("aorta: proxy destination not set")
			continue
		}

		// Handle the command
		switch commandName {
		case "CACHED":
			if len(args) < 3 {
				client.WriteError("ERR wrong number of arguments for 'cached' command")
				return
			}
			secs, err := strconv.Atoi(args[1])
			if err != nil {
				client.WriteError("ERR syntax error")
			}
			panic(secs)
			// TODO cached get
		default:
			response, err := server.Do(command)
			if err != nil {
				client.WriteError(err.Error())
				continue
			}

			err = client.Write(response.Raw())
			if err != nil {
				return
			}
		}
	}
}