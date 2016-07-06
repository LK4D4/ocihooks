package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const ifaceName = "ipv0"

type cfg struct {
	pid      int
	nsHandle netns.NsHandle

	parentIndex int
	addr        *netlink.Addr
	mode        netlink.IPVlanMode
}

var parent = flag.String("parent", "eth0", "name of parent interface")
var address = flag.String("address", "192.168.0.1/24", "address of ipvlan interface")
var mode = flag.String("mode", "l2", "ipvlan mode (l2 or l3)")

func validate() (*cfg, error) {
	var s struct {
		Pid int
	}

	if err := json.NewDecoder(os.Stdin).Decode(&s); err != nil {
		return nil, fmt.Errorf("decoding state: %v", err)
	}

	nsH, err := netns.GetFromPid(s.Pid)
	if err != nil {
		return nil, fmt.Errorf("get namespace from pid %d: %v", s.Pid, err)
	}

	var ipvlanMode netlink.IPVlanMode

	switch *mode {
	case "l2":
		ipvlanMode = netlink.IPVLAN_MODE_L2
	case "l3":
		ipvlanMode = netlink.IPVLAN_MODE_L3
	default:
		return nil, fmt.Errorf("invalid ipvlan mode: %s, expected l2 or l3", *mode)
	}

	parentLnk, err := netlink.LinkByName(*parent)
	if err != nil {
		return nil, fmt.Errorf("get parent link %s: %v", *parent, err)
	}

	addr, err := netlink.ParseAddr(*address)
	if err != nil {
		return nil, fmt.Errorf("parse address %s: %v", *address, err)
	}

	return &cfg{
		pid:      s.Pid,
		nsHandle: nsH,

		parentIndex: parentLnk.Attrs().Index,
		addr:        addr,
		mode:        ipvlanMode,
	}, nil
}

func loopbackUp() error {
	lnk, err := netlink.LinkByName("lo")
	if err != nil {
		return fmt.Errorf("get loopback: %v", err)
	}
	if err := netlink.LinkSetUp(lnk); err != nil {
		return fmt.Errorf("setting loopback up %v", err)
	}

	return nil
}

func setupNS(c *cfg) error {
	la := netlink.NewLinkAttrs()
	la.Name = ifaceName
	la.ParentIndex = c.parentIndex
	la.Namespace = netlink.NsPid(c.pid)

	if err := netlink.LinkAdd(&netlink.IPVlan{LinkAttrs: la, Mode: c.mode}); err != nil {
		return fmt.Errorf("add ipvlan link: %v", err)
	}

	if err := netns.Set(c.nsHandle); err != nil {
		return fmt.Errorf("set process namespace: %v", err)
	}

	if err := loopbackUp(); err != nil {
		return err
	}

	lnk, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return fmt.Errorf("get link by name in namespace %v", err)
	}
	if err := netlink.AddrAdd(lnk, c.addr); err != nil {
		return fmt.Errorf("add addr to link: %v", err)
	}

	if err := netlink.LinkSetUp(lnk); err != nil {
		return fmt.Errorf("setting link up %v", err)
	}
	return nil
}

func main() {
	runtime.LockOSThread()
	flag.Parse()
	cfg, err := validate()
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	if err := setupNS(cfg); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}
