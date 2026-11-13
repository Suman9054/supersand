package comands

import "github.com/suman9054/supersand/store"

/* i have to creat init,status,config,pull,run,stop and delete commnad. */

// supersand command's

type comands struct {
	s *store.Storeapi
}

func (c *comands) Init() error {

	return nil
}

func (c *comands) Close() error {

	return nil
}

func (c *comands) Status() error {

	return nil
}

func Pull() error {

	return nil
}

func (c *comands) Edit() error {

	return nil
}

func (c *comands) Delete() error {

	return nil
}

func (c *comands) Logs() error {

	return nil
}

func (c *comands) Sessions() error {

	return nil
}

func (c *comands) EnterSession() error {

	return nil
}

func Exit() error {

	return nil
}
