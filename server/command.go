package server

import (
	"bytes"
	"fmt"
	. "github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go/hack"
)

type Handler interface {
	UseDB(dbName string) error
	HandleQuery(query string) (*Result, error)
	HandleFieldList(table string, fieldWildcard string) ([]*Field, error)
	HandleStmtPreprare(id uint32, sql string) (*Stmt, error)
	HandleStmtExecute(s *Stmt) (*Result, error)
}

func (c *Conn) HandleCommand() error {
	data, err := c.ReadPacket()
	if err != nil {
		c.Close()
		return err
	}

	v := c.dispatch(data)

	err = c.writeValue(v)

	c.ResetSequence()

	if err != nil {
		c.Close()
	}
	return err
}

func (c *Conn) dispatch(data []byte) interface{} {
	cmd := data[0]
	data = data[1:]

	switch cmd {
	case COM_QUIT:
		c.Close()
		return nil
	case COM_QUERY:
		if r, err := c.h.HandleQuery(hack.String(data)); err != nil {
			return err
		} else {
			return r
		}
	case COM_PING:
		return &Result{}
	case COM_INIT_DB:
		if err := c.h.UseDB(hack.String(data)); err != nil {
			return err
		} else {
			return &Result{}
		}
	case COM_FIELD_LIST:
		index := bytes.IndexByte(data, 0x00)
		table := hack.String(data[0:index])
		wildcard := hack.String(data[index+1:])

		if fs, err := c.h.HandleFieldList(table, wildcard); err != nil {
			return err
		} else {
			return fs
		}
	case COM_STMT_PREPARE:
		c.stmtID++
		if st, err := c.h.HandleStmtPreprare(c.stmtID, hack.String(data)); err != nil {
			return err
		} else {
			c.stmts[c.stmtID] = st
			return st
		}
	case COM_STMT_EXECUTE:
		if r, err := c.handleStmtExecute(data); err != nil {
			return err
		} else {
			return r
		}
	case COM_STMT_CLOSE:
		c.handleStmtClose(data)
		return nil
	case COM_STMT_SEND_LONG_DATA:
		c.handleStmtSendLongData(data)
		return nil
	case COM_STMT_RESET:
		if r, err := c.handleStmtReset(data); err != nil {
			return err
		} else {
			return r
		}
	default:
		msg := fmt.Sprintf("command %d is not supported now", cmd)
		return NewError(ER_UNKNOWN_ERROR, msg)
	}

	return fmt.Errorf("command %d is not handled correctly", cmd)
}
