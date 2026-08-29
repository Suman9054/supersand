package store

import (
	"fmt"

	rpc_api "github.com/suman9054/supersand/rpc"
	"google.golang.org/protobuf/proto"
)

type UserObject = rpc_api.UserObject


type store struct {
	Chash       stable[[16]byte, *UserObject]
  Wal         Walapi
}

type Storeapi interface{
 CreatUser(user *UserObject)error
 UpdateuserData(userdata *UserObject)error
 DeleteUser(key [16]byte) error
}

func Newstore() Storeapi {
	return &store{
		Chash:       NewChach(1024),
		Wal:         Initwal(),
		}
}


func(s *store) CreatUser(user *UserObject)error{
  
 userpayload,err:=proto.Marshal(user)
  if err != nil {
	 return fmt.Errorf("failed to parse user data error:%w",err)
 }
 
 var userkey [16]byte
 copy(userkey[:],user.Id)

 s.Chash.Set(userkey,user)
 err = s.Wal.SetRecord(TypeData,userpayload,userkey)
 
 if err != nil {
	 return fmt.Errorf("failed to creat user error:%w",err)
 }
	return nil
}

func (s *store) UpdateuserData(userdata *UserObject)error{
 userpayload,err:=proto.Marshal(userdata)
  if err != nil {
	 return fmt.Errorf("failed to parse user data error:%w",err)
 }
 
 var userkey [16]byte
 copy(userkey[:],userdata.Id)

 s.Chash.Set(userkey,userdata)
 err = s.Wal.SetRecord(TypeData,userpayload,userkey)
 
 if err != nil {
	 return fmt.Errorf("failed to creat user error:%w",err)
 }


return nil
}


func (s *store) DeleteUser(key [16]byte) error{
	if ok:= s.Chash.Remove(key);!ok {
		return fmt.Errorf("failed to Delete User")
	}
 var delpayload []byte

	if err:=s.Wal.SetRecord(TypeDelete,delpayload,key); err != nil {
		return fmt.Errorf("failed to Delete User error;%w",err)
	}
 return nil
}


func (s *store) GetuserData(key [16]byte)(error,*UserObject){
  payload ,ok:=s.Chash.Get(key)
  var defpayload *UserObject
	var user  UserObject
	if ok {
    return nil,payload
	}
 record,err:=s.Wal.GetRecord(key)
  if err != nil {
		return err,defpayload
	}

	err=proto.Unmarshal(record,&user)

	if err !=nil {
		return fmt.Errorf("failed to prash record error:%w",err),defpayload
	}
	return nil,&user
}
