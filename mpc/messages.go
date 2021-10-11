package main

import (
	"github.com/ldsec/lattigo/v2/bfv"
	"github.com/ldsec/lattigo/v2/rlwe"
)

type MsgSetup struct {
	Pk  *rlwe.PublicKey
	Rlk *rlwe.RelinearizationKey
}

type MsgReceiver struct {
	Cts []*bfv.Ciphertext
}

type MsgSender struct {
	Cts []*bfv.Ciphertext
}

func (c *Connection) ReadMsgSetup() (MsgSetup, error) {
	var msg MsgSetup
	err := c.Decode(&msg)
	return msg, err
}

func (c *Connection) ReadMsgReceiver() (MsgReceiver, error) {
	var msg MsgReceiver
	err := c.Decode(&msg)
	return msg, err
}

func (c *Connection) ReadMsgSender() (MsgSender, error) {
	var msg MsgSender
	err := c.Decode(&msg)
	return msg, err
}

/*
func (c *Connection) WriteMsgSetup(msg *MsgSetup) error {
	pk_data, err := msg.Pk.MarshalBinary()
	if err != nil {
		return err
	}
	if err := c.WriteSlice(pk_data); err != nil {
		return err
	}

	rlk_data, err := msg.Rlk.MarshalBinary()
	return c.WriteSlice(rlk_data)
}

func (c *Connection) ReadMsgSetup() (MsgSetup, error) {
	var msg MsgSetup

	msg.Pk = &rlwe.PublicKey{}
	msg.Rlk = &rlwe.RelinearizationKey{}

	pk_data, err := c.ReadSlice()
	if err != nil {
		return msg, err
	}

	if err := msg.Pk.UnmarshalBinary(pk_data); err != nil {
		return msg, err
	}

	rlk_data, err := c.ReadSlice()
	if err != nil {
		return msg, err
	}

	if err := msg.Rlk.UnmarshalBinary(rlk_data); err != nil {
		return msg, err
	}

	return msg, nil
}

func (c *Connection) ReadSlice() ([]byte, error) {
	data_size, err := c.ReadInt()
	if err != nil {
		return nil, err
	}
	data := make([]byte, data_size)
	err = c.Read(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Connection) WriteSlice(data []byte) error {
	if err := c.WriteInt(len(data)); err != nil {
		return err
	}
	return c.Write(data)
}

func (c *Connection) WriteMsgSender(msg *MsgSender) error {
	return c.WriteCTs(msg.Cts)
}

func (c *Connection) WriteMsgReceiver(msg *MsgReceiver) error {
	return c.WriteCTs(msg.Cts)
}

func (c *Connection) ReadMsgSender() (MsgSender, error) {
	var msg MsgSender
	cts, err := c.ReadCTs()
	msg.Cts = cts
	return msg, err
}

func (c *Connection) ReadMsgReceiver() (MsgReceiver, error) {
	var msg MsgReceiver
	cts, err := c.ReadCTs()
	msg.Cts = cts
	return msg, err
}

func (c *Connection) WriteInt(v int) error {
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], uint64(v))
	return c.Write(tmp[:])
}

func (c *Connection) ReadInt() (int, error) {
	var tmp [8]byte
	if err := c.Read(tmp[:]); err != nil {
		return 0, err
	}
	return int(binary.LittleEndian.Uint64(tmp[:])), nil
}

func (c *Connection) WriteCTs(cts []*bfv.Ciphertext) error {
	c.WriteInt(len(cts))
	for i := 0; i < len(cts); i++ {

		data, err := cts[i].MarshalBinary()
		if err != nil {
			return err
		}
		c.WriteSlice(data)
	}
	return nil
}

func (c *Connection) ReadCTs() ([]*bfv.Ciphertext, error) {
	size, err := c.ReadInt()
	if err != nil {
		return nil, err
	}

	cts := make([]*bfv.Ciphertext, size)
	for i := 0; i < len(cts); i++ {
		data, err := c.ReadSlice()
		if err != nil {
			return nil, err
		}

		cts[i] = &bfv.Ciphertext{}

		if err := cts[i].UnmarshalBinary(data); err != nil {
			return nil, err
		}
	}

	return cts, nil
}
*/
