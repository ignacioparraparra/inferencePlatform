package LoadBalancer

type Registry struct {
	addrs []string
	next  int
}

func (lb *Registry) RoundRobin() (addr string) {
	addr = lb.addrs[lb.next]
	lb.next++

	if lb.next == len(lb.addrs) {
		lb.next = 0
	}

	return addr
}

func (lb *Registry) AddAddress(addr string) {
	lb.addrs = append(lb.addrs, addr)
}
