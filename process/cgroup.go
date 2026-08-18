package process

import (
	"github.com/containerd/cgroups/v3/cgroup2"
)

func (s *Process) setupCgroup() error {

	limit := int64(100 * 1024 * 1024)    // 100 mb is the max ram limit of any container
	highlimit := int64(80 * 1024 * 1024) // at 80 mb usage it will tregare an event
	swaplimit := int64(30 * 1024 * 1024) // this is limit for disk
	minlimit := int64(10 * 1024 * 1024)  // min ra limit is 10 mb

	cpuweight := uint64(100) // cpu priority is ideal for every contaner
	cpuqota := int64(10000)  // every container is allowd to use 0.1 core of cpu
	cpuperiod := uint64(100000)

	resouces := &cgroup2.Resources{
		CPU: &cgroup2.CPU{
			Weight: &cpuweight,
			Max:    cgroup2.NewCPUMax(&cpuqota, &cpuperiod),
		},

		Memory: &cgroup2.Memory{
			Swap: &swaplimit,
			Min:  &minlimit,
			Max:  &limit,
			Low:  &minlimit,
			High: &highlimit,
		},

		Pids: &cgroup2.Pids{
			Max: 30,
		},
	}
	manger, err := cgroup2.NewManager("/sys/fs/cgroup/", s.id.String(), resouces)
	if err != nil {
		manger.Delete()
		return err
	}

	if errr := manger.AddProc(uint64(s.pid)); errr != nil {
		manger.Delete()
		return errr
	}
	s.cgroupmanager = manger
	return nil
}

func (s *Process) CreatEventlistner() error {
	event, err := s.cgroupmanager.EventChan()
	if err != nil {
		return <-err
	}

	s.cgroupeventchan = <-event

	return nil
}
