package redis

import (
	"bufio"
	"io"
	"log"
	"os"
	"sync"
)

type Conn struct {
	RW io.ReadWriter
	Rd *bufio.Reader
}

func NewConn(rw io.ReadWriter) *Conn {
	return &Conn{
		RW: rw,
		Rd: bufio.NewReaderSize(rw, 1024),
	}
}

type ConnPool interface {
	Get() (*Conn, bool, error)
	Add(*Conn)
	Remove(*Conn)
	Len() int
}

//------------------------------------------------------------------------------

type MultiConnPool struct {
	Logger      *log.Logger
	cond        *sync.Cond
	conns       []*Conn
	OpenConn    OpenConnFunc
	CloseConn   CloseConnFunc
	cap, MaxCap int64
}

func NewMultiConnPool(openConn OpenConnFunc, closeConn CloseConnFunc, maxCap int64) *MultiConnPool {
	logger := log.New(
		os.Stdout,
		"redis.connpool: ",
		log.Ldate|log.Ltime|log.Lshortfile,
	)
	return &MultiConnPool{
		cond:      sync.NewCond(&sync.Mutex{}),
		Logger:    logger,
		conns:     make([]*Conn, 0),
		OpenConn:  openConn,
		CloseConn: closeConn,
		MaxCap:    maxCap,
	}
}

func (p *MultiConnPool) Get() (*Conn, bool, error) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	for len(p.conns) == 0 && p.cap >= p.MaxCap {
		p.cond.Wait()
	}

	if len(p.conns) == 0 {
		rw, err := p.OpenConn()
		if err != nil {
			return nil, false, err
		}

		p.cap++
		return NewConn(rw), true, nil
	}

	last := len(p.conns) - 1
	conn := p.conns[last]
	p.conns[last] = nil
	p.conns = p.conns[:last]

	return conn, false, nil
}

func (p *MultiConnPool) Add(conn *Conn) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	p.conns = append(p.conns, conn)
	p.cond.Signal()
}

func (p *MultiConnPool) Remove(conn *Conn) {
	p.cond.L.Lock()
	p.cap--
	p.cond.Signal()
	p.cond.L.Unlock()

	if p.CloseConn != nil && conn != nil {
		p.CloseConn(conn.RW)
	}
}

func (p *MultiConnPool) Len() int {
	return len(p.conns)
}

//------------------------------------------------------------------------------

type SingleConnPool struct {
	conn *Conn
}

func NewSingleConnPool(conn *Conn) *SingleConnPool {
	return &SingleConnPool{conn: conn}
}

func (p *SingleConnPool) Get() (*Conn, bool, error) {
	return p.conn, false, nil
}

func (p *SingleConnPool) Add(conn *Conn) {}

func (p *SingleConnPool) Remove(conn *Conn) {}

func (p *SingleConnPool) Len() int {
	return 1
}