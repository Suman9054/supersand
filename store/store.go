package store

import (
	"fmt"

	model "github.com/suman9054/supersand/model"
	"google.golang.org/protobuf/proto"
)

type UserObject = model.UserObjet
type Sessionobject = model.Session
type SecrecObject = model.Secret

type store struct {
	Chash stable[[16]byte, any]
	Wal   Walapi
}

type Storeapi interface {
	CreatUser(user *UserObject) error
	UpdateuserData(userdata *UserObject) error
	DeleteUser(key [16]byte) error
	GetuserData(key [16]byte) (error, *UserObject)
	CreatNewSession(session *Sessionobject) error
	DeleteSession(Id [16]byte) error
	GetsessionData(id [16]byte) (error, *Sessionobject)
	CreatNewSecret(secret *SecrecObject) error
	DeleteSecret(Id [16]byte) error
	GetsecretData(id [16]byte) (error, *SecrecObject)
}

func Newstore() Storeapi {
	return &store{
		Chash: newChach(1024),
		Wal:   initwal(),
	}
}

func (s *store) CreatUser(user *UserObject) error {

	userpayload, err := proto.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to parse user data error:%w", err)
	}

	var userkey [16]byte
	copy(userkey[:], user.Id)

	s.Chash.Set(userkey, user)
	err = s.Wal.SetRecord(TypeUserData, userpayload, userkey)

	if err != nil {
		return fmt.Errorf("failed to creat user error:%w", err)
	}
	return nil
}

func (s *store) UpdateuserData(userdata *UserObject) error {
	userpayload, err := proto.Marshal(userdata)
	if err != nil {
		return fmt.Errorf("failed to parse user data error:%w", err)
	}

	var userkey [16]byte
	copy(userkey[:], userdata.Id)

	s.Chash.Set(userkey, userdata)
	err = s.Wal.SetRecord(TypeUserData, userpayload, userkey)

	if err != nil {
		return fmt.Errorf("failed to creat user error:%w", err)
	}

	return nil
}

func (s *store) DeleteUser(id [16]byte) error {
	if ok := s.Chash.Remove(id); !ok {
		return fmt.Errorf("failed to Delete User")
	}
	var delpayload []byte

	if err := s.Wal.SetRecord(TypeDelete, delpayload, id); err != nil {
		return fmt.Errorf("failed to Delete User error;%w", err)
	}
	return nil
}

func (s *store) GetuserData(id [16]byte) (error, *UserObject) {
	payload, ok := s.Chash.Get(id)
	var defpayload UserObject
	var user UserObject
	if ok {
		return nil, payload.(*UserObject)
	}
	record, err := s.Wal.GetRecord(id)
	if err != nil {
		return err, &defpayload
	}

	err = proto.Unmarshal(record, &user)

	if err != nil {
		return fmt.Errorf("failed to prash record error:%w", err), &defpayload
	}
	return nil, &user
}

func (s *store) CreatNewSession(session *Sessionobject) error {
	sessionpayload, err := proto.Marshal(session)

	if err != nil {
		return err
	}

	var sessionid [16]byte

	copy(sessionid[:], session.Id)

	s.Chash.Set(sessionid, session)
	if err := s.Wal.SetRecord(TypeSessionsData, sessionpayload, sessionid); err != nil {
		return err
	}
	return nil
}

func (s *store) DeleteSession(Id [16]byte) error {
	if ok := s.Chash.Remove(Id); !ok {
		return fmt.Errorf("failed to Delete User")
	}
	var delpayload []byte

	if err := s.Wal.SetRecord(TypeDelete, delpayload, Id); err != nil {
		return fmt.Errorf("failed to Delete User error;%w", err)
	}
	return nil
}

func (s *store) GetsessionData(id [16]byte) (error, *Sessionobject) {
	payload, ok := s.Chash.Get(id)
	var defpayload Sessionobject
	var user Sessionobject
	if ok {
		return nil, payload.(*Sessionobject)
	}
	record, err := s.Wal.GetRecord(id)
	if err != nil {
		return err, &defpayload
	}

	err = proto.Unmarshal(record, &user)

	if err != nil {
		return fmt.Errorf("failed to prash record error:%w", err), &defpayload
	}
	return nil, &user
}

func (s *store) CreatNewSecret(secret *SecrecObject) error {
	sessionpayload, err := proto.Marshal(secret)

	if err != nil {
		return err
	}

	var sessionid [16]byte

	copy(sessionid[:], secret.Id)

	s.Chash.Set(sessionid, secret)
	if err := s.Wal.SetRecord(TypeSessionsData, sessionpayload, sessionid); err != nil {
		return err
	}
	return nil
}

func (s *store) DeleteSecret(Id [16]byte) error {
	if ok := s.Chash.Remove(Id); !ok {
		return fmt.Errorf("failed to Delete User")
	}
	var delpayload []byte

	if err := s.Wal.SetRecord(TypeDelete, delpayload, Id); err != nil {
		return fmt.Errorf("failed to Delete User error;%w", err)
	}
	return nil
}

func (s *store) GetsecretData(id [16]byte) (error, *SecrecObject) {
	payload, ok := s.Chash.Get(id)
	var defpayload SecrecObject
	var user SecrecObject
	if ok {
		return nil, payload.(*SecrecObject)
	}
	record, err := s.Wal.GetRecord(id)
	if err != nil {
		return err, &defpayload
	}

	err = proto.Unmarshal(record, &user)

	if err != nil {
		return fmt.Errorf("failed to prash record error:%w", err), &defpayload
	}
	return nil, &user
}
