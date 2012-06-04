// Copyright (c) 2012, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Source code and contact info at http://github.com/soundcloud/visor

package visor

import (
	"fmt"
	"net"
	_path "path"
	"strconv"
	"strings"
	"time"
)

// An Instance represents a running process of a specific type.
// Instances belong to Revisions.
type Instance struct {
	Snapshot
	ProcType *ProcType    // ProcType the instance belongs to
	Addr     *net.TCPAddr // TCP address of the running instance
	State    State        // Current state of the instance
}

// InstanceInfo represents instance information as ids,
// when it's impractical to use the full Instance type.
type InstanceInfo struct {
	Name         string
	AppName      string
	RevisionName string
	ProcessName  ProcessName
	Host         string
	Port         int
	State        State
}

func (i InstanceInfo) AddrString() string {
	return i.Host + ":" + strconv.Itoa(i.Port)
}
func (i InstanceInfo) RevString() string {
	return i.AppName + "-" + i.RevisionName
}
func (i InstanceInfo) LogString() string {
	return fmt.Sprintf("%s (%s)", i.RevString(), i.AddrString())
}

const (
	InsStateInitial State = "initial"
	InsStateStarted       = "started"
	InsStateReady         = "ready"
	InsStateFailed        = "failed"
	InsStateDead          = "dead"
	InsStateExited        = "exited"
)

const INSTANCES_PATH = "instances"

// NewInstance creates and returns a new Instance object.
func NewInstance(ptype *ProcType, addr string, state State, snapshot Snapshot) (ins *Instance, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}

	ins = &Instance{Addr: tcpAddr, ProcType: ptype, State: state, Snapshot: snapshot}

	return
}

// FastForward advances the instance in time. It returns
// a new instance of Instance with the supplied revision.
func (i *Instance) FastForward(rev int64) *Instance {
	return i.Snapshot.fastForward(i, rev).(*Instance)
}

func (i *Instance) createSnapshot(rev int64) Snapshotable {
	return &Instance{Addr: i.Addr, State: i.State, ProcType: i.ProcType, Snapshot: Snapshot{rev, i.conn}}
}

// Register registers an instance with the registry.
func (i *Instance) Register() (instance *Instance, err error) {
	exists, _, err := i.conn.Exists(i.Path())
	if err != nil {
		return
	}
	if exists {
		return nil, ErrKeyConflict
	}
	if i.State != InsStateInitial {
		return nil, ErrInvalidState
	}

	rev, err := i.conn.SetMulti(i.Path(), map[string][]byte{
		"host":  []byte(i.Addr.IP.String()),
		"port":  []byte(strconv.Itoa(i.Addr.Port)),
		"state": []byte(i.State)}, i.Rev)
	if err != nil {
		return i, err
	}
	now := []byte(time.Now().UTC().String())

	rev, err = i.conn.Set(i.Path()+"/registered", i.Rev, now)
	if err != nil {
		return i, err
	}
	rev, err = i.conn.Set(i.ProcType.InstancePath(i.Id()), i.Rev, now)
	instance = i.FastForward(rev)

	return
}

// Unregister unregisters an instance with the registry.
func (i *Instance) Unregister() (err error) {
	rev := i.Rev

	err = i.conn.Del(i.ProcType.InstancePath(i.Id()), rev)
	if err != nil {
		return
	}
	err = i.conn.Del(i.Path(), rev)
	return
}

// UpdateState updates the instance's state file in
// the coordinator to the given value.
func (i *Instance) UpdateState(s State) (ins *Instance, err error) {
	newrev, err := i.conn.Set(i.Path()+"/state", i.Rev, []byte(s))
	if err != nil {
		return
	}
	ins = i.FastForward(newrev)
	ins.State = s

	return
}

// Path returns the instance's directory path in the registry.
func (i *Instance) Path() (path string) {
	return "/instances/" + i.Id()
}

func (i *Instance) Id() string {
	return strings.Replace(strings.Replace(i.Addr.String(), ".", "-", -1), ":", "-", -1)
}

func (i *Instance) String() string {
	return fmt.Sprintf("%#v", i)
}

// Instances returns returns an array of all registered instances.
func Instances(s Snapshot) (instances []*Instance, err error) {
	ptypes, err := ProcTypes(s)
	if err != nil {
		return
	}

	instances = []*Instance{}

	for i := range ptypes {
		iss, e := ProcTypeInstances(s, ptypes[i])
		if e != nil {
			return nil, e
		}
		instances = append(instances, iss...)
	}

	return
}

// ProcTypeInstances returns an array of all registered instances belonging
// to the given ProcType.
func ProcTypeInstances(s Snapshot, ptype *ProcType) (instances []*Instance, err error) {
	names, err := s.conn.Getdir(ptype.InstancesPath(), s.Rev)
	if err != nil {
		return
	}

	instances = make([]*Instance, len(names))

	for i := range names {
		path := _path.Join(INSTANCES_PATH, names[i])
		vals, e := s.conn.GetMulti(path, nil, s.Rev)

		if e != nil {
			return nil, e
		}

		addr := string(vals["host"]) + ":" + string(vals["port"])
		state := State(vals["state"])

		instances[i], err = NewInstance(ptype, addr, state, s)
		if err != nil {
			return
		}
	}

	return
}

// HostInstances returns an array of all registered instances belonging
// to the given host.
func HostInstances(s Snapshot, addr string) ([]Instance, error) {
	return nil, nil
}

// GetInstanceInfo returns an InstanceInfo from the given app, rev, proc and instance ids.
func GetInstanceInfo(s Snapshot, app string, rev string, proc string, ins string) (*InstanceInfo, error) {
	path := _path.Join(INSTANCES_PATH, ins)

	state, _, err := s.conn.Get(path+"/state", &s.Rev)
	host, _, err := s.conn.Get(path+"/host", &s.Rev)
	port, _, err := s.conn.Get(path+"/port", &s.Rev)

	iport, err := strconv.Atoi(string(port))

	instance := &InstanceInfo{
		Name:         ins,
		AppName:      app,
		RevisionName: rev,
		ProcessName:  ProcessName(proc),
		Host:         string(host),
		Port:         iport,
		State:        State(state),
	}
	return instance, err
}
