package comands

import (
	"fmt"
	model "github.com/suman9054/supersand/model"
	"os"

	"github.com/suman9054/supersand/store"
)

/* i have to creat init,status,config,pull,run,stop and delete commnad. */

// supersand command's

type comands struct {
	s          store.Storeapi
	configpath string
	logpath    string
}

type user struct {
	id       [16]byte
	username string
	password string
}

func newcomands(s store.Storeapi) *comands {
	return &comands{s: s}
}

// user related comands

func (c *comands) CreatNewuser(u user) error {
	if err := c.s.CreatUser(&store.UserObject{Id: u.id[:], Username: []byte(u.username), Password: []byte(u.password)}); err != nil {
		return err
	}
	return nil
}

func (c *comands) Loginuser(u user) error {
	err, userd := c.s.GetuserData(u.id)

	if err != nil {
		return err
	}

	if string(userd.Username) != u.username || string(userd.Password) != u.password {
		return fmt.Errorf("wrong password or username")
	}

	userd.Active = true
	if err := c.s.UpdateuserData(userd); err != nil {
		return err
	}
	return nil
}

func (c *comands) Deleteuser(id [16]byte) error {

	if err := c.s.DeleteUser(id); err != nil {
		return err
	}
	return nil
}

// session related commands
type session struct {
	id      [16]byte
	users   user
	sespath string
}

func (c *comands) CreatNewsession(se session) error {
	rootpath := fmt.Sprintf("~/.config/supersand/db/sessiondb/%s", se.sespath)
	if err := os.MkdirAll(rootpath, 0755); err != nil {
		return err
	}

	// todo i have to write the config file baesd of template
	if err := c.s.CreatNewSession(&store.Sessionobject{Id: se.id[:], UserId: se.users.id[:]}); err != nil {
		os.RemoveAll(rootpath)
		return err
	}
	c.configpath = rootpath
	return nil
}

func (c *comands) Deletsession(id [16]byte) error {
	if err := c.s.DeleteSession(id); err != nil {
		return err
	}
	// todo i have to also delet the secrets of this session
	return nil
}

func (c *comands) GetsessionData(id [16]byte) (error, *store.Sessionobject) {

	var def *store.Sessionobject

	err, data := c.s.GetsessionData(id)

	if err != nil {
		return err, def
	}

	return nil, data
}

// secrets comands

type secrect struct {
	id        [16]byte
	key       string
	val       string
	userid    [16]byte
	golbal    bool
	sessionid [16]byte
}

func (c *comands) CreatNewSecret(sec secrect) error {
	// todo i have to encode secret fields

	if sec.golbal != false {
		if err := c.s.CreatNewSecret(&store.SecrecObject{Id: sec.id[:], Key: []byte(sec.key), Value: []byte(sec.val), Userid: sec.userid[:], Scope: &model.Secret_SessionId{SessionId: sec.sessionid[:]}}); err != nil {
			return err
		}

	}
	if err := c.s.CreatNewSecret(&store.SecrecObject{Id: sec.id[:], Key: []byte(sec.key), Value: []byte(sec.val), Userid: sec.userid[:], Scope: &model.Secret_Global{}}); err != nil {
		return err
	}
	return nil
}

func (c *comands) DeletSecret(id [16]byte) error {
	if err := c.s.DeleteSecret(id); err != nil {
		return err
	}

	return nil
}
